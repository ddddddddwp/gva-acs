// 示例代码：展示如何使用TR069库的zap日志适配器
package main

import (
	"os"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// 方法1：使用默认日志记录器
	defaultLogger := logger.DefaultLogger()
	defaultLogger.Info("使用默认日志记录器")

	// 方法2：使用工厂函数创建日志记录器
	factory := logger.NewZapLoggerFactory()
	customLogger := factory(
		logger.WithLogLevel(interfaces.LogLevelDebug),
		logger.WithLogFormat(interfaces.LogFormatJSON),
	)
	
	customLogger.Debug("这是一条调试日志")
	customLogger.Info("这是一条信息日志", 
		interfaces.LogField{Key: "module", Value: "parser"},
		interfaces.LogField{Key: "version", Value: "1.0.0"},
	)

	// 方法3：直接使用zap创建日志记录器并适配
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	config.OutputPaths = []string{"stdout"}
	
	zapLogger, _ := config.Build()
	sugar := zapLogger.Sugar()
	
	// 使用适配器将zap.SugaredLogger适配到interfaces.Logger
	adaptedLogger := logger.NewZapAdapter(sugar)
	
	adaptedLogger.Info("使用适配器创建的日志记录器")
	
	// 添加上下文字段
	sessionLogger := adaptedLogger.WithFields(
		interfaces.LogField{Key: "session_id", Value: "session-123"},
		interfaces.LogField{Key: "device_id", Value: "device-456"},
	)
	
	sessionLogger.Info("处理设备请求")
	sessionLogger.Error("设备连接失败", 
		interfaces.LogField{Key: "error", Value: "connection timeout"},
		interfaces.LogField{Key: "retry_count", Value: 3},
	)
	
	// 使用全局日志函数
	logger.SetGlobalLogger(customLogger)
	logger.Info("使用全局日志函数", interfaces.LogField{Key: "global", Value: true})
	logger.Error("全局错误日志", interfaces.LogField{Key: "error_code", Value: 500})
	
	// 创建开发环境日志记录器
	devLogger := logger.NewDevelopmentZapLogger()
	devLogger.Debug("开发环境调试日志")
	
	// 演示不同日志级别
	devLogger.SetLevel(interfaces.LogLevelWarn)
	devLogger.Debug("此消息不会显示") // 不会显示，因为级别已设置为Warn
	devLogger.Info("此消息不会显示")  // 不会显示，因为级别已设置为Warn
	devLogger.Warn("此警告消息会显示") // 会显示
	devLogger.Error("此错误消息会显示") // 会显示
}