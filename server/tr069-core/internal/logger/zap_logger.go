// Package logger provides implementation for detailed logging functionality.
package logger

import (
	"io"
	"os"
	"sync"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// zapLogger 实现了interfaces.Logger接口，使用zap.SugaredLogger作为底层实现
type zapLogger struct {
	mu       sync.Mutex
	sugar  *zap.SugaredLogger
	level  interfaces.LogLevel
	format interfaces.LogFormat
	fields []interfaces.LogField
	ctx    interface{}
	zapLevel zap.AtomicLevel
}

// NewZapLogger 创建一个新的基于zap的日志记录器
func NewZapLogger(options ...interfaces.LoggerOption) interfaces.Logger {
	// 创建默认的zap配置
	zapLevel := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	
	// 默认输出到标准输出
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 默认使用JSON编码器
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	
	// 默认输出到标准输出
	core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapLevel)
	
	// 创建zap logger
	zapLogger := zap.New(core)
	sugar := zapLogger.Sugar()

	// 创建我们的logger包装
	logger := &zapLogger{
		sugar:    sugar,
		level:    interfaces.LogLevelInfo,
		format:   interfaces.LogFormatJSON,
		fields:   make([]interfaces.LogField, 0),
		zapLevel: zapLevel,
	}

	// 应用选项
	for _, option := range options {
		option(logger)
	}

	return logger
}

// Debug 实现Debug级别日志记录
func (l *zapLogger) Debug(msg string, fields ...interfaces.LogField) {
	if l.level <= interfaces.LogLevelDebug {
		args := l.convertFields(fields)
		l.sugar.Debugw(msg, args...)
	}
}

// Info 实现Info级别日志记录
func (l *zapLogger) Info(msg string, fields ...interfaces.LogField) {
	if l.level <= interfaces.LogLevelInfo {
		args := l.convertFields(fields)
		l.sugar.Infow(msg, args...)
	}
}

// Warn 实现Warn级别日志记录
func (l *zapLogger) Warn(msg string, fields ...interfaces.LogField) {
	if l.level <= interfaces.LogLevelWarn {
		args := l.convertFields(fields)
		l.sugar.Warnw(msg, args...)
	}
}

// Error 实现Error级别日志记录
func (l *zapLogger) Error(msg string, fields ...interfaces.LogField) {
	if l.level <= interfaces.LogLevelError {
		args := l.convertFields(fields)
		l.sugar.Errorw(msg, args...)
	}
}

// Fatal 实现Fatal级别日志记录
func (l *zapLogger) Fatal(msg string, fields ...interfaces.LogField) {
	if l.level <= interfaces.LogLevelFatal {
		args := l.convertFields(fields)
		l.sugar.Fatalw(msg, args...)
	}
}

// WithFields 创建带有固定字段的新日志记录器
func (l *zapLogger) WithFields(fields ...interfaces.LogField) interfaces.Logger {
	// 创建新的logger
	newLogger := &zapLogger{
		sugar:    l.sugar,
		level:    l.level,
		format:   l.format,
		ctx:      l.ctx,
		zapLevel: l.zapLevel,
	}

	// 复制现有字段
	newLogger.fields = make([]interfaces.LogField, len(l.fields)+len(fields))
	copy(newLogger.fields, l.fields)
	copy(newLogger.fields[len(l.fields):], fields)

	// 将字段添加到zap logger
	args := l.convertFields(fields)
	newLogger.sugar = l.sugar.With(args...)

	return newLogger
}

// WithContext 创建带有上下文的新日志记录器
func (l *zapLogger) WithContext(ctx interface{}) interfaces.Logger {
	newLogger := &zapLogger{
		sugar:    l.sugar,
		level:    l.level,
		format:   l.format,
		ctx:      ctx,
		zapLevel: l.zapLevel,
	}

	// 复制现有字段
	newLogger.fields = make([]interfaces.LogField, len(l.fields))
	copy(newLogger.fields, l.fields)

	// 添加上下文字段
	newLogger.sugar = l.sugar.With("context", ctx)

	return newLogger
}

// SetLevel 设置日志级别
func (l *zapLogger) SetLevel(level interfaces.LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	l.level = level
	
	// 同时设置zap的日志级别
	switch level {
	case interfaces.LogLevelDebug:
		l.zapLevel.SetLevel(zapcore.DebugLevel)
	case interfaces.LogLevelInfo:
		l.zapLevel.SetLevel(zapcore.InfoLevel)
	case interfaces.LogLevelWarn:
		l.zapLevel.SetLevel(zapcore.WarnLevel)
	case interfaces.LogLevelError:
		l.zapLevel.SetLevel(zapcore.ErrorLevel)
	case interfaces.LogLevelFatal:
		l.zapLevel.SetLevel(zapcore.FatalLevel)
	}
}

// GetLevel 获取当前日志级别
func (l *zapLogger) GetLevel() interfaces.LogLevel {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// SetFormat 设置日志格式
func (l *zapLogger) SetFormat(format interfaces.LogFormat) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	// 保存格式设置
	l.format = format
	
	// 重新创建logger以应用新格式
	// 注意：这里简化处理，实际应用中可能需要更复杂的重建逻辑
	var encoder zapcore.Encoder
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	
	switch format {
	case interfaces.LogFormatJSON:
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	case interfaces.LogFormatText:
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		// 默认使用JSON
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}
	
	// 获取当前的输出
	core := l.sugar.Desugar().Core()
	// 这里简化处理，实际应用中需要更复杂的逻辑来保留现有的输出目标
	zapLogger := zap.New(zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), l.zapLevel))
	l.sugar = zapLogger.Sugar()
}

// GetFormat 获取当前日志格式
func (l *zapLogger) GetFormat() interfaces.LogFormat {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.format
}

// SetOutput 设置日志输出目标
func (l *zapLogger) SetOutput(output interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	var writer zapcore.WriteSyncer
	if w, ok := output.(io.Writer); ok {
		writer = zapcore.AddSync(w)
	} else {
		// 默认输出到标准输出
		writer = zapcore.AddSync(os.Stdout)
	}
	
	// 获取当前的编码器和日志级别
	oldCore := l.sugar.Desugar().Core()
	// 这里简化处理，实际应用中需要更复杂的逻辑来提取现有的编码器
	var encoder zapcore.Encoder
	if l.format == interfaces.LogFormatJSON {
		encoder = zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		})
	} else {
		encoder = zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		})
	}
	
	// 创建新的core和logger
	core := zapcore.NewCore(encoder, writer, l.zapLevel)
	zapLogger := zap.New(core)
	l.sugar = zapLogger.Sugar()
}

// convertFields 将interfaces.LogField转换为zap的键值对
func (l *zapLogger) convertFields(fields []interfaces.LogField) []interface{} {
	if len(fields) == 0 {
		return nil
	}
	
	args := make([]interface{}, 0, len(fields)*2)
	for _, field := range fields {
		args = append(args, field.Key, field.Value)
	}
	
	return args
}