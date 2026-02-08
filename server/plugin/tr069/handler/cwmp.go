package handler

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/adapter"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/engine"
	"github.com/ddddddddwp/tr069-core-only/factory"
	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CWMPHandler(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	clientIP := remoteIPFromRequest(c.Request)

	if len(body) > 0 {
		p := factory.NewParser(tr069.WithStrictMode(false))
		msg, parseErr := p.ParseMessage(ctx, body)
		if parseErr == nil && msg != nil && msg.Method == tr069.MethodInform && msg.DeviceID != nil {
			ctx = adapter.WithDeviceMeta(ctx, msg.DeviceID, clientIP)
		}
	}

	eng, err := engine.Get()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	reqID := c.GetHeader("X-Request-Id")
	if reqID == "" {
		reqID = uuid.NewString()
	}

	headers := map[string]string{}
	copyHeaderIfPresent(headers, c.Request, "Cookie")
	copyHeaderIfPresent(headers, c.Request, "User-Agent")
	copyHeaderIfPresent(headers, c.Request, "X-Forwarded-Proto")
	copyHeaderIfPresent(headers, c.Request, "X-Forwarded-For")
	copyHeaderIfPresent(headers, c.Request, "X-Forwarded-Ssl")
	copyHeaderIfPresent(headers, c.Request, "X-Real-Ip")
	copyHeaderIfPresent(headers, c.Request, "X-TR069-Session")

	resp, err := eng.Handle(ctx, &core.Request{
		ID:         reqID,
		RemoteIP:   clientIP,
		Headers:    headers,
		Body:       body,
		ReceivedAt: time.Now(),
	})
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if resp == nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	for k, v := range resp.Headers {
		if k != "" && v != "" {
			c.Header(k, v)
		}
	}
	if resp.StatusCode == http.StatusNoContent {
		c.Status(resp.StatusCode)
		return
	}
	c.Data(resp.StatusCode, "text/xml", resp.Body)
}

func copyHeaderIfPresent(out map[string]string, r *http.Request, name string) {
	if out == nil || r == nil || name == "" {
		return
	}
	if v := r.Header.Get(name); v != "" {
		out[name] = v
	}
}

func remoteIPFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-Ip")); xrip != "" {
		return xrip
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
