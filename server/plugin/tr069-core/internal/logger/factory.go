// Package logger provides implementation for detailed logging functionality.
package logger

import "github.com/root/demo/tr069/interfaces"

// NewLoggerFactory 创建日志记录器工厂函数
func NewLoggerFactory() func(options ...interfaces.LoggerOption) interfaces.Logger {
return func(options ...interfaces.LoggerOption) interfaces.Logger {
return NewDefaultLogger(options...)
}
}
