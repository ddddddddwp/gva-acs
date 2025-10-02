// Package interfaces defines the public interfaces for the TR069 library.
package interfaces

import (
	"time"
)

// MetricType 定义了指标类型
type MetricType string

const (
	// MetricTypeCounter 计数器类型指标，只增不减
	MetricTypeCounter MetricType = "counter"
	// MetricTypeGauge 仪表类型指标，可增可减
	MetricTypeGauge MetricType = "gauge"
	// MetricTypeHistogram 直方图类型指标，用于统计分布情况
	MetricTypeHistogram MetricType = "histogram"
)

// MetricValue 定义了指标值
type MetricValue struct {
	// Name 指标名称
	Name string
	// Type 指标类型
	Type MetricType
	// Value 指标值
	Value float64
	// Labels 指标标签
	Labels map[string]string
	// Timestamp 指标时间戳
	Timestamp time.Time
}

// MetricSnapshot 定义了指标快照
type MetricSnapshot struct {
	// Metrics 指标列表
	Metrics []MetricValue
	// Timestamp 快照时间戳
	Timestamp time.Time
}

// MonitorListener 定义了监控监听器接口
type MonitorListener interface {
	// OnMetricUpdate 当指标更新时调用
	OnMetricUpdate(metric MetricValue)
}

// Monitor 定义了监控接口
type Monitor interface {
	// RegisterMetric 注册一个指标
	RegisterMetric(name string, metricType MetricType, initialValue float64, labels map[string]string) error

	// UpdateMetric 更新指标值
	UpdateMetric(name string, value float64) error

	// IncrementMetric 增加指标值
	IncrementMetric(name string, delta float64) error

	// GetMetric 获取指标值
	GetMetric(name string) (*MetricValue, error)

	// GetAllMetrics 获取所有指标
	GetAllMetrics() []MetricValue

	// GetSnapshot 获取指标快照
	GetSnapshot() *MetricSnapshot

	// ExportMetrics 导出指标数据
	ExportMetrics(format string) ([]byte, error)

	// AddListener 添加监听器
	AddListener(listener MonitorListener)

	// RemoveListener 移除监听器
	RemoveListener(listener MonitorListener)

	// Start 启动监控
	Start() error

	// Stop 停止监控
	Stop() error
}

// MonitorOption 定义了监控选项函数
type MonitorOption func(Monitor)

// WithExportInterval 设置导出间隔
func WithExportInterval(interval time.Duration) MonitorOption {
	return func(m Monitor) {
		if mm, ok := m.(interface{ SetExportInterval(time.Duration) }); ok {
			mm.SetExportInterval(interval)
		}
	}
}

// WithMetricPrefix 设置指标前缀
func WithMetricPrefix(prefix string) MonitorOption {
	return func(m Monitor) {
		if mm, ok := m.(interface{ SetMetricPrefix(string) }); ok {
			mm.SetMetricPrefix(prefix)
		}
	}
}

// WithHistogramBuckets 设置直方图桶
func WithHistogramBuckets(metricName string, buckets []float64) MonitorOption {
	return func(m Monitor) {
		if mm, ok := m.(interface{ SetHistogramBuckets(string, []float64) }); ok {
			mm.SetHistogramBuckets(metricName, buckets)
		}
	}
}
