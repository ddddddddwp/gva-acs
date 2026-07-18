package handler

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"time"

	gvaGlobal "github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/adapter"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/engine"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/trace"
	"github.com/ddddddddwp/tr069-core-only/factory"
	tr069 "github.com/ddddddddwp/tr069-core-only/interface"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func CWMPHandler(c *gin.Context) {
	traceIDValue, _ := c.Get("traceId")
	traceID, _ := traceIDValue.(string)
	if traceID == "" {
		traceID = uuid.NewString()
		c.Set("traceId", traceID)
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
	ctx = trace.WithTraceID(ctx, traceID)
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
	copyHeaderIfPresent(headers, c.Request, "Authorization")

	trace.Add(ctx, "engine.handle", "engine handle begin", map[string]string{
		"hasCookie": strconv.FormatBool(headers["Cookie"] != ""),
		"hasAuth":   strconv.FormatBool(c.Request.Header.Get("Authorization") != ""),
	})
	handleStart := time.Now()
	resp, err := eng.Handle(ctx, &core.Request{
		TraceID:    traceID,
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

	if config.CurrentRuntime().Settings.Debug {
		gvaGlobal.GVA_LOG.Debug("TR069 Trace", zap.String("traceId", traceID), zap.Any("trace", trace.Get(ctx, traceID)))
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
