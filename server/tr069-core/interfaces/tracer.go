// Package interfaces defines the interfaces for TR069 protocol library.
package interfaces

import "time"

// TraceID 定义追踪ID类型
type TraceID string

// SpanID 定义跨度ID类型
type SpanID string

// TraceEvent 定义追踪事件类型
type TraceEvent struct {
// EventType 事件类型
EventType string
// Timestamp 事件时间戳
Timestamp time.Time
// Data 事件相关数据
Data map[string]interface{}
}

// TraceSpan 定义追踪跨度
type TraceSpan struct {
// SpanID 跨度ID
SpanID SpanID
// ParentSpanID 父跨度ID
ParentSpanID SpanID
// Name 跨度名称
Name string
// StartTime 开始时间
StartTime time.Time
// EndTime 结束时间
EndTime time.Time
// Events 事件列表
Events []TraceEvent
// Tags 标签
Tags map[string]string
// Metrics 指标
Metrics map[string]float64
}

// Tracer 定义消息追踪接口
type Tracer interface {
// StartTrace 开始一个新的追踪
StartTrace(name string) TraceID

// StartSpan 开始一个新的跨度
StartSpan(traceID TraceID, name string, parentSpanID SpanID) SpanID

// EndSpan 结束一个跨度
EndSpan(traceID TraceID, spanID SpanID)

// AddEvent 添加事件
AddEvent(traceID TraceID, spanID SpanID, eventType string, data map[string]interface{})

// AddTag 添加标签
AddTag(traceID TraceID, spanID SpanID, key, value string)

// AddMetric 添加指标
AddMetric(traceID TraceID, spanID SpanID, key string, value float64)

// GetTrace 获取完整追踪信息
GetTrace(traceID TraceID) []TraceSpan

// ExportTrace 导出追踪信息
ExportTrace(traceID TraceID, format string) ([]byte, error)

// ListTraces 列出所有追踪
ListTraces() []TraceID

// ClearTrace 清除追踪信息
ClearTrace(traceID TraceID)
}

// TracerOption 定义追踪器配置选项
type TracerOption func(Tracer)
