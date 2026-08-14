package handler

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/middleware"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
)

const multipartEnvelopeAllowance int64 = 1 << 20

var errMultipartFileRequired = errors.New("multipart file field is required")

type FileRequestAuthenticator interface {
	Authenticate(*http.Request) (string, []string, error)
}

type UploadDeviceResolver interface {
	Resolve(context.Context, string, string) (service.UploadDeviceIdentity, error)
}

type TransferReceiveService interface {
	Receive(context.Context, service.ReceiveRequest) (model.Artifact, error)
}

type FileIngressChannel struct {
	Name                   string
	MaxFileSize            int64
	MaxConcurrent          int
	MaxConcurrentPerDevice int
	UploadTimeout          time.Duration
	RetentionDays          int
	StoragePrefix          string
	Driver                 string
}

type FileIngressChannelProvider interface {
	Channel(string) (FileIngressChannel, bool)
}

func NewFileIngressHandler(auth FileRequestAuthenticator, resolver UploadDeviceResolver, receiver TransferReceiveService, channels FileIngressChannelProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodPut && c.Request.Method != http.MethodPost {
			c.Header("Allow", "PUT, POST")
			c.Status(http.StatusMethodNotAllowed)
			return
		}
		if !supportedFileIngressSuffix(c.Request.Method, c.Param("filename")) {
			c.Status(http.StatusNotFound)
			return
		}
		channelName, challenges, err := auth.Authenticate(c.Request)
		if err != nil {
			for _, challenge := range challenges {
				c.Writer.Header().Add("WWW-Authenticate", challenge)
			}
			if errors.Is(err, middleware.ErrFileAuthentication) {
				c.Status(http.StatusUnauthorized)
			} else {
				c.Status(http.StatusInternalServerError)
			}
			return
		}
		channel, ok := channels.Channel(channelName)
		if !ok || !strings.EqualFold(channel.Name, channelName) {
			c.Status(http.StatusForbidden)
			return
		}
		multipartUpload := isMultipartUpload(c.Request)
		requestSizeLimit := channel.MaxFileSize
		if multipartUpload {
			requestSizeLimit += multipartEnvelopeAllowance
		}
		if c.Request.ContentLength > requestSizeLimit {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		sourceIP := remoteIPFromRequest(c.Request)
		device, err := resolver.Resolve(c.Request.Context(), sourceIP, channel.Name)
		if err != nil {
			if errors.Is(err, service.ErrUploadDeviceNotFound) || errors.Is(err, service.ErrUploadDeviceAmbiguous) {
				c.Status(http.StatusForbidden)
			} else {
				c.Status(http.StatusInternalServerError)
			}
			return
		}
		body, contentLength, originalName, contentType, err := ingressArtifactBody(c, channel.MaxFileSize, multipartUpload)
		if err != nil {
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				c.Status(http.StatusRequestEntityTooLarge)
			} else {
				c.Status(http.StatusBadRequest)
			}
			return
		}
		_, err = receiver.Receive(c.Request.Context(), service.ReceiveRequest{
			Device: device, Channel: channel.Name, Body: body, ContentLength: contentLength,
			OriginalName: originalName, ContentType: contentType, SourceIP: sourceIP,
			Driver: channel.Driver, StoragePrefix: channel.StoragePrefix, MaxFileSize: channel.MaxFileSize,
			UploadTimeout: channel.UploadTimeout, RetentionDays: channel.RetentionDays,
			MaxConcurrent: channel.MaxConcurrent, MaxConcurrentPerDevice: channel.MaxConcurrentPerDevice,
		})
		if err != nil {
			var maxBytesError *http.MaxBytesError
			switch {
			case errors.Is(err, service.ErrTransferFileTooLarge), errors.As(err, &maxBytesError):
				c.Status(http.StatusRequestEntityTooLarge)
			case errors.Is(err, service.ErrTransferBusy):
				c.Header("Retry-After", "5")
				c.Status(http.StatusServiceUnavailable)
			case errors.Is(err, service.ErrTransferContentConflict):
				c.Status(http.StatusConflict)
			case errors.Is(err, service.ErrDeviceDeleting):
				c.Status(http.StatusConflict)
			case errors.Is(err, context.DeadlineExceeded):
				c.Status(http.StatusRequestTimeout)
			default:
				c.Status(http.StatusInternalServerError)
			}
			return
		}
		c.Status(http.StatusCreated)
	}
}

func isMultipartUpload(request *http.Request) bool {
	if request == nil || request.Method != http.MethodPost {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	return err == nil && strings.EqualFold(mediaType, "multipart/form-data")
}

func supportedFileIngressSuffix(method, value string) bool {
	suffix := strings.TrimPrefix(value, "/")
	if suffix == "" {
		return true
	}
	return method == http.MethodPut && !strings.Contains(suffix, "/")
}

func ingressArtifactBody(c *gin.Context, maxFileSize int64, multipartUpload bool) (io.Reader, int64, string, string, error) {
	if !multipartUpload {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFileSize+1)
		originalName := originalFilename(c.Request.Header.Get("Content-Disposition"))
		if originalName == "" {
			originalName = routeFilename(c.Param("filename"))
		}
		return c.Request.Body, c.Request.ContentLength, originalName, c.Request.Header.Get("Content-Type"), nil
	}

	// Multipart is accepted only as a compatibility envelope for vendor CPEs.
	// MultipartReader keeps processing streaming and avoids ParseMultipartForm,
	// multipart.FileHeader, temporary files, and whole-body buffering.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFileSize+multipartEnvelopeAllowance)
	reader, err := c.Request.MultipartReader()
	if err != nil {
		return nil, 0, "", "", err
	}
	part, err := reader.NextPart()
	if errors.Is(err, io.EOF) {
		return nil, 0, "", "", errMultipartFileRequired
	}
	if err != nil {
		return nil, 0, "", "", err
	}
	if part.FormName() != "file" || strings.TrimSpace(part.FileName()) == "" {
		return nil, 0, "", "", errMultipartFileRequired
	}
	return part, -1, service.SanitizeArtifactOriginalName(part.FileName()), part.Header.Get("Content-Type"), nil
}

func routeFilename(value string) string {
	return service.SanitizeArtifactOriginalName(strings.TrimPrefix(value, "/"))
}

func originalFilename(contentDisposition string) string {
	if strings.TrimSpace(contentDisposition) == "" {
		return ""
	}
	_, parameters, err := mime.ParseMediaType(contentDisposition)
	if err != nil {
		return ""
	}
	return service.SanitizeArtifactOriginalName(parameters["filename"])
}

type RuntimeFileIngressChannelProvider struct{}

func (RuntimeFileIngressChannelProvider) Channel(name string) (FileIngressChannel, bool) {
	runtime := config.CurrentRuntime()
	for configuredName, channel := range runtime.FileIngress.Channels {
		if !channel.Enabled || !strings.EqualFold(configuredName, name) {
			continue
		}
		prefix := strings.Trim(runtime.Settings.FileIngress.ArtifactStore.Prefix, "/")
		if prefix == "" {
			prefix = strings.Trim(channel.StoragePrefix, "/")
		}
		prefix = path.Clean(prefix)
		return FileIngressChannel{
			Name: strings.ToUpper(configuredName), MaxFileSize: channel.MaxFileSize,
			MaxConcurrent: channel.MaxConcurrent, MaxConcurrentPerDevice: channel.MaxConcurrentPerDevice,
			UploadTimeout: channel.UploadTimeout, RetentionDays: channel.RetentionDays,
			StoragePrefix: prefix, Driver: runtime.Settings.FileIngress.ArtifactStore.Driver,
		}, true
	}
	return FileIngressChannel{}, false
}
