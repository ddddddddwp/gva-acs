package middleware

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/model/system"
	systemRequest "github.com/ddddddddwp/gva-acs/server/model/system/request"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const downloadAuditMetadataKey = "tr069.download.audit.metadata"

// DownloadAuditMetadata is the deliberately small, non-secret audit payload
// recorded for an artifact download. The artifact bytes are never buffered.
type DownloadAuditMetadata struct {
	FileID   uint64 `json:"fileId,omitempty"`
	DeviceID uint   `json:"deviceId,omitempty"`
}

func SetDownloadAuditMetadata(c *gin.Context, metadata DownloadAuditMetadata) {
	c.Set(downloadAuditMetadataKey, metadata)
}

// DownloadAudit records only request metadata and deliberately leaves the
// response writer untouched so large artifact downloads remain streaming.
func DownloadAudit() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		body := "{}"
		if value, exists := c.Get(downloadAuditMetadataKey); exists {
			if metadata, ok := value.(DownloadAuditMetadata); ok {
				if encoded, err := json.Marshal(metadata); err == nil {
					body = string(encoded)
				}
			}
		}

		record := system.SysOperationRecord{
			Ip:           c.ClientIP(),
			Method:       c.Request.Method,
			Path:         c.Request.URL.Path,
			Status:       c.Writer.Status(),
			Latency:      time.Since(startedAt),
			Agent:        c.Request.UserAgent(),
			ErrorMessage: c.Errors.ByType(gin.ErrorTypePrivate).String(),
			Body:         body,
			Resp:         "",
			UserID:       downloadAuditUserID(c),
		}
		if global.GVA_DB == nil {
			return
		}
		if err := global.GVA_DB.Create(&record).Error; err != nil && global.GVA_LOG != nil {
			global.GVA_LOG.Error("create artifact download audit record", zap.Error(err))
		}
	}
}

func downloadAuditUserID(c *gin.Context) int {
	if value, exists := c.Get("claims"); exists {
		if claims, ok := value.(*systemRequest.CustomClaims); ok && claims != nil {
			return int(claims.BaseClaims.ID)
		}
	}
	id, _ := strconv.Atoi(c.GetHeader("x-user-id"))
	return id
}
