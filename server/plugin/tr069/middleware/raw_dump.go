package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/trace"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RawDumpConfig struct {
	MaxBytes      int
	RedactAuth    bool
	RedactCookie  bool
	PrintResponse bool
}

func EnsureRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		if v, ok := c.Get("requestId"); ok {
			if s, ok := v.(string); ok && s != "" {
				c.Next()
				return
			}
		}
		reqID := c.GetHeader("X-Request-Id")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		c.Set("requestId", reqID)
		c.Next()
	}
}

func RawDump(cfg RawDumpConfig) gin.HandlerFunc {
	maxBytes := cfg.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 64 * 1024
	}
	return func(c *gin.Context) {
		reqID, _ := c.Get("requestId")
		requestID, _ := reqID.(string)

		var reqBody []byte
		if c.Request != nil && c.Request.Body != nil {
			b, err := io.ReadAll(c.Request.Body)
			if err == nil {
				reqBody = b
				c.Request.Body = io.NopCloser(bytes.NewReader(b))
			}
		}

		reqDump := dumpRequest(c.Request, reqBody, requestID, cfg.RedactAuth, cfg.RedactCookie, maxBytes)
		_, _ = fmt.Fprintln(os.Stdout, reqDump)
		if requestID != "" {
			trace.Default.Add(requestID, trace.Entry{
				At:      time.Now(),
				Stage:   "raw.request",
				Message: truncateBytes(reqBody, maxBytes),
			})
		}

		var capture *responseCaptureWriter
		if cfg.PrintResponse {
			capture = newResponseCaptureWriter(c.Writer, maxBytes)
			c.Writer = capture
		}

		start := time.Now()
		c.Next()
		elapsed := time.Since(start)

		if capture != nil {
			respDump := dumpResponse(c.Writer.Status(), c.Writer.Header(), capture.body.Bytes(), requestID, elapsed, maxBytes)
			_, _ = fmt.Fprintln(os.Stdout, respDump)
			if requestID != "" {
				trace.Default.Add(requestID, trace.Entry{
					At:      time.Now(),
					Stage:   "raw.response",
					Message: truncateBytes(capture.body.Bytes(), maxBytes),
					Fields: map[string]string{
						"status": fmt.Sprintf("%d", c.Writer.Status()),
					},
				})
			}
		}
	}
}

func dumpRequest(r *http.Request, body []byte, requestID string, redactAuth, redactCookie bool, maxBytes int) string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("----- TR069 RAW REQUEST BEGIN -----\n")
	if requestID != "" {
		b.WriteString("requestId: " + requestID + "\n")
	}
	b.WriteString(fmt.Sprintf("%s %s %s\n", r.Method, r.URL.RequestURI(), r.Proto))
	b.WriteString("Host: " + r.Host + "\n")
	b.WriteString("RemoteAddr: " + r.RemoteAddr + "\n")

	keys := make([]string, 0, len(r.Header))
	for k := range r.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		val := strings.Join(r.Header.Values(k), ", ")
		lk := strings.ToLower(k)
		if redactAuth && lk == "authorization" {
			val = "<redacted>"
		}
		if redactCookie && lk == "cookie" {
			val = "<redacted>"
		}
		b.WriteString(k + ": " + val + "\n")
	}
	b.WriteString("\n")
	b.WriteString(truncateBytes(body, maxBytes))
	if len(body) > maxBytes && maxBytes > 0 {
		b.WriteString("\n\n<TRUNCATED>")
	}
	b.WriteString("\n----- TR069 RAW REQUEST END -----")
	return b.String()
}

type responseCaptureWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
	max  int
}

func newResponseCaptureWriter(w gin.ResponseWriter, maxBytes int) *responseCaptureWriter {
	max := maxBytes
	if max <= 0 {
		max = 64 * 1024
	}
	return &responseCaptureWriter{ResponseWriter: w, max: max}
}

func (w *responseCaptureWriter) Write(p []byte) (int, error) {
	if w.max > 0 && w.body.Len() < w.max {
		remain := w.max - w.body.Len()
		if remain > 0 {
			if len(p) > remain {
				_, _ = w.body.Write(p[:remain])
			} else {
				_, _ = w.body.Write(p)
			}
		}
	}
	return w.ResponseWriter.Write(p)
}

func dumpResponse(status int, headers http.Header, body []byte, requestID string, elapsed time.Duration, maxBytes int) string {
	var b strings.Builder
	b.WriteString("----- TR069 RAW RESPONSE BEGIN -----\n")
	if requestID != "" {
		b.WriteString("requestId: " + requestID + "\n")
	}
	b.WriteString(fmt.Sprintf("status: %d\n", status))
	b.WriteString("elapsed: " + elapsed.String() + "\n")

	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		b.WriteString(k + ": " + strings.Join(headers.Values(k), ", ") + "\n")
	}
	b.WriteString("\n")
	b.WriteString(truncateBytes(body, maxBytes))
	if len(body) > maxBytes && maxBytes > 0 {
		b.WriteString("\n\n<TRUNCATED>")
	}
	b.WriteString("\n----- TR069 RAW RESPONSE END -----")
	return b.String()
}

func truncateBytes(b []byte, max int) string {
	if max <= 0 || len(b) <= max {
		return string(b)
	}
	return string(b[:max])
}
