// Package tracer provides implementation for message tracing functionality.
package tracer

import "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"

// NewTracerFactory 创建消息追踪器工厂函数
func NewTracerFactory() func(options ...interfaces.TracerOption) interfaces.Tracer {
return func(options ...interfaces.TracerOption) interfaces.Tracer {
return NewDefaultTracer(options...)
}
}
