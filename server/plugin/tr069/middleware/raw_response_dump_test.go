package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/gin-gonic/gin"
)

func TestRawResponseDump_WritesInfoLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	previousRuntime := config.CurrentRuntime()
	tmp := t.TempDir()
	config.StoreRuntime(config.TR069Config{
		InfoLogEnable: true,
		InfoLogDir:    tmp,
	})
	t.Cleanup(func() {
		config.StoreRuntime(previousRuntime.Settings)
	})

	r := gin.New()
	r.Use(EnsureTraceID())
	r.Use(RawResponseDump(RawResponseDumpConfig{MaxBytes: 64 * 1024, DumpToConsole: false}))
	r.POST("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/xml", []byte("<soap>ok</soap>"))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("<in/>"))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", w.Code)
	}

	date := time.Now().Format("2006-01-02")
	p := filepath.Join(tmp, date, "tr069info.log")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read info log: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, "TR069 RAW RESPONSE BEGIN") {
		t.Fatalf("expected response dump written")
	}
	if !strings.Contains(s, "<soap>ok</soap>") {
		t.Fatalf("expected response body written")
	}
}

func TestRawResponseDumpSanitizesLogWithoutMutatingResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	previousRuntime := config.CurrentRuntime()
	tmp := t.TempDir()
	config.StoreRuntime(config.TR069Config{InfoLogEnable: true, InfoLogDir: tmp})
	t.Cleanup(func() { config.StoreRuntime(previousRuntime.Settings) })

	body := []byte(`<Envelope><ParameterValueStruct><Name>Device.ManagementServer.ConnectionRequestPassword</Name><Value>response-secret</Value></ParameterValueStruct><Password>download-secret</Password></Envelope>`)
	r := gin.New()
	r.Use(EnsureTraceID())
	r.Use(RawResponseDump(RawResponseDumpConfig{}))
	r.POST("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/xml", body)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader("<in/>")))
	if w.Body.String() != string(body) {
		t.Fatalf("wire response changed: got %q want %q", w.Body.Bytes(), body)
	}

	logPath := filepath.Join(tmp, time.Now().Format("2006-01-02"), "tr069info.log")
	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read info log: %v", err)
	}
	if strings.Contains(string(logged), "response-secret") || !strings.Contains(string(logged), "******") {
		t.Fatalf("response log was not sanitized: %s", logged)
	}
	if !strings.Contains(string(logged), "download-secret") {
		t.Fatalf("unrelated response password changed: %s", logged)
	}
}
