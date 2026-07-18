package handler

import (
	"context"
	"errors"
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
		if c.Request.ContentLength > channel.MaxFileSize {
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
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, channel.MaxFileSize+1)
		_, err = receiver.Receive(c.Request.Context(), service.ReceiveRequest{
			Device: device, Channel: channel.Name, Body: c.Request.Body, ContentLength: c.Request.ContentLength,
			OriginalName: originalFilename(c.Request.Header.Get("Content-Disposition")), ContentType: c.Request.Header.Get("Content-Type"), SourceIP: sourceIP,
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
			case errors.Is(err, context.DeadlineExceeded):
				c.Status(http.StatusRequestTimeout)
			default:
				c.Status(http.StatusInternalServerError)
			}
			return
		}
		c.Status(http.StatusNoContent)
	}
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
