package test

import (
	"fmt"
	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/internal/monitor"
	"testing"
	"time"
)

func TestStatusMonitor(t *testing.T) {
// 创建状态监控器
factory := monitor.NewStatusMonitorFactory()
statusMonitor := factory(
monitor.WithMonitorInterval(100*time.Millisecond),
)

// 测试注册和获取状态
statusMonitor.RegisterStatus("connection", "connected")
if status := statusMonitor.GetStatus("connection"); status != "connected" {
t.Errorf("Expected status 'connected', got '%s'", status)
}

// 测试状态更新
statusMonitor.UpdateStatus("connection", "disconnected")
if status := statusMonitor.GetStatus("connection"); status != "disconnected" {
t.Errorf("Expected status 'disconnected', got '%s'", status)
}

// 测试监听器
statusChanged := false
listener := monitor.NewDefaultListener(func(key string, oldValue, newValue interface{}) {
if key == "connection" && oldValue == "disconnected" && newValue == "reconnecting" {
statusChanged = true
}
})

statusMonitor.AddListener(listener)
statusMonitor.UpdateStatus("connection", "reconnecting")

// 等待监听器处理
time.Sleep(200 * time.Millisecond)
if !statusChanged {
t.Error("Status listener was not called correctly")
}

// 测试移除监听器
statusChanged = false
statusMonitor.RemoveListener(listener)
statusMonitor.UpdateStatus("connection", "connected")
time.Sleep(200 * time.Millisecond)
if statusChanged {
t.Error("Status listener was called after removal")
}

// 测试获取所有状态
statusMonitor.RegisterStatus("device", "online")
allStatus := statusMonitor.GetAllStatus()
if len(allStatus) != 2 {
t.Errorf("Expected 2 status entries, got %d", len(allStatus))
}
if allStatus["connection"] != "connected" || allStatus["device"] != "online" {
t.Error("GetAllStatus returned incorrect values")
}

fmt.Println("StatusMonitor tests passed")
}
