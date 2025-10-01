// Package tracer provides implementation for message tracing functionality.
package tracer

import "github.com/root/demo/tr069/interfaces"

// WithMaxTraces 设置最大追踪数量
func WithMaxTraces(maxTraces int) interfaces.TracerOption {
return func(tracer interfaces.Tracer) {
if t, ok := tracer.(*defaultTracer); ok {
t.maxTraces = maxTraces
}
}
}

// WithMaxSpansPerTrace 设置每个追踪的最大跨度数量
func WithMaxSpansPerTrace(maxSpans int) interfaces.TracerOption {
return func(tracer interfaces.Tracer) {
if t, ok := tracer.(*defaultTracer); ok {
t.maxSpansPerTrace = maxSpans
}
}
}
