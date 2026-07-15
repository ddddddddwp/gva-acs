package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/gin-gonic/gin"
)

type trackingReadCloser struct {
	reader io.Reader
	reads  int
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
