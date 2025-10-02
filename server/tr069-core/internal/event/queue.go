// Package event implements the TR069 event notification mechanism.
// 包 event 实现了 TR069 事件通知机制。
package event

import (
	"container/list"
	"context"
	"sync"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// queueManager implements the EventQueueManager interface.
// queueManager 实现了 EventQueueManager 接口。
type queueManager struct {
	queue    *list.List
	maxSize  int
	mutex    sync.Mutex
	notifier interfaces.EventNotifier
}

// NewQueueManager creates a new event queue manager.
// NewQueueManager 创建一个新的事件队列管理器。
func NewQueueManager(notifier interfaces.EventNotifier) interfaces.EventQueueManager {
	return &queueManager{
		queue:    list.New(),
		maxSize:  1000, // Default maximum size
		// 默认最大大小
		notifier: notifier,
	}
}

// Enqueue adds an event to the queue.
// Enqueue 将事件添加到队列中。
func (q *queueManager) Enqueue(event *interfaces.Event) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// Check if queue is at maximum size
	// 检查队列是否达到最大大小
	if q.maxSize > 0 && q.queue.Len() >= q.maxSize {
		// Remove the oldest event to make room
		// 删除最旧的事件以腾出空间
		q.queue.Remove(q.queue.Front())
	}

	// Add the new event to the back of the queue
	// 将新事件添加到队列的末尾
	q.queue.PushBack(event)
	return nil
}

// Dequeue removes and returns the next event from the queue.
// Dequeue 从队列中移除并返回下一个事件。
func (q *queueManager) Dequeue() (*interfaces.Event, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// Check if queue is empty
	// 检查队列是否为空
	if q.queue.Len() == 0 {
		return nil, nil // Queue is empty
		// 队列为空
	}

	// Remove and return the front element
	// 删除并返回前端元素
	element := q.queue.Front()
	q.queue.Remove(element)
	return element.Value.(*interfaces.Event), nil
}

// Peek returns the next event from the queue without removing it.
// Peek 返回队列中的下一个事件但不移除它。
func (q *queueManager) Peek() (*interfaces.Event, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// Check if queue is empty
	// 检查队列是否为空
	if q.queue.Len() == 0 {
		return nil, nil // Queue is empty
		// 队列为空
	}

	// Return the front element without removing it
	// 返回前端元素但不删除它
	element := q.queue.Front()
	return element.Value.(*interfaces.Event), nil
}

// Size returns the number of events in the queue.
// Size 返回队列中的事件数量。
func (q *queueManager) Size() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return q.queue.Len()
}

// IsEmpty checks if the queue is empty.
// IsEmpty 检查队列是否为空。
func (q *queueManager) IsEmpty() bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return q.queue.Len() == 0
}

// Clear removes all events from the queue.
// Clear 从队列中移除所有事件。
func (q *queueManager) Clear() error {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.queue.Init() // Reinitialize the list to clear it
	// 重新初始化列表以清除它
	return nil
}

// SetMaxSize sets the maximum size of the queue.
// SetMaxSize 设置队列的最大大小。
func (q *queueManager) SetMaxSize(maxSize int) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.maxSize = maxSize
}

// GetMaxSize returns the maximum size of the queue.
// GetMaxSize 返回队列的最大大小。
func (q *queueManager) GetMaxSize() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	return q.maxSize
}

// ProcessQueue processes all events in the queue by sending them to the ACS.
// ProcessQueue 通过将队列中的所有事件发送到 ACS 来处理它们。
func (q *queueManager) ProcessQueue(ctx context.Context) error {
	for !q.IsEmpty() {
		event, err := q.Dequeue()
		if err != nil {
			return err
		}
		if event != nil {
			if err := q.notifier.Notify(ctx, event); err != nil {
				// If notification fails, put the event back in the queue
				// 如果通知失败，将事件放回队列
				q.mutex.Lock()
				q.queue.PushFront(event)
				q.mutex.Unlock()
				return err
			}
		}
	}
	return nil
}