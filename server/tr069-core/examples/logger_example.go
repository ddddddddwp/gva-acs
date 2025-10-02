// 示例代码：展示如何使用TR069库的日志组件（基于zap实现）
package main

import (
	"os"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/internal/logger"
)

func main() {
	// 创建基于Zap的日志记录器
	zapLogger := logger.NewZapLogger(
		logger.WithLogLevel(interfaces.LogLevelDebug),
		logger.WithLogFormat(interfaces.LogFormatJSON),
		logger.WithLogOutput(os.Stdout),
	)

	// 使用日志记录器
	zapLogger.Info("TR069库初始化成功", interfaces.LogField{Key: "version", Value: "1.0.0"})

	// 添加固定字段
	sessionLogger := zapLogger.WithFields(
		interfaces.LogField{Key: "session_id", Value: "12345"},
		interfaces.LogField{Key: "device_id", Value: "device-001"},
	)

	// 记录不同级别的日志
	sessionLogger.Debug("开始解析消息")
	sessionLogger.Info("消息解析完成", interfaces.LogField{Key: "duration_ms", Value: 15})

	// 模拟处理错误
	sessionLogger.Error("解析参数失败",
		interfaces.LogField{Key: "error", Value: "invalid parameter format"},
		interfaces.LogField{Key: "parameter", Value: "InternetGatewayDevice.DeviceInfo.ModelName"},
	)

	// 使用上下文
	requestLogger := sessionLogger.WithContext(map[string]string{
		"request_id": "req-789",
		"method":     "GetParameterValues",
	})

	requestLogger.Info("处理RPC请求")

	// 修改日志级别
	zapLogger.SetLevel(interfaces.LogLevelWarn)
	zapLogger.Debug("此消息不会显示") // 不会显示，因为级别已设置为Warn
	zapLogger.Warn("警告消息会显示")  // 会显示

	// 修改日志格式
	zapLogger.SetFormat(interfaces.LogFormatText)
	zapLogger.Info("以文本格式显示的消息", interfaces.LogField{Key: "time", Value: time.Now().String()})

	// 创建一个新的日志记录器工厂
	loggerFactory := logger.NewZapLoggerFactory()

	// 使用工厂创建日志记录器
	factoryLogger := loggerFactory(
		logger.WithLogLevel(interfaces.LogLevelInfo),
		logger.WithLogFormat(interfaces.LogFormatJSON),
	)

	factoryLogger.Info("使用工厂创建的日志记录器")
}
