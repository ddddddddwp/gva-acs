// Package logger provides implementation for detailed logging functionality.
package logger

import (
"io"
"github.com/root/demo/tr069/interfaces"
)

// WithLogLevel 设置日志级别
func WithLogLevel(level interfaces.LogLevel) interfaces.LoggerOption {
return func(logger interfaces.Logger) {
logger.SetLevel(level)
}
}

// WithLogFormat 设置日志格式
func WithLogFormat(format interfaces.LogFormat) interfaces.LoggerOption {
return func(logger interfaces.Logger) {
logger.SetFormat(format)
}
}

// WithLogOutput 设置日志输出目标
func WithLogOutput(output io.Writer) interfaces.LoggerOption {
return func(logger interfaces.Logger) {
logger.SetOutput(output)
}
}

// WithLogFields 设置默认日志字段
func WithLogFields(fields ...interfaces.LogField) interfaces.LoggerOption {
return func(logger interfaces.Logger) {
for _, field := range fields {
logger = logger.WithFields(field)
}
}
}

// WithLogContext 设置日志上下文
func WithLogContext(ctx interface{}) interfaces.LoggerOption {
return func(logger interfaces.Logger) {
logger.WithContext(ctx)
}
}
