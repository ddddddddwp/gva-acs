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
	if tr069Global.GlobalConfig != nil {
		global.GVA_LOG.Info("TR069 Config Check",
			zap.Bool("DumpRaw", tr069Global.GlobalConfig.DumpRaw),
			zap.Int("DumpMaxBytes", tr069Global.GlobalConfig.DumpMaxBytes),
			zap.Bool("InfoLogEnable", tr069Global.GlobalConfig.InfoLogEnable),
			zap.String("InfoLogDir", tr069Global.GlobalConfig.InfoLogDir),
		)
		// Force print to stdout to ensure visibility even if logger is file-only
		fmt.Printf("\n[TR069-DEBUG] Config Loaded - DumpRaw: %v InfoLogEnable: %v InfoLogDir: %s\n",
			tr069Global.GlobalConfig.DumpRaw,
			tr069Global.GlobalConfig.InfoLogEnable,
			tr069Global.GlobalConfig.InfoLogDir,
		)
	} else {
		global.GVA_LOG.Error("TR069 GlobalConfig is nil")
		fmt.Println("\n[TR069-DEBUG] GlobalConfig is nil")
	}

	addr := tr069Global.GlobalConfig.Address
	if addr == "" {
		addr = ":7458" // Default port
	}

	engine := SetupEngine()

	go func() {
		global.GVA_LOG.Info("Starting TR069 Server", zap.String("address", addr))
		if err := engine.Run(addr); err != nil {
			global.GVA_LOG.Error("TR069 Server failed to start", zap.Error(err))
		}
	}()
}

// SetupEngine creates and configures the gin engine for TR069
// Exported for testing purposes
func SetupEngine() *gin.Engine {
	engine := gin.New()
	engine.Use(gin.Recovery())
	// TR069 调试辅助：为每个请求生成仅用于内部日志链路的 traceId。
	// 删除/禁用：移除这一行即可，不影响核心 TR069 处理逻辑。
	engine.Use(middleware.EnsureTraceID())
	// 原始报文中间件始终安装；每个请求从原子运行时快照决定是否捕获。
	engine.Use(middleware.RawDump(middleware.RawDumpConfig{}))
	engine.Use(middleware.RawResponseDump(middleware.RawResponseDumpConfig{}))

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
		if len(cwmpID) > 12 {
			cwmpID = cwmpID[:12]
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

	return engine
}
