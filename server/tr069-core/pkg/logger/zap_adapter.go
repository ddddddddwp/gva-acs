// Package logger 提供TR069库的日志组件实现
package logger

import (
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"go.uber.org/zap"
)

// NewZapAdapter 创建一个将zap.SugaredLogger适配到interfaces.Logger的适配器
func NewZapAdapter(sugar *zap.SugaredLogger) interfaces.Logger {
	return &zapAdapter{
		sugar: sugar,
		level: interfaces.LogLevelInfo,
	}
}

// NewDefaultZapLogger 使用默认配置创建一个基于zap的日志记录器
func NewDefaultZapLogger() interfaces.Logger {
	config := zap.NewProductionConfig()
	logger, _ := config.Build()
	sugar := logger.Sugar()
	return NewZapAdapter(sugar)
}

// NewDevelopmentZapLogger 创建一个适合开发环境的zap日志记录器
func NewDevelopmentZapLogger() interfaces.Logger {
	config := zap.NewDevelopmentConfig()
	logger, _ := config.Build()
	sugar := logger.Sugar()
	return NewZapAdapter(sugar)
}

// NewCustomZapLogger 使用自定义配置创建zap日志记录器
func NewCustomZapLogger(config zap.Config) (interfaces.Logger, error) {
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	sugar := logger.Sugar()
	return NewZapAdapter(sugar), nil
}

// zapAdapter 是zap.SugaredLogger到interfaces.Logger的适配器
type zapAdapter struct {
	sugar  *zap.SugaredLogger
	level  interfaces.LogLevel
	format interfaces.LogFormat
}

// Debug 实现Debug级别日志记录
func (l *zapAdapter) Debug(msg string, fields ...interfaces.LogField) {
	if l.level <= interfaces.LogLevelDebug {
		args := convertFields(fields)
		l.sugar.Debugw(msg, args...)
	}
}

// Info 实现Info级别日志记录
func (l *zapAdapter) Info(msg string, fields ...interfaces.LogField) {
	if l.level <= interfaces.LogLevelInfo {
		args := convertFields(fields)
		l.sugar.Infow(msg, args...)
	}
}

// Warn 实现Warn级别日志记录
func (l *zapAdapter) Warn(msg string, fields ...interfaces.LogField) {
	if l.level <= interfaces.LogLevelWarn {
		args := convertFields(fields)
		l.sugar.Warnw(msg, args...)
	}
}

// Error 实现Error级别日志记录
func (l *zapAdapter) Error(msg string, fields ...interfaces.LogField) {
	if l.level <= interfaces.LogLevelError {
		args := convertFields(fields)
		l.sugar.Errorw(msg, args...)
	}
}

// Fatal 实现Fatal级别日志记录
func (l *zapAdapter) Fatal(msg string, fields ...interfaces.LogField) {
	if l.level <= interfaces.LogLevelFatal {
		args := convertFields(fields)
		l.sugar.Fatalw(msg, args...)
	}
}

// WithFields 创建带有固定字段的新日志记录器
func (l *zapAdapter) WithFields(fields ...interfaces.LogField) interfaces.Logger {
	args := convertFields(fields)
	newSugar := l.sugar.With(args...)

	return &zapAdapter{
		sugar:  newSugar,
		level:  l.level,
		format: l.format,
	}
}

// WithContext 创建带有上下文的新日志记录器
func (l *zapAdapter) WithContext(ctx interface{}) interfaces.Logger {
	newSugar := l.sugar.With("context", ctx)

	return &zapAdapter{
		sugar:  newSugar,
		level:  l.level,
		format: l.format,
	}
}

// SetLevel 设置日志级别
func (l *zapAdapter) SetLevel(level interfaces.LogLevel) {
	l.level = level

	// 同时设置zap的日志级别
	switch level {
	case interfaces.LogLevelDebug:
		// zapLevel = zapcore.DebugLevel
	case interfaces.LogLevelInfo:
		// zapLevel = zapcore.InfoLevel
	case interfaces.LogLevelWarn:
		// zapLevel = zapcore.WarnLevel
	case interfaces.LogLevelError:
		// zapLevel = zapcore.ErrorLevel
	case interfaces.LogLevelFatal:
		// zapLevel = zapcore.FatalLevel
	}
	// 注意：动态设置zap日志级别需要重新创建logger，这里暂时跳过
}

// GetLevel 获取当前日志级别
func (l *zapAdapter) GetLevel() interfaces.LogLevel {
	return l.level
}

// SetFormat 设置日志格式
func (l *zapAdapter) SetFormat(format interfaces.LogFormat) {
	l.format = format
	// 注意：zap.SugaredLogger没有直接暴露修改格式的方法
	// 这里只是记录格式，但实际上没有改变zap的输出格式
}

// GetFormat 获取当前日志格式
func (l *zapAdapter) GetFormat() interfaces.LogFormat {
	return l.format
}

// SetOutput 设置日志输出目标
func (l *zapAdapter) SetOutput(output interface{}) {
	// 注意：zap.SugaredLogger没有直接暴露修改输出的方法
	// 这里简化处理，实际应用中需要重新创建logger
}

// convertFields 将interfaces.LogField转换为zap的键值对
func convertFields(fields []interfaces.LogField) []interface{} {
	if len(fields) == 0 {
		return nil
	}

	args := make([]interface{}, 0, len(fields)*2)
	for _, field := range fields {
		args = append(args, field.Key, field.Value)
	}

	return args
}
