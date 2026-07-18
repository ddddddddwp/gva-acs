package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestEnsureTraceIDIgnoresExternalRequestHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(EnsureTraceID())
	generated := ""
	router.POST("/", func(c *gin.Context) {
		generated = c.GetString("traceId")
		c.Status(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set("X-Request-ID", "external-trace-id")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if len(generated) != 36 || generated == "external-trace-id" {
		t.Fatalf("traceId = %q, want independently generated UUID", generated)
	}
	if got := recorder.Header().Get("X-Request-ID"); got != "" {
		t.Fatalf("response X-Request-ID = %q, want empty", got)
	}
}

func TestEnsureTraceIDGeneratesUUIDWhenHeaderIsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(EnsureTraceID())
	generated := ""
	router.POST("/", func(c *gin.Context) {
		generated = c.GetString("traceId")
		if len(generated) != 36 {
			t.Errorf("generated traceId = %q, want UUID", generated)
		}
		c.Status(http.StatusNoContent)
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/", nil))
	if got := recorder.Header().Get("X-Request-ID"); got != "" {
		t.Fatalf("response X-Request-ID = %q, want empty", got)
	}
}

func TestEnsureTraceIDPreservesExistingContextValueWithoutResponseHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("traceId", "trace-from-context")
		c.Next()
	})
	router.Use(EnsureTraceID())
	router.POST("/", func(c *gin.Context) {
		if got := c.GetString("traceId"); got != "trace-from-context" {
			t.Fatalf("traceId = %q, want trace-from-context", got)
		}
		c.Status(http.StatusNoContent)
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/", nil))
	if got := recorder.Header().Get("X-Request-ID"); got != "" {
		t.Fatalf("response X-Request-ID = %q, want empty", got)
	}
}
