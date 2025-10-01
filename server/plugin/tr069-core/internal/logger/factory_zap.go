// Package logger provides implementation for detailed logging functionality.
package logger

import "github.com/root/demo/tr069/interfaces"

// NewZapLoggerFactory 创建基于Zap的日志记录器工厂函数
func NewZapLoggerFactory() func(options ...interfaces.LoggerOption) interfaces.Logger {
	return func(options ...interfaces.LoggerOption) interfaces.Logger {
		return NewZapLogger(options...)
	}
}