// Package monitor provides implementation for monitoring functionality.
package monitor

import "github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"

// NewMonitorFactory 创建监控实例的工厂函数
func NewMonitorFactory() func(options ...interfaces.MonitorOption) interfaces.Monitor {
	return func(options ...interfaces.MonitorOption) interfaces.Monitor {
		return NewMonitor(options...)
	}
}