// Package monitor provides implementation for monitoring functionality.
package monitor

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// monitor 实现了 interfaces.Monitor 接口
type monitor struct {
	metrics        map[string]*metricData
	metricsMutex   sync.RWMutex
	listeners      []interfaces.MonitorListener
	listenersMutex sync.RWMutex
	exportInterval time.Duration
	metricPrefix   string
	histogramData  map[string][]float64
	stopChan       chan struct{}
	wg             sync.WaitGroup
	started        bool
	startedMutex   sync.RWMutex
}

// metricData 存储指标数据
type metricData struct {
	name       string
	metricType interfaces.MetricType
	value      float64
	labels     map[string]string
	timestamp  time.Time
	mutex      sync.RWMutex
}

// NewMonitor 创建一个新的监控实例
func NewMonitor(options ...interfaces.MonitorOption) interfaces.Monitor {
	m := &monitor{
		metrics:        make(map[string]*metricData),
		listeners:      make([]interfaces.MonitorListener, 0),
		exportInterval: 60 * time.Second, // 默认导出间隔为60秒
		histogramData:  make(map[string][]float64),
		stopChan:       make(chan struct{}),
	}

	// 应用选项
	for _, option := range options {
		option(m)
	}

	return m
}

// RegisterMetric 注册一个指标
func (m *monitor) RegisterMetric(name string, metricType interfaces.MetricType, initialValue float64, labels map[string]string) error {
	if name == "" {
		return fmt.Errorf("metric name cannot be empty")
	}

	// 如果有前缀，添加前缀
	if m.metricPrefix != "" {
		name = m.metricPrefix + "." + name
	}

	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()

	// 检查是否已存在
	if _, exists := m.metrics[name]; exists {
		return fmt.Errorf("metric %s already registered", name)
	}

	// 创建标签的副本
	labelsCopy := make(map[string]string)
	if labels != nil {
		for k, v := range labels {
			labelsCopy[k] = v
		}
	}

	// 创建并存储指标
	m.metrics[name] = &metricData{
		name:       name,
		metricType: metricType,
		value:      initialValue,
		labels:     labelsCopy,
		timestamp:  time.Now(),
	}

	return nil
}

// UpdateMetric 更新指标值
func (m *monitor) UpdateMetric(name string, value float64) error {
	// 如果有前缀，添加前缀
	if m.metricPrefix != "" {
		name = m.metricPrefix + "." + name
	}

	m.metricsMutex.RLock()
	metric, exists := m.metrics[name]
	m.metricsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("metric %s not found", name)
	}

	metric.mutex.Lock()
	metric.value = value
	metric.timestamp = time.Now()
	metric.mutex.Unlock()

	// 通知监听器
	m.notifyListeners(name)

	return nil
}

// IncrementMetric 增加指标值
func (m *monitor) IncrementMetric(name string, delta float64) error {
	// 如果有前缀，添加前缀
	if m.metricPrefix != "" {
		name = m.metricPrefix + "." + name
	}

	m.metricsMutex.RLock()
	metric, exists := m.metrics[name]
	m.metricsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("metric %s not found", name)
	}

	metric.mutex.Lock()
	metric.value += delta
	metric.timestamp = time.Now()
	metric.mutex.Unlock()

	// 通知监听器
	m.notifyListeners(name)

	return nil
}

// GetMetric 获取指标值
func (m *monitor) GetMetric(name string) (*interfaces.MetricValue, error) {
	// 如果有前缀，添加前缀
	if m.metricPrefix != "" {
		name = m.metricPrefix + "." + name
	}

	m.metricsMutex.RLock()
	metric, exists := m.metrics[name]
	m.metricsMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("metric %s not found", name)
	}

	metric.mutex.RLock()
	defer metric.mutex.RUnlock()

	// 创建标签的副本
	labelsCopy := make(map[string]string)
	for k, v := range metric.labels {
		labelsCopy[k] = v
	}

	return &interfaces.MetricValue{
		Name:      metric.name,
		Type:      metric.metricType,
		Value:     metric.value,
		Labels:    labelsCopy,
		Timestamp: metric.timestamp,
	}, nil
}

// GetAllMetrics 获取所有指标
func (m *monitor) GetAllMetrics() []interfaces.MetricValue {
	m.metricsMutex.RLock()
	defer m.metricsMutex.RUnlock()

	metrics := make([]interfaces.MetricValue, 0, len(m.metrics))

	for _, metric := range m.metrics {
		metric.mutex.RLock()

		// 创建标签的副本
		labelsCopy := make(map[string]string)
		for k, v := range metric.labels {
			labelsCopy[k] = v
		}

		metrics = append(metrics, interfaces.MetricValue{
			Name:      metric.name,
			Type:      metric.metricType,
			Value:     metric.value,
			Labels:    labelsCopy,
			Timestamp: metric.timestamp,
		})

		metric.mutex.RUnlock()
	}

	return metrics
}

// GetSnapshot 获取指标快照
func (m *monitor) GetSnapshot() *interfaces.MetricSnapshot {
	metrics := m.GetAllMetrics()
	return &interfaces.MetricSnapshot{
		Metrics:   metrics,
		Timestamp: time.Now(),
	}
}

// ExportMetrics 导出指标数据
func (m *monitor) ExportMetrics(format string) ([]byte, error) {
	snapshot := m.GetSnapshot()

	switch format {
	case "json":
		return json.Marshal(snapshot)
	default:
		return json.Marshal(snapshot) // 默认使用JSON格式
	}
}

// AddListener 添加监听器
func (m *monitor) AddListener(listener interfaces.MonitorListener) {
	if listener == nil {
		return
	}

	m.listenersMutex.Lock()
	defer m.listenersMutex.Unlock()

	m.listeners = append(m.listeners, listener)
}

// RemoveListener 移除监听器
func (m *monitor) RemoveListener(listener interfaces.MonitorListener) {
	if listener == nil {
		return
	}

	m.listenersMutex.Lock()
	defer m.listenersMutex.Unlock()

	for i, l := range m.listeners {
		if l == listener {
			m.listeners = append(m.listeners[:i], m.listeners[i+1:]...)
			break
		}
	}
}

// Start 启动监控
func (m *monitor) Start() error {
	m.startedMutex.Lock()
	defer m.startedMutex.Unlock()

	if m.started {
		return fmt.Errorf("monitor already started")
	}

	m.started = true
	m.stopChan = make(chan struct{})

	// 启动定期导出指标的协程
	m.wg.Add(1)
	go m.exportLoop()

	return nil
}

// Stop 停止监控
func (m *monitor) Stop() error {
	m.startedMutex.Lock()
	defer m.startedMutex.Unlock()

	if !m.started {
		return nil
	}

	close(m.stopChan)
	m.wg.Wait()
	m.started = false

	return nil
}

// exportLoop 定期导出指标
func (m *monitor) exportLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.exportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 这里可以实现定期导出指标的逻辑
			// 例如，将指标导出到文件或发送到远程服务器
		case <-m.stopChan:
			return
		}
	}
}

// notifyListeners 通知所有监听器指标更新
func (m *monitor) notifyListeners(metricName string) {
	metricValue, err := m.GetMetric(metricName)
	if err != nil {
		return
	}

	m.listenersMutex.RLock()
	listeners := make([]interfaces.MonitorListener, len(m.listeners))
	copy(listeners, m.listeners)
	m.listenersMutex.RUnlock()

	for _, listener := range listeners {
		go listener.OnMetricUpdate(*metricValue)
	}
}

// SetExportInterval 设置导出间隔
func (m *monitor) SetExportInterval(interval time.Duration) {
	m.exportInterval = interval
}

// SetMetricPrefix 设置指标前缀
func (m *monitor) SetMetricPrefix(prefix string) {
	m.metricPrefix = prefix
}

// SetHistogramBuckets 设置直方图桶
func (m *monitor) SetHistogramBuckets(metricName string, buckets []float64) {
	m.histogramData[metricName] = buckets
}
