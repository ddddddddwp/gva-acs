package initialize

import (
	"fmt"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	tr069Global "github.com/ddddddddwp/gva-acs/server/plugin/tr069/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/handler"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func StartTR069Server() {
	addr := tr069Global.GlobalConfig.Address
	if addr == "" {
		addr = ":7458" // Default port
	}

	engine := gin.New()
	engine.Use(gin.Recovery())
	// TR069 调试辅助：确保每个请求都有 requestId（Header: X-Request-Id，缺省则自动生成）。
	// 删除/禁用：移除这一行即可，不影响核心 TR069 处理逻辑。
	engine.Use(middleware.EnsureRequestID())

	if tr069Global.GlobalConfig != nil && tr069Global.GlobalConfig.DumpRaw {
		// TR069 调试辅助：原始报文 Dump（打印请求行/头/Body 到终端），默认关闭。
		// 开关：config.yaml -> tr069.dumpRaw
		// 删除/禁用：删除本段或将 dumpRaw=false 即可。
		engine.Use(middleware.RawDump(middleware.RawDumpConfig{
			MaxBytes:      tr069Global.GlobalConfig.DumpMaxBytes,
			RedactAuth:    tr069Global.GlobalConfig.DumpRedactAuth,
			RedactCookie:  tr069Global.GlobalConfig.DumpRedactCookie,
			PrintResponse: false,
		}))
	}

	engine.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		var statusColor, methodColor, resetColor string
		if param.IsOutputColor() {
			statusColor = param.StatusCodeColor()
			methodColor = param.MethodColor()
			resetColor = param.ResetColor()
		}
		if param.Latency > time.Minute {
			param.Latency = param.Latency.Truncate(time.Second)
		}
		cwmpID := ""
		if param.Keys != nil {
			if v, ok := param.Keys["cwmpId"]; ok {
				if s, ok := v.(string); ok {
					cwmpID = s
				}
			}
		}
		reqPart := ""
		if cwmpID != "" {
			reqPart = " | cwmp:" + cwmpID
		}
		return fmt.Sprintf(
			"[TR069] %s |%s %3d %s| %13v | %15s |%s %-7s %s %s%s\n%s",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			statusColor,
			param.StatusCode,
			resetColor,
			param.Latency,
			param.ClientIP,
			methodColor,
			param.Method,
			param.Path,
			resetColor,
			reqPart,
			param.ErrorMessage,
		)
	}))

	// Register CWMP Handler
	engine.POST("/", handler.CWMPHandler)
	engine.POST("/acs", handler.CWMPHandler)

	go func() {
		global.GVA_LOG.Info("Starting TR069 Server", zap.String("address", addr))
		if err := engine.Run(addr); err != nil {
			global.GVA_LOG.Error("TR069 Server failed to start", zap.Error(err))
		}
	}()
}
