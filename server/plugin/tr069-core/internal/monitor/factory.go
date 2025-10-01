// Package monitor provides implementation for monitoring functionality.
package monitor

import "github.com/root/demo/tr069/interfaces"

// NewMonitorFactory 创建监控实例的工厂函数
func NewMonitorFactory() func(options ...interfaces.MonitorOption) interfaces.Monitor {
	return func(options ...interfaces.MonitorOption) interfaces.Monitor {
		return NewMonitor(options...)
	}
}