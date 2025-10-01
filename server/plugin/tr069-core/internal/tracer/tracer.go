// Package tracer provides implementation for message tracing functionality.
package tracer

import (
"encoding/json"
"fmt"
"sync"
"time"

"github.com/root/demo/tr069/interfaces"
)

// defaultTracer 实现了interfaces.Tracer接口
type defaultTracer struct {
	mu               sync.RWMutex
	traces           map[interfaces.TraceID]map[interfaces.SpanID]*interfaces.TraceSpan
	maxTraces        int
	maxSpansPerTrace int
}

// NewDefaultTracer 创建一个新的默认追踪器
func NewDefaultTracer(options ...interfaces.TracerOption) interfaces.Tracer {
	tracer := &defaultTracer{
		traces:           make(map[interfaces.TraceID]map[interfaces.SpanID]*interfaces.TraceSpan),
		maxTraces:        1000,
		maxSpansPerTrace: 100,
	}

	for _, option := range options {
		option(tracer)
	}

	return tracer
}

// StartTrace 开始一个新的追踪
func (t *defaultTracer) StartTrace(name string) interfaces.TraceID {
t.mu.Lock()
defer t.mu.Unlock()

// 生成唯一的追踪ID
traceID := interfaces.TraceID(fmt.Sprintf("%d", time.Now().UnixNano()))

// 初始化追踪
t.traces[traceID] = make(map[interfaces.SpanID]*interfaces.TraceSpan)

// 创建根跨度
rootSpanID := interfaces.SpanID(fmt.Sprintf("%d", time.Now().UnixNano()))
rootSpan := &interfaces.TraceSpan{
SpanID:      rootSpanID,
ParentSpanID: "",
Name:        name,
StartTime:   time.Now(),
Events:      make([]interfaces.TraceEvent, 0),
Tags:        make(map[string]string),
Metrics:     make(map[string]float64),
}

t.traces[traceID][rootSpanID] = rootSpan

return traceID
}

// StartSpan 开始一个新的跨度
func (t *defaultTracer) StartSpan(traceID interfaces.TraceID, name string, parentSpanID interfaces.SpanID) interfaces.SpanID {
t.mu.Lock()
defer t.mu.Unlock()

// 检查追踪是否存在
spans, exists := t.traces[traceID]
if !exists {
return ""
}

// 生成唯一的跨度ID
spanID := interfaces.SpanID(fmt.Sprintf("%d", time.Now().UnixNano()))

// 创建新跨度
span := &interfaces.TraceSpan{
SpanID:      spanID,
ParentSpanID: parentSpanID,
Name:        name,
StartTime:   time.Now(),
Events:      make([]interfaces.TraceEvent, 0),
Tags:        make(map[string]string),
Metrics:     make(map[string]float64),
}

spans[spanID] = span

return spanID
}

// EndSpan 结束一个跨度
func (t *defaultTracer) EndSpan(traceID interfaces.TraceID, spanID interfaces.SpanID) {
t.mu.Lock()
defer t.mu.Unlock()

// 检查追踪和跨度是否存在
spans, exists := t.traces[traceID]
if !exists {
return
}

span, exists := spans[spanID]
if !exists {
return
}

// 设置结束时间
span.EndTime = time.Now()

// 计算处理时间并添加为指标
duration := span.EndTime.Sub(span.StartTime).Milliseconds()
span.Metrics["duration_ms"] = float64(duration)
}

// AddEvent 添加事件
func (t *defaultTracer) AddEvent(traceID interfaces.TraceID, spanID interfaces.SpanID, eventType string, data map[string]interface{}) {
t.mu.Lock()
defer t.mu.Unlock()

// 检查追踪和跨度是否存在
spans, exists := t.traces[traceID]
if !exists {
return
}

span, exists := spans[spanID]
if !exists {
return
}

// 创建事件
event := interfaces.TraceEvent{
EventType: eventType,
Timestamp: time.Now(),
Data:      data,
}

// 添加事件到跨度
span.Events = append(span.Events, event)
}

// AddTag 添加标签
func (t *defaultTracer) AddTag(traceID interfaces.TraceID, spanID interfaces.SpanID, key, value string) {
t.mu.Lock()
defer t.mu.Unlock()

// 检查追踪和跨度是否存在
spans, exists := t.traces[traceID]
if !exists {
return
}

span, exists := spans[spanID]
if !exists {
return
}

// 添加标签
span.Tags[key] = value
}

// AddMetric 添加指标
func (t *defaultTracer) AddMetric(traceID interfaces.TraceID, spanID interfaces.SpanID, key string, value float64) {
t.mu.Lock()
defer t.mu.Unlock()

// 检查追踪和跨度是否存在
spans, exists := t.traces[traceID]
if !exists {
return
}

span, exists := spans[spanID]
if !exists {
return
}

// 添加指标
span.Metrics[key] = value
}

// GetTrace 获取完整追踪信息
func (t *defaultTracer) GetTrace(traceID interfaces.TraceID) []interfaces.TraceSpan {
t.mu.RLock()
defer t.mu.RUnlock()

// 检查追踪是否存在
spans, exists := t.traces[traceID]
if !exists {
return nil
}

// 转换为切片
result := make([]interfaces.TraceSpan, 0, len(spans))
for _, span := range spans {
result = append(result, *span)
}

return result
}

// ExportTrace 导出追踪信息
func (t *defaultTracer) ExportTrace(traceID interfaces.TraceID, format string) ([]byte, error) {
trace := t.GetTrace(traceID)
if trace == nil {
return nil, fmt.Errorf("trace not found: %s", traceID)
}

switch format {
case "json":
return json.Marshal(trace)
default:
return nil, fmt.Errorf("unsupported export format: %s", format)
}
}

// ListTraces 列出所有追踪
func (t *defaultTracer) ListTraces() []interfaces.TraceID {
t.mu.RLock()
defer t.mu.RUnlock()

result := make([]interfaces.TraceID, 0, len(t.traces))
for traceID := range t.traces {
result = append(result, traceID)
}

return result
}

// ClearTrace 清除追踪信息
func (t *defaultTracer) ClearTrace(traceID interfaces.TraceID) {
t.mu.Lock()
defer t.mu.Unlock()

delete(t.traces, traceID)
}
