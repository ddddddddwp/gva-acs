// Package event implements the TR069 event notification mechanism.
// 包 event 实现了 TR069 事件通知机制。
package event

import (
	"context"
	"sync"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// subscriber implements the EventSubscriber interface.
// subscriber 实现了 EventSubscriber 接口。
type subscriber struct {
	subscriptions map[interfaces.EventType]interfaces.EventCallback
	mutex         sync.RWMutex
}

// NewSubscriber creates a new event subscriber.
// NewSubscriber 创建一个新的事件订阅器。
func NewSubscriber() interfaces.EventSubscriber {
	return &subscriber{
		subscriptions: make(map[interfaces.EventType]interfaces.EventCallback),
	}
}

// Subscribe subscribes to events of a specific type.
// Subscribe 订阅特定类型的事件。
func (s *subscriber) Subscribe(ctx context.Context, eventType interfaces.EventType, callback interfaces.EventCallback) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.subscriptions[eventType] = callback
	return nil
}

// Unsubscribe unsubscribes from events of a specific type.
// Unsubscribe 取消订阅特定类型的事件。
func (s *subscriber) Unsubscribe(ctx context.Context, eventType interfaces.EventType) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.subscriptions, eventType)
	return nil
}

// ListSubscriptions returns a list of currently subscribed event types.
// ListSubscriptions 返回当前订阅的事件类型列表。
func (s *subscriber) ListSubscriptions() []interfaces.EventType {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	types := make([]interfaces.EventType, 0, len(s.subscriptions))
	for eventType := range s.subscriptions {
		types = append(types, eventType)
	}
	return types
}

// IsSubscribed checks if a specific event type is subscribed.
// IsSubscribed 检查是否订阅了特定事件类型。
func (s *subscriber) IsSubscribed(eventType interfaces.EventType) bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	_, exists := s.subscriptions[eventType]
	return exists
}

// HandleEvent handles an incoming event by calling the appropriate callback.
// HandleEvent 通过调用适当的回调函数来处理传入的事件。
func (s *subscriber) HandleEvent(ctx context.Context, event *interfaces.Event) error {
	s.mutex.RLock()
	callback, exists := s.subscriptions[event.EventType]
	s.mutex.RUnlock()

	if !exists {
		return nil // No subscription for this event type
		// 没有订阅此事件类型
	}

	return callback(ctx, event)
}