// Package event implements the TR069 event notification mechanism.
// 包 event 实现了 TR069 事件通知机制。
package event

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// notifier implements the EventNotifier interface.
// notifier 实现了 EventNotifier 接口。
type notifier struct {
	endpoint      string
	maxRetries    int
	retryInterval time.Duration
	httpClient    *http.Client
	mutex         sync.RWMutex
}

// NewNotifier creates a new event notifier.
// NewNotifier 创建一个新的事件通知器。
func NewNotifier() interfaces.EventNotifier {
	return &notifier{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries:    3,
		retryInterval: 5 * time.Second,
	}
}

// Notify sends an event notification to the ACS.
// Notify 向 ACS 发送事件通知。
func (n *notifier) Notify(ctx context.Context, event *interfaces.Event) error {
	// Convert event to JSON
	// 将事件转换为 JSON
	eventData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Get endpoint
	// 获取端点
	endpoint := n.GetNotificationEndpoint()
	if endpoint == "" {
		return fmt.Errorf("notification endpoint is not set")
	}

	// Send notification with retry logic
	// 使用重试逻辑发送通知
	var lastErr error
	maxRetries, retryInterval := n.GetRetryPolicy()
	for i := 0; i <= maxRetries; i++ {
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(eventData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := n.httpClient.Do(req)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Success
			// 成功
			resp.Body.Close()
			return nil
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			resp.Body.Close()
		}

		// Wait before retrying
		// 重试前等待
		if i < maxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryInterval):
			}
		}
	}

	maxRetries, _ = n.GetRetryPolicy()
	return fmt.Errorf("failed to send event notification after %d retries: %w", maxRetries, lastErr)
}

// NotifyBatch sends a batch of event notifications to the ACS.
// NotifyBatch 向 ACS 发送一批事件通知。
func (n *notifier) NotifyBatch(ctx context.Context, events []*interfaces.Event) error {
	// Send events one by one
	// 逐个发送事件
	for _, event := range events {
		if err := n.Notify(ctx, event); err != nil {
			return fmt.Errorf("failed to send event notification: %w", err)
		}
	}
	return nil
}

// SetNotificationEndpoint sets the ACS endpoint for event notifications.
// SetNotificationEndpoint 设置事件通知的 ACS 端点。
func (n *notifier) SetNotificationEndpoint(endpoint string) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.endpoint = endpoint
}

// GetNotificationEndpoint returns the current ACS endpoint for event notifications.
// GetNotificationEndpoint 返回事件通知的当前 ACS 端点。
func (n *notifier) GetNotificationEndpoint() string {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.endpoint
}

// SetRetryPolicy sets the retry policy for failed notifications.
// SetRetryPolicy 设置失败通知的重试策略。
func (n *notifier) SetRetryPolicy(maxRetries int, retryInterval time.Duration) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.maxRetries = maxRetries
	n.retryInterval = retryInterval
}

// GetRetryPolicy returns the current retry policy.
// GetRetryPolicy 返回当前的重试策略。
func (n *notifier) GetRetryPolicy() (maxRetries int, retryInterval time.Duration) {
	n.mutex.RLock()
	defer n.mutex.RUnlock()
	return n.maxRetries, n.retryInterval
}