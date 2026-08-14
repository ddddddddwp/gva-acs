package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/model/system"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func TestOperationRecordWithBodySanitizerPersistsCopyAndPreservesHandlerBody(t *testing.T) {
	db := newOperationRecordTestDB(t)
	body := `{"password":"operation-secret","other":"unchanged"}`
	router := gin.New()
	router.Use(OperationRecordWithBodySanitizer(func(in []byte) ([]byte, error) {
		if string(in) != body {
			t.Fatalf("sanitizer input = %q, want original body", in)
		}
		return []byte(`{"password":"******","other":"unchanged"}`), nil
	}))
	router.POST("/operation", func(c *gin.Context) {
		got, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != body {
			t.Fatalf("handler body changed: got %q want %q", got, body)
		}
		c.Status(http.StatusNoContent)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/operation", strings.NewReader(body)))

	var record system.SysOperationRecord
	if err := db.Order("id DESC").First(&record).Error; err != nil {
		t.Fatalf("load operation record: %v", err)
	}
	if record.Body != `{"password":"******","other":"unchanged"}` {
		t.Fatalf("recorded body = %q", record.Body)
	}
}

func TestOperationRecordWithBodySanitizerFailureHidesBodyAndPreservesHandlerBody(t *testing.T) {
	db := newOperationRecordTestDB(t)
	body := `{"password":"operation-secret"`
	router := gin.New()
	router.Use(OperationRecordWithBodySanitizer(func([]byte) ([]byte, error) {
		return nil, errors.New("malformed protected request")
	}))
	router.POST("/operation", func(c *gin.Context) {
		got, _ := io.ReadAll(c.Request.Body)
		if string(got) != body {
			t.Fatalf("handler body changed: got %q want %q", got, body)
		}
		c.Status(http.StatusNoContent)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/operation", strings.NewReader(body)))

	var record system.SysOperationRecord
	if err := db.Order("id DESC").First(&record).Error; err != nil {
		t.Fatalf("load operation record: %v", err)
	}
	if strings.Contains(record.Body, "operation-secret") || record.Body != operationRecordBodyRedacted {
		t.Fatalf("failed sanitizer recorded unsafe body %q", record.Body)
	}
}

func TestOperationRecordWithoutSanitizerRemainsBackwardCompatible(t *testing.T) {
	db := newOperationRecordTestDB(t)
	body := `{"ordinary":"body"}`
	router := gin.New()
	router.Use(OperationRecord())
	router.POST("/operation", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/operation", strings.NewReader(body)))

	var record system.SysOperationRecord
	if err := db.Order("id DESC").First(&record).Error; err != nil {
		t.Fatalf("load operation record: %v", err)
	}
	if record.Body != body {
		t.Fatalf("recorded body = %q, want %q", record.Body, body)
	}
}

func newOperationRecordTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(new(system.SysOperationRecord)); err != nil {
		t.Fatal(err)
	}
	previous := global.GVA_DB
	previousLog := global.GVA_LOG
	global.GVA_DB = db
	global.GVA_LOG = zap.NewNop()
	t.Cleanup(func() {
		global.GVA_DB = previous
		global.GVA_LOG = previousLog
	})
	return db
}
