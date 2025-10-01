// Package logger 提供TR069库的日志组件实现
package logger

import (
	"github.com/root/demo/tr069/interfaces"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewZapLoggerFactory 创建基于Zap的日志记录器工厂函数
func NewZapLoggerFactory() func(options ...interfaces.LoggerOption) interfaces.Logger {
	return func(options ...interfaces.LoggerOption) interfaces.Logger {
		// 创建默认的zap logger
		zapLogger, _ := zap.NewProduction()
		sugar := zapLogger.Sugar()
		return NewZapAdapter(sugar)
	}
}

// newZapLoggerWithOptions 内部使用的工厂函数，支持Option类型
func newZapLoggerWithOptions(options ...Option) interfaces.Logger {
	opts := defaultOptions()
	applyOptions(opts, options...)
	
	// 创建zap配置
	zapConfig := zap.NewProductionConfig()
	
	// 设置日志级别
	switch opts.level {
	case interfaces.LogLevelDebug:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case interfaces.LogLevelInfo:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case interfaces.LogLevelWarn:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case interfaces.LogLevelError:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case interfaces.LogLevelFatal:
		zapConfig.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	}
	
	// 设置日志格式
	switch opts.format {
	case interfaces.LogFormatJSON:
		zapConfig.Encoding = "json"
	case interfaces.LogFormatText:
		zapConfig.Encoding = "console"
	}
	
	// 创建zap logger
	logger, err := zapConfig.Build()
	if err != nil {
		// 如果创建失败，使用默认配置
		logger, _ = zap.NewProduction()
	}
	
	// 创建SugaredLogger
	sugar := logger.Sugar()
	
	// 创建适配器
	return NewZapAdapter(sugar)
}

// DefaultLogger 创建默认的日志记录器，使用文件存储
func DefaultLogger() interfaces.Logger {
	// 使用默认的Core日志存储配置
	config := DefaultCoreLogStorageConfig()
	return NewFileLogger(config)
}

// DefaultCoreLogger 创建专门用于TR069 Core的日志记录器
func DefaultCoreLogger() interfaces.Logger {
	config := DefaultCoreLogStorageConfig()
	return NewFileLogger(config)
}

// DefaultConsoleLogger 创建输出到控制台的日志记录器
func DefaultConsoleLogger() interfaces.Logger {
	factory := NewZapLoggerFactory()
	return factory()
}

// SetGlobalLogger 设置全局日志记录器
var globalLogger interfaces.Logger = DefaultCoreLogger() // 使用Core日志记录器作为默认全局记录器

// GetGlobalLogger 获取全局日志记录器
func GetGlobalLogger() interfaces.Logger {
	return globalLogger
}

// SetGlobalLogger 设置全局日志记录器
func SetGlobalLogger(logger interfaces.Logger) {
	globalLogger = logger
}

// Debug 使用全局日志记录器记录Debug级别日志
func Debug(msg string, fields ...interfaces.LogField) {
	globalLogger.Debug(msg, fields...)
}

// Info 使用全局日志记录器记录Info级别日志
func Info(msg string, fields ...interfaces.LogField) {
	globalLogger.Info(msg, fields...)
}

// Warn 使用全局日志记录器记录Warn级别日志
func Warn(msg string, fields ...interfaces.LogField) {
	globalLogger.Warn(msg, fields...)
}

// Error 使用全局日志记录器记录Error级别日志
func Error(msg string, fields ...interfaces.LogField) {
	globalLogger.Error(msg, fields...)
}

// Fatal 使用全局日志记录器记录Fatal级别日志
func Fatal(msg string, fields ...interfaces.LogField) {
	globalLogger.Fatal(msg, fields...)
}