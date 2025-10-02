// Package logger provides implementation for detailed logging functionality.
package logger

import "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"

// NewZapLoggerFactory 创建基于Zap的日志记录器工厂函数
func NewZapLoggerFactory() func(options ...interfaces.LoggerOption) interfaces.Logger {
	return func(options ...interfaces.LoggerOption) interfaces.Logger {
		return NewZapLogger(options...)
	}
}