package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestEnsureTraceIDMapsExternalRequestHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(EnsureTraceID())
	router.POST("/", func(c *gin.Context) {
		if got := c.GetString("traceId"); got != "trace-gva-1" {
			t.Errorf("traceId = %q, want trace-gva-1", got)
		}
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("X-Request-ID", "trace-gva-1")
	router.ServeHTTP(httptest.NewRecorder(), request)
}

func TestEnsureTraceIDGeneratesUUIDWhenHeaderIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(EnsureTraceID())
	router.POST("/", func(c *gin.Context) {
		if got := c.GetString("traceId"); len(got) != 36 {
			t.Errorf("generated traceId = %q, want UUID", got)
		}
		c.Status(http.StatusNoContent)
	})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
}
