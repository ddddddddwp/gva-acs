package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/model/system"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestDownloadAuditDoesNotBufferStreamedResponse(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(new(system.SysOperationRecord)); err != nil {
		t.Fatalf("migrate operation record: %v", err)
	}
	previousDB := global.GVA_DB
	global.GVA_DB = db
	t.Cleanup(func() { global.GVA_DB = previousDB })
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/download", DownloadAudit(), func(c *gin.Context) {
		SetDownloadAuditMetadata(c, DownloadAuditMetadata{FileID: 101, DeviceID: 42})
		c.Data(http.StatusOK, "application/octet-stream", bytes.Repeat([]byte("x"), 20<<20))
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/download", nil))
	if recorder.Body.Len() != 20<<20 {
		t.Fatalf("response bytes=%d", recorder.Body.Len())
	}
	var record system.SysOperationRecord
	if err := db.Last(&record).Error; err != nil {
		t.Fatalf("load operation record: %v", err)
	}
	if record.Resp != "" || len(record.Body) > 512 || record.Status != http.StatusOK {
		t.Fatalf("audit record buffered response: %#v", record)
	}
}
