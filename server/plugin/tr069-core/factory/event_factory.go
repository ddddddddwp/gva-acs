// Package factory provides factory functions for creating TR069 components.
// 包 factory 提供创建 TR069 组件的工厂函数。
package factory

import (
	"github.com/root/demo/tr069/internal/event"
	"github.com/root/demo/tr069/interfaces"
)

// CreateEventNotifier creates a new event notifier.
// CreateEventNotifier 创建一个新的事件通知器。
func CreateEventNotifier() interfaces.EventNotifier {
	return event.NewNotifier()
}

// CreateEventSubscriber creates a new event subscriber.
// CreateEventSubscriber 创建一个新的事件订阅器。
func CreateEventSubscriber() interfaces.EventSubscriber {
	return event.NewSubscriber()
}

// CreateEventQueueManager creates a new event queue manager.
// CreateEventQueueManager 创建一个新的事件队列管理器。
func CreateEventQueueManager(notifier interfaces.EventNotifier) interfaces.EventQueueManager {
	return event.NewQueueManager(notifier)
}