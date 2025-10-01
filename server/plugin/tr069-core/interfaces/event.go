// Package interfaces defines the public interfaces for the TR069 library.
// 包 interfaces 定义了 TR069 库的公共接口。
package interfaces

import (
	"context"
	"time"
)

// EventType represents the type of TR-069 event
// EventType 表示 TR-069 事件类型
type EventType string

// Common TR-069 event types
// 常见的 TR-069 事件类型
const (
	EventBoot            EventType = "0 BOOTSTRAP"
	EventPeriodic        EventType = "1 PERIODIC"
	EventScheduled       EventType = "2 SCHEDULED"
	EventValueChange     EventType = "4 VALUE CHANGE"
	EventKicked          EventType = "5 KICKED"
	EventConnectionReady EventType = "6 CONNECTION REQUEST"
	EventTransferComplete EventType = "7 TRANSFER COMPLETE"
	EventDiagnosticComplete EventType = "8 DIAGNOSTIC COMPLETE"
	EventRequestDownload  EventType = "9 REQUEST DOWNLOAD"
	EventAutonomousTransferComplete EventType = "10 AUTONOMOUS TRANSFER COMPLETE"
	EventDUStateChangeComplete EventType = "11 DU STATE CHANGE COMPLETE"
	EventAutonomousDUStateChangeComplete EventType = "12 AUTONOMOUS DU STATE CHANGE COMPLETE"
)

// Event represents a TR-069 event notification
// Event 表示一个 TR-069 事件通知
type Event struct {
	// EventType is the type of the event
	// EventType 是事件的类型
	EventType EventType `json:"eventType"`

	// EventCode is the code of the event (same as EventType for standard events)
	// EventCode 是事件的代码（对于标准事件与 EventType 相同）
	EventCode string `json:"eventCode"`

	// CommandKey is an optional identifier for the event
	// CommandKey 是事件的可选标识符
	CommandKey string `json:"commandKey,omitempty"`

	// Timestamp is when the event occurred
	// Timestamp 是事件发生的时间
	Timestamp time.Time `json:"timestamp"`

	// DeviceID is the identifier of the device that generated the event
	// DeviceID 是生成事件的设备标识符
	DeviceID string `json:"deviceId"`

	// Parameters are any additional parameters associated with the event
	// Parameters 是与事件关联的任何附加参数
	Parameters map[string]interface{} `json:"parameters,omitempty"`

	// RetryCount is the number of times this event has been retried
	// RetryCount 是此事件已重试的次数
	RetryCount int `json:"retryCount"`

	// MaxRetries is the maximum number of retries allowed for this event
	// MaxRetries 是此事件允许的最大重试次数
	MaxRetries int `json:"maxRetries"`
}

// EventNotifier is the interface for sending event notifications from CPE to ACS
// EventNotifier 是从 CPE 向 ACS 发送事件通知的接口
type EventNotifier interface {
	// Notify sends an event notification to the ACS
	// Notify 向 ACS 发送事件通知
	Notify(ctx context.Context, event *Event) error

	// NotifyBatch sends a batch of event notifications to the ACS
	// NotifyBatch 向 ACS 发送一批事件通知
	NotifyBatch(ctx context.Context, events []*Event) error

	// SetNotificationEndpoint sets the ACS endpoint for event notifications
	// SetNotificationEndpoint 设置事件通知的 ACS 端点
	SetNotificationEndpoint(endpoint string)

	// GetNotificationEndpoint returns the current ACS endpoint for event notifications
	// GetNotificationEndpoint 返回事件通知的当前 ACS 端点
	GetNotificationEndpoint() string

	// SetRetryPolicy sets the retry policy for failed notifications
	// SetRetryPolicy 设置失败通知的重试策略
	SetRetryPolicy(maxRetries int, retryInterval time.Duration)

	// GetRetryPolicy returns the current retry policy
	// GetRetryPolicy 返回当前的重试策略
	GetRetryPolicy() (maxRetries int, retryInterval time.Duration)
}

// EventSubscriber is the interface for managing event subscriptions
// EventSubscriber 是管理事件订阅的接口
type EventSubscriber interface {
	// Subscribe subscribes to events of a specific type
	// Subscribe 订阅特定类型的事件
	Subscribe(ctx context.Context, eventType EventType, callback EventCallback) error

	// Unsubscribe unsubscribes from events of a specific type
	// Unsubscribe 取消订阅特定类型的事件
	Unsubscribe(ctx context.Context, eventType EventType) error

	// ListSubscriptions returns a list of currently subscribed event types
	// ListSubscriptions 返回当前订阅的事件类型列表
	ListSubscriptions() []EventType

	// IsSubscribed checks if a specific event type is subscribed
	// IsSubscribed 检查是否订阅了特定事件类型
	IsSubscribed(eventType EventType) bool
}

// EventCallback is a function type for handling received events
// EventCallback 是处理接收到的事件的函数类型
type EventCallback func(ctx context.Context, event *Event) error

// EventQueueManager is the interface for managing event queues
// EventQueueManager 是管理事件队列的接口
type EventQueueManager interface {
	// Enqueue adds an event to the queue
	// Enqueue 将事件添加到队列中
	Enqueue(event *Event) error

	// Dequeue removes and returns the next event from the queue
	// Dequeue 从队列中移除并返回下一个事件
	Dequeue() (*Event, error)

	// Peek returns the next event from the queue without removing it
	// Peek 返回队列中的下一个事件但不移除它
	Peek() (*Event, error)

	// Size returns the number of events in the queue
	// Size 返回队列中的事件数量
	Size() int

	// IsEmpty checks if the queue is empty
	// IsEmpty 检查队列是否为空
	IsEmpty() bool

	// Clear removes all events from the queue
	// Clear 从队列中移除所有事件
	Clear() error

	// SetMaxSize sets the maximum size of the queue
	// SetMaxSize 设置队列的最大大小
	SetMaxSize(maxSize int)

	// GetMaxSize returns the maximum size of the queue
	// GetMaxSize 返回队列的最大大小
	GetMaxSize() int
}