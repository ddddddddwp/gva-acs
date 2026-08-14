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

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/infolog"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/redact"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/trace"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type RawDumpConfig struct {
	MaxBytes      int
	RedactAuth    bool
	RedactCookie  bool
	PrintResponse bool
	DumpToConsole bool // 新增：控制是否打印到终端
}

const defaultDumpMaxBytes = 64 * 1024

// EnsureTraceID 是 TR069 调试辅助中间件：
// - 为每个请求生成内部 UUID，并写入 gin.Context(key="traceId")
// - 不接受外部链路标识覆盖，也不向 CPE 返回内部 Trace ID
// 删除/禁用：从 TR069 server 的 middleware 链中移除此中间件即可。
func EnsureTraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		if v, ok := c.Get("traceId"); ok {
			if s, ok := v.(string); ok && s != "" {
				c.Next()
				return
			}
		}
		c.Set("traceId", uuid.NewString())
		c.Next()
	}
}

// RawDump 是 TR069 调试辅助中间件：把 CPE 发来的原始 HTTP 报文（请求行/头/Body）打印到终端。
// 特点：
// - 不依赖 TR069 解析，可用于定位 CPE 实际发送内容
// - 使用 BEGIN/END 多行块输出，避免和访问日志混在同一行
// - 支持脱敏 Authorization/Cookie，并限制最大打印字节数
// 删除/禁用：从 TR069 server 的 middleware 链中移除或将 config.yaml 的 tr069.dumpRaw=false。
// cfg 参数仅保留源码兼容；运行行为由每个请求开始时的原子配置快照决定。
func RawDump(_ RawDumpConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		settings := config.CurrentRuntime().Settings
		if !rawDumpEnabled(settings) {
			c.Next()
			return
		}
		maxBytes := dumpMaxBytes(settings)
		traceIDValue, _ := c.Get("traceId")
		traceID, _ := traceIDValue.(string)

		var reqBody []byte
		if c.Request != nil && c.Request.Body != nil {
			b, err := io.ReadAll(c.Request.Body)
			if err == nil {
				reqBody = b
				c.Request.Body = io.NopCloser(bytes.NewReader(b))
			}
		}

		logBody := sanitizedCWMPLogCopy(reqBody)
		reqDump := dumpRequest(c.Request, logBody, traceID, settings.DumpRedactAuth, settings.DumpRedactCookie, maxBytes)
		if settings.DumpRaw {
			_, _ = fmt.Fprintln(os.Stdout, reqDump)
		}
		if settings.InfoLogEnable {
			// 如果已开启独立文件日志，则不再打印到 GVA_LOG，避免 Zap 结构化日志将换行符转义为 \n 导致阅读困难
			// if global.GVA_LOG != nil {
			// 	global.GVA_LOG.Info("TR069 RAW REQUEST", zap.String("dump", reqDump))
			// }
			writeInfoLog(reqDump, settings)
		}
		if traceID != "" {
			trace.Default.Add(traceID, trace.Entry{
				At:      time.Now(),
				Stage:   "raw.request",
				Message: truncateBytes(logBody, maxBytes),
			})
		}

		c.Next()
	}
}

func sanitizedCWMPLogCopy(body []byte) []byte {
	sanitized, err := redact.CWMPXML(append([]byte(nil), body...))
	if err != nil {
		return []byte("<REDACTED: malformed XML>")
	}
	return sanitized
}

func rawDumpEnabled(settings config.TR069Config) bool {
	return settings.DumpRaw || settings.InfoLogEnable
}

func dumpMaxBytes(settings config.TR069Config) int {
	if settings.DumpMaxBytes > 0 {
		return settings.DumpMaxBytes
	}
	return defaultDumpMaxBytes
}

func dumpRequest(r *http.Request, body []byte, traceID string, redactAuth, redactCookie bool, maxBytes int) string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("----- TR069 RAW REQUEST BEGIN -----\n")
	if traceID != "" {
		b.WriteString("traceId: " + traceID + "\n")
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

func dumpResponse(status int, headers http.Header, body []byte, traceID string, elapsed time.Duration, maxBytes int) string {
	var b strings.Builder
	b.WriteString("----- TR069 RAW RESPONSE BEGIN -----\n")
	if traceID != "" {
		b.WriteString("traceId: " + traceID + "\n")
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

func writeInfoLog(s string, settings config.TR069Config) {
	infolog.WriteWithSettings(s, settings)
}
