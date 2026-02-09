package handler

import (
	"context"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	gvaGlobal "github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/adapter"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/engine"
	tr069Global "github.com/ddddddddwp/gva-acs/server/plugin/tr069/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/trace"
	"github.com/ddddddddwp/tr069-core-only/factory"
	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func CWMPHandler(c *gin.Context) {
	reqIDVal, _ := c.Get("requestId")
	reqID, _ := reqIDVal.(string)
	if reqID == "" {
		reqID = c.GetHeader("X-Request-Id")
		if reqID == "" {
			reqID = uuid.NewString()
		}
		c.Set("requestId", reqID)
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	ctx := c.Request.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = trace.WithRequestID(ctx, reqID)
	clientIP := remoteIPFromRequest(c.Request)

	trace.Add(ctx, "http.recv", "request received", map[string]string{
		"ip":      clientIP,
		"path":    c.Request.URL.Path,
		"len":     strconv.Itoa(len(body)),
		"hasBody": strconv.FormatBool(len(body) > 0),
	})

	if len(body) > 0 {
		p := factory.NewParser(tr069.WithStrictMode(false))
		parseStart := time.Now()
		msg, parseErr := p.ParseMessage(ctx, body)
		trace.Add(ctx, "parse.done", "parser finished", map[string]string{
			"elapsed": time.Since(parseStart).String(),
			"ok":      strconv.FormatBool(parseErr == nil),
		})
		if parseErr == nil && msg != nil {
			c.Set("cwmpId", msg.ID)
			trace.Add(ctx, "parse.msg", "message parsed", map[string]string{
				"cwmpId": msg.ID,
				"method": msg.Method,
			})
		}
		if parseErr == nil && msg != nil && msg.Method == tr069.MethodInform && msg.DeviceID != nil {
			ctx = adapter.WithDeviceMeta(ctx, msg.DeviceID, clientIP)
			trace.Add(ctx, "inform.device", "device meta extracted", map[string]string{
				"manufacturer": msg.DeviceID.Manufacturer,
				"oui":          msg.DeviceID.OUI,
				"productClass": msg.DeviceID.ProductClass,
				"serialNumber": msg.DeviceID.SerialNumber,
			})
		}
	}

	eng, err := engine.Get()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	headers := map[string]string{}
	copyHeaderIfPresent(headers, c.Request, "Cookie")
	copyHeaderIfPresent(headers, c.Request, "User-Agent")
	copyHeaderIfPresent(headers, c.Request, "X-Forwarded-Proto")
	copyHeaderIfPresent(headers, c.Request, "X-Forwarded-For")
	copyHeaderIfPresent(headers, c.Request, "X-Forwarded-Ssl")
	copyHeaderIfPresent(headers, c.Request, "X-Real-Ip")
	copyHeaderIfPresent(headers, c.Request, "X-TR069-Session")

	trace.Add(ctx, "engine.handle", "engine handle begin", map[string]string{
		"hasCookie": strconv.FormatBool(headers["Cookie"] != ""),
		"hasAuth":   strconv.FormatBool(c.Request.Header.Get("Authorization") != ""),
	})
	handleStart := time.Now()
	resp, err := eng.Handle(ctx, &core.Request{
		ID:         reqID,
		RemoteIP:   clientIP,
		Headers:    headers,
		Body:       body,
		ReceivedAt: time.Now(),
	})
	trace.Add(ctx, "engine.handle", "engine handle end", map[string]string{
		"elapsed": time.Since(handleStart).String(),
		"ok":      strconv.FormatBool(err == nil),
	})
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	if resp == nil {
		c.Status(http.StatusInternalServerError)
		return
	}
	trace.Add(ctx, "http.resp", "response ready", map[string]string{
		"status": strconv.Itoa(resp.StatusCode),
	})
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

	if tr069Global.GlobalConfig != nil && tr069Global.GlobalConfig.Debug {
		gvaGlobal.GVA_LOG.Debug("TR069 Trace", zap.String("requestId", reqID), zap.Any("trace", trace.Get(ctx, reqID)))
	}
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
