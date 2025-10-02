// Package monitor provides implementation for monitoring functionality.
package monitor

import (
"time"

"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// WithExportInterval 设置导出间隔的选项
func WithExportInterval(interval time.Duration) interfaces.MonitorOption {
return func(m interfaces.Monitor) {
if monitor, ok := m.(*monitor); ok {
monitor.SetExportInterval(interval)
}
}
}

// WithMetricPrefix 设置指标前缀的选项
func WithMetricPrefix(prefix string) interfaces.MonitorOption {
return func(m interfaces.Monitor) {
if monitor, ok := m.(*monitor); ok {
monitor.SetMetricPrefix(prefix)
}
}
}

// WithHistogramBuckets 设置直方图桶的选项
func WithHistogramBuckets(metricName string, buckets []float64) interfaces.MonitorOption {
return func(m interfaces.Monitor) {
if monitor, ok := m.(*monitor); ok {
monitor.SetHistogramBuckets(metricName, buckets)
}
}
}
