// Package monitor provides implementation for monitoring functionality.
package monitor

import (
	"sync"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// defaultListener 实现了 interfaces.MonitorListener 接口
type defaultListener struct {
	callback func(interfaces.MetricValue)
	mutex    sync.RWMutex
}

// NewDefaultListener 创建一个默认的监听器
func NewDefaultListener(callback func(interfaces.MetricValue)) interfaces.MonitorListener {
	return &defaultListener{
		callback: callback,
	}
}

// OnMetricUpdate 当指标更新时调用
func (l *defaultListener) OnMetricUpdate(metric interfaces.MetricValue) {
	l.mutex.RLock()
	defer l.mutex.RUnlock()

	if l.callback != nil {
		l.callback(metric)
	}
}

// SetCallback 设置回调函数
func (l *defaultListener) SetCallback(callback func(interfaces.MetricValue)) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	l.callback = callback
}