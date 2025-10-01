// Package event implements the TR069 event notification mechanism.
// 包 event 实现了 TR069 事件通知机制。
package event

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/root/demo/tr069/interfaces"
)

// TestNotifier_Notify tests the Notify method of the notifier.
// TestNotifier_Notify 测试通知器的 Notify 方法。
func TestNotifier_Notify(t *testing.T) {
	// Create a test server to receive notifications
	// 创建一个测试服务器来接收通知
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method
		// 检查请求方法
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		// Check content type
		// 检查内容类型
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		
		// Send response
		// 发送响应
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a notifier
	// 创建通知器
	notifier := NewNotifier()
	
	// Set the notification endpoint
	// 设置通知端点
	notifier.SetNotificationEndpoint(server.URL)
	
	// Create a test event
	// 创建测试事件
	event := &interfaces.Event{
		EventType: interfaces.EventTypeBoot,
		EventCode: "0 BOOTSTRAP",
		CommandKey: "test-key",
		FaultCode:  0,
		FaultString: "",
		TimeStamp:  time.Now(),
		RetryCount: 0,
		MaxRetries: 3,
	}
	
	// Send notification
	// 发送通知
	ctx := context.Background()
	err := notifier.Notify(ctx, event)
	if err != nil {
		t.Errorf("Notify failed: %v", err)
	}
}

// TestNotifier_NotifyBatch tests the NotifyBatch method of the notifier.
// TestNotifier_NotifyBatch 测试通知器的 NotifyBatch 方法。
func TestNotifier_NotifyBatch(t *testing.T) {
	// Create a test server to receive notifications
	// 创建一个测试服务器来接收通知
	receivedCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment received count
		// 增加接收计数
		receivedCount++
		
		// Send response
		// 发送响应
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create a notifier
	// 创建通知器
	notifier := NewNotifier()
	
	// Set the notification endpoint
	// 设置通知端点
	notifier.SetNotificationEndpoint(server.URL)
	
	// Create test events
	// 创建测试事件
	events := []*interfaces.Event{
		{
			EventType: interfaces.EventTypeBoot,
			EventCode: "0 BOOTSTRAP",
			CommandKey: "test-key-1",
			FaultCode:  0,
			FaultString: "",
			TimeStamp:  time.Now(),
			RetryCount: 0,
			MaxRetries: 3,
		},
		{
			EventType: interfaces.EventTypePeriodic,
			EventCode: "1 PERIODIC",
			CommandKey: "test-key-2",
			FaultCode:  0,
			FaultString: "",
			TimeStamp:  time.Now(),
			RetryCount: 0,
			MaxRetries: 3,
		},
	}
	
	// Send batch notification
	// 发送批量通知
	ctx := context.Background()
	err := notifier.NotifyBatch(ctx, events)
	if err != nil {
		t.Errorf("NotifyBatch failed: %v", err)
	}
	
	// Check that all events were received
	// 检查是否收到了所有事件
	if receivedCount != len(events) {
		t.Errorf("Expected %d notifications, got %d", len(events), receivedCount)
	}
}

// TestNotifier_SetRetryPolicy tests the SetRetryPolicy method of the notifier.
// TestNotifier_SetRetryPolicy 测试通知器的 SetRetryPolicy 方法。
func TestNotifier_SetRetryPolicy(t *testing.T) {
	// Create a notifier
	// 创建通知器
	notifier := NewNotifier()
	
	// Set retry policy
	// 设置重试策略
	maxRetries := 5
	retryInterval := 10 * time.Second
	notifier.SetRetryPolicy(maxRetries, retryInterval)
	
	// Get retry policy
	// 获取重试策略
	actualMaxRetries, actualRetryInterval := notifier.GetRetryPolicy()
	
	// Check values
	// 检查值
	if actualMaxRetries != maxRetries {
		t.Errorf("Expected max retries %d, got %d", maxRetries, actualMaxRetries)
	}
	
	if actualRetryInterval != retryInterval {
		t.Errorf("Expected retry interval %v, got %v", retryInterval, actualRetryInterval)
	}
}