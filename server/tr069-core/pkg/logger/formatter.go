package logger

import (
	"fmt"
	"time"

	"go.uber.org/zap/zapcore"
)

// CustomTimeEncoder 自定义时间编码器，输出RFC3339Nano格式（包含毫秒）
func CustomTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	// 格式：2025-09-25T08:02:35.464Z
	enc.AppendString(t.UTC().Format("2006-01-02T15:04:05.000Z"))
}

// TabSeparatedConsoleEncoder 创建Tab分隔的控制台编码器
func TabSeparatedConsoleEncoder() zapcore.Encoder {
	config := zapcore.EncoderConfig{
		TimeKey:          "time",
		LevelKey:         "level",
		NameKey:          "logger",
		CallerKey:        "caller",
		FunctionKey:      zapcore.OmitKey,
		MessageKey:       "msg",
		StacktraceKey:    "stacktrace",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeLevel:      CustomLevelEncoder,
		EncodeTime:       CustomTimeEncoder,
		EncodeDuration:   zapcore.StringDurationEncoder,
		EncodeCaller:     CustomCallerEncoder,
		ConsoleSeparator: "\t", // 使用Tab分隔
	}
	return zapcore.NewConsoleEncoder(config)
}

// CustomLevelEncoder 自定义级别编码器，输出大写级别名
func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(level.CapitalString())
}

// CustomCallerEncoder 自定义调用者编码器，输出文件路径和行号
func CustomCallerEncoder(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	if !caller.Defined {
		enc.AppendString("undefined")
		return
	}
	// 输出格式：path/file.go:line
	enc.AppendString(fmt.Sprintf("%s:%d", caller.File, caller.Line))
}

// CompactConsoleEncoder 创建紧凑格式的控制台编码器
// 格式：2025-09-25T08:02:35.464Z INFO inverters/invertersallinone.go:82 message
func CompactConsoleEncoder() zapcore.Encoder {
	config := zapcore.EncoderConfig{
		TimeKey:          "time",
		LevelKey:         "level",
		NameKey:          zapcore.OmitKey,
		CallerKey:        "caller",
		FunctionKey:      zapcore.OmitKey,
		MessageKey:       "msg",
		StacktraceKey:    zapcore.OmitKey,
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeLevel:      CustomLevelEncoder,
		EncodeTime:       CustomTimeEncoder,
		EncodeDuration:   zapcore.StringDurationEncoder,
		EncodeCaller:     CustomCallerEncoder,
		ConsoleSeparator: "\t", // 使用Tab分隔
	}
	return zapcore.NewConsoleEncoder(config)
}
