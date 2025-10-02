// Package logger provides implementation for detailed logging functionality.
package logger

import "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"

// NewLoggerFactory 创建日志记录器工厂函数
func NewLoggerFactory() func(options ...interfaces.LoggerOption) interfaces.Logger {
return func(options ...interfaces.LoggerOption) interfaces.Logger {
return NewDefaultLogger(options...)
}
}
