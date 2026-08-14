package middleware

import (
	"io"
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

type trackingReadCloser struct {
	reader io.Reader
	reads  int
}

func TestRawRequestDumpSanitizesLogWithoutMutatingHandlerBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousRuntime := config.CurrentRuntime()
	tmp := t.TempDir()
	config.StoreRuntime(config.TR069Config{InfoLogEnable: true, InfoLogDir: tmp})
	t.Cleanup(func() { config.StoreRuntime(previousRuntime.Settings) })

	body := `<Envelope><ParameterValueStruct><Name>Device.ManagementServer.ConnectionRequestPassword</Name><Value>request-secret</Value></ParameterValueStruct><Password>download-secret</Password></Envelope>`
	router := gin.New()
	router.Use(EnsureTraceID())
	router.Use(RawDump(RawDumpConfig{}))
	router.POST("/", func(c *gin.Context) {
		got, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Fatalf("read handler body: %v", err)
		}
		if string(got) != body {
			t.Fatalf("handler body changed: got %q want %q", got, body)
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	router.ServeHTTP(httptest.NewRecorder(), req)

	logPath := filepath.Join(tmp, time.Now().Format("2006-01-02"), "tr069info.log")
	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read info log: %v", err)
	}
	if strings.Contains(string(logged), "request-secret") || !strings.Contains(string(logged), "******") {
		t.Fatalf("request log was not sanitized: %s", logged)
	}
	if !strings.Contains(string(logged), "download-secret") {
		t.Fatalf("unrelated request password changed: %s", logged)
	}
}

func TestRawRequestDumpMalformedXMLFailsClosedButHandlerGetsOriginal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousRuntime := config.CurrentRuntime()
	tmp := t.TempDir()
	config.StoreRuntime(config.TR069Config{InfoLogEnable: true, InfoLogDir: tmp})
	t.Cleanup(func() { config.StoreRuntime(previousRuntime.Settings) })

	body := `<Envelope><ParameterValueStruct><Name>Device.ManagementServer.ConnectionRequestPassword</Name><Value>malformed-secret</Envelope>`
	router := gin.New()
	router.Use(RawDump(RawDumpConfig{}))
	router.POST("/", func(c *gin.Context) {
		got, _ := io.ReadAll(c.Request.Body)
		if string(got) != body {
			t.Fatalf("handler body changed: got %q want %q", got, body)
		}
		c.Status(http.StatusNoContent)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))

	logPath := filepath.Join(tmp, time.Now().Format("2006-01-02"), "tr069info.log")
	logged, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read info log: %v", err)
	}
	if strings.Contains(string(logged), "malformed-secret") {
		t.Fatalf("malformed request leaked to log: %s", logged)
	}
}

func (r *trackingReadCloser) Read(p []byte) (int, error) {
	r.reads++
	return r.reader.Read(p)
}

func (r *trackingReadCloser) Close() error { return nil }

func TestRawMiddlewareDisabledSkipsBodyCapture(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousRuntime := config.CurrentRuntime()
	config.StoreRuntime(config.TR069Config{})
	t.Cleanup(func() {
		config.StoreRuntime(previousRuntime.Settings)
	})

	body := &trackingReadCloser{reader: strings.NewReader("body-must-not-be-read")}
	router := gin.New()
	router.Use(RawDump(RawDumpConfig{
		MaxBytes:      1,
		RedactAuth:    true,
		RedactCookie:  true,
		PrintResponse: true,
		DumpToConsole: false,
	}))
	router.Use(RawResponseDump(RawResponseDumpConfig{MaxBytes: 1, DumpToConsole: false}))
	router.POST("/", func(c *gin.Context) {
		if body.reads != 0 {
			t.Errorf("disabled raw middleware read the request body %d times", body.reads)
		}
		if _, capturing := c.Writer.(*responseCaptureWriter); capturing {
			t.Error("disabled raw middleware wrapped the response writer")
		}
		c.String(http.StatusOK, "response-must-not-be-captured")
	})

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Body = body
	router.ServeHTTP(httptest.NewRecorder(), req)

	if body.reads != 0 {
		t.Fatalf("disabled raw middleware read the request body %d times", body.reads)
	}
}
