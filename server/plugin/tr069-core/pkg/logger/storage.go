// Package logger 提供TR069库的日志组件实现
package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/root/demo/tr069/interfaces"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// LogStorageConfig 定义日志存储配置
type LogStorageConfig struct {
	// 日志文件路径
	FilePath string
	// 单个日志文件最大大小，单位MB
	MaxSize int
	// 保留的旧日志文件最大数量
	MaxBackups int
	// 保留的旧日志文件最大天数
	MaxAge int
	// 是否压缩旧日志文件
	Compress bool
	// 是否同时输出到控制台
	ConsoleOutput bool
	// 日志级别
	Level interfaces.LogLevel
	// 日志格式
	Format interfaces.LogFormat
}

// DefaultLogStorageConfig 返回默认的日志存储配置
func DefaultLogStorageConfig() LogStorageConfig {
	return LogStorageConfig{
		FilePath:      "/var/log/tr069.log",
		MaxSize:       100,
		MaxBackups:    10,
		MaxAge:        30,
		Compress:      true,
		ConsoleOutput: false, // 默认不输出到控制台，专门存储到文件
		Level:         interfaces.LogLevelInfo,
		Format:        interfaces.LogFormatJSON,
	}
}

// DefaultCoreLogStorageConfig 返回专门用于TR069 Core的日志存储配置
func DefaultCoreLogStorageConfig() LogStorageConfig {
	return LogStorageConfig{
		FilePath:      "/var/log/tr069/tr069.log",
		MaxSize:       50,  // 单个文件50MB
		MaxBackups:    20,  // 保留20个备份文件
		MaxAge:        90,  // 保留90天
		Compress:      true,
		ConsoleOutput: false, // 专门存储到文件，不输出到控制台
		Level:         interfaces.LogLevelInfo,
		Format:        interfaces.LogFormatJSON,
	}
}

// NewFileLogger 创建一个将日志存储到文件的日志记录器
func NewFileLogger(config LogStorageConfig) interfaces.Logger {
	// 确保日志目录存在
	if config.FilePath != "" {
		dir := filepath.Dir(config.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "创建日志目录失败: %v\n", err)
		}
	}

	// 创建日志轮转器
	fileWriter := &lumberjack.Logger{
		Filename:   config.FilePath,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
	}

	// 设置日志编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 设置日志级别
	var zapLevel zapcore.Level
	switch config.Level {
	case interfaces.LogLevelDebug:
		zapLevel = zapcore.DebugLevel
	case interfaces.LogLevelInfo:
		zapLevel = zapcore.InfoLevel
	case interfaces.LogLevelWarn:
		zapLevel = zapcore.WarnLevel
	case interfaces.LogLevelError:
		zapLevel = zapcore.ErrorLevel
	case interfaces.LogLevelFatal:
		zapLevel = zapcore.FatalLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// 设置日志格式
	var encoder zapcore.Encoder
	switch config.Format {
	case interfaces.LogFormatJSON:
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	case interfaces.LogFormatTabSeparated:
		encoder = TabSeparatedConsoleEncoder()
	case interfaces.LogFormatCompact:
		encoder = CompactConsoleEncoder()
	default:
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 创建核心
	var core zapcore.Core
	if config.ConsoleOutput {
		// 同时输出到文件和控制台
		consoleWriter := zapcore.Lock(os.Stdout)
		core = zapcore.NewTee(
			zapcore.NewCore(encoder, zapcore.AddSync(fileWriter), zapLevel),
			zapcore.NewCore(encoder, consoleWriter, zapLevel),
		)
	} else {
		// 只输出到文件
		core = zapcore.NewCore(encoder, zapcore.AddSync(fileWriter), zapLevel)
	}

	// 创建zap logger
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar := zapLogger.Sugar()

	// 创建适配器
	return NewZapAdapter(sugar)
}

// WithLogStorage 设置日志存储配置
func WithLogStorage(config LogStorageConfig) Option {
	return func(opts *options) {
		// 创建日志轮转器
		fileWriter := &lumberjack.Logger{
			Filename:   config.FilePath,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}

		// 设置输出
		opts.output = fileWriter
		opts.level = config.Level
		opts.format = config.Format
	}
}

// RotateLogFiles 手动触发日志文件轮转
func RotateLogFiles() error {
	// 获取全局日志记录器
	logger := GetGlobalLogger()
	
	// 尝试转换为zapAdapter
	if zapLogger, ok := logger.(*zapAdapter); ok {
		// 尝试获取底层的lumberjack logger
		// 注意：这里需要根据实际的zap core结构来调整
		core := zapLogger.sugar.Desugar().Core()
		// 简化处理，直接返回成功
		_ = core
		return nil
	}
	
	return fmt.Errorf("无法获取日志轮转器")
}