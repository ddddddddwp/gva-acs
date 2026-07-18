package api

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ddddddddwp/gva-acs/server/model/common/response"
	tr069Middleware "github.com/ddddddddwp/gva-acs/server/plugin/tr069/middleware"
	artifactRequest "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/request"
	artifactResponse "github.com/ddddddddwp/gva-acs/server/plugin/tr069/model/response"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const artifactDownloadBufferSize = 64 * 1024

var artifactDownloadBuffers = sync.Pool{
	New: func() any { return make([]byte, artifactDownloadBufferSize) },
}

type ArtifactApi struct {
	transfers *service.TransferStore
	objects   service.ArtifactStore
}

func NewArtifactApi(transfers *service.TransferStore, objects service.ArtifactStore) *ArtifactApi {
	return &ArtifactApi{transfers: transfers, objects: objects}
}

// List returns artifact metadata only. Storage keys, source addresses and
// backend driver details are intentionally excluded from the response DTO.
func (a *ArtifactApi) List(c *gin.Context) {
	var in artifactRequest.ArtifactListRequest
	if err := c.ShouldBindQuery(&in); err != nil {
		response.FailWithMessage("查询参数错误", c)
		return
	}
	if in.Page <= 0 {
		in.Page = 1
	}
	if in.PageSize <= 0 {
		in.PageSize = 20
	} else if in.PageSize > 100 {
		in.PageSize = 100
	}
	createdFrom, err := parseArtifactListTime(in.CreatedFrom)
	if err != nil {
		response.FailWithMessage("开始时间格式错误", c)
		return
	}
	createdTo, err := parseArtifactListTime(in.CreatedTo)
	if err != nil {
		response.FailWithMessage("结束时间格式错误", c)
		return
	}
	if a == nil || a.transfers == nil {
		response.FailWithMessage("日志制品服务未初始化", c)
		return
	}
	items, total, err := a.transfers.ListArtifacts(c.Request.Context(), service.ArtifactListFilter{
		DeviceID: in.DeviceID, Channel: strings.ToUpper(strings.TrimSpace(in.Channel)),
		Status: strings.ToUpper(strings.TrimSpace(in.Status)), CreatedFrom: createdFrom, CreatedTo: createdTo,
		Offset: (in.Page - 1) * in.PageSize, Limit: in.PageSize,
	})
	if err != nil {
		response.FailWithMessage("查询日志文件失败", c)
		return
	}
	list := make([]artifactResponse.ArtifactSummary, 0, len(items))
	for _, item := range items {
		list = append(list, artifactResponse.ArtifactSummary{
			ArtifactID: item.ArtifactID, TaskID: item.TaskID, DeviceID: item.DeviceID,
			SerialNumber: item.SerialNumber, OUI: item.OUI, Channel: item.Channel,
			Source: item.Source, Status: item.Status, OriginalName: item.OriginalName,
			ContentType: item.ContentType, Size: item.Size, SHA256: item.SHA256,
			ReceivedAt: item.ReceivedAt, CreatedAt: item.CreatedAt,
		})
	}
	response.OkWithDetailed(response.PageResult{List: list, Total: total, Page: in.Page, PageSize: in.PageSize}, "获取成功", c)
}

// Download streams an available artifact through the protected GVA API.
func (a *ArtifactApi) Download(c *gin.Context) {
	artifactID := strings.TrimSpace(c.Param("artifactId"))
	if artifactID == "" || len(artifactID) > 64 || a == nil || a.transfers == nil {
		c.Status(http.StatusNotFound)
		return
	}
	artifact, err := a.transfers.GetAvailableArtifact(c.Request.Context(), artifactID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Status(http.StatusNotFound)
		} else {
			c.Status(http.StatusInternalServerError)
		}
		return
	}
	tr069Middleware.SetDownloadAuditMetadata(c, tr069Middleware.DownloadAuditMetadata{
		ArtifactID: artifact.ArtifactID,
		DeviceID:   artifact.DeviceID,
	})
	if a.objects == nil {
		c.Status(http.StatusServiceUnavailable)
		return
	}
	reader, object, err := a.objects.Open(c.Request.Context(), artifact.ObjectKey)
	if err != nil {
		if errors.Is(err, service.ErrArtifactNotFound) {
			c.Status(http.StatusNotFound)
		} else {
			c.Status(http.StatusInternalServerError)
		}
		return
	}
	defer reader.Close()

	filename := service.SanitizeArtifactOriginalName(artifact.OriginalName)
	if filename == "" {
		filename = artifact.ArtifactID + ".bin"
	}
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	if disposition == "" {
		disposition = `attachment; filename="artifact.bin"`
	}
	contentType := strings.TrimSpace(artifact.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	size := artifact.Size
	if size <= 0 {
		size = object.Size
	}
	c.Header("Content-Disposition", disposition)
	c.Header("Content-Type", contentType)
	if size >= 0 {
		c.Header("Content-Length", strconv.FormatInt(size, 10))
	}
	c.Status(http.StatusOK)
	buffer := artifactDownloadBuffers.Get().([]byte)
	defer artifactDownloadBuffers.Put(buffer)
	if _, err := io.CopyBuffer(c.Writer, reader, buffer); err != nil {
		_ = c.Error(err).SetType(gin.ErrorTypePrivate)
	}
}

func parseArtifactListTime(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			parsed = parsed.UTC()
			return &parsed, nil
		}
	}
	return nil, errors.New("invalid time")
}
