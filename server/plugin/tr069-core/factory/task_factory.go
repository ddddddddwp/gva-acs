// Package factory provides factory methods for creating TR069 components.
package factory

import (
	"github.com/root/demo/tr069/interfaces"
	"github.com/root/demo/tr069/internal/task"
)

// CreateTaskQueue creates a new task queue with default configuration.
func CreateTaskQueue() interfaces.TaskQueue {
	return task.NewTaskQueue(10) // Default concurrency of 10
}

// CreateTaskQueueWithConcurrency creates a new task queue with the specified concurrency.
func CreateTaskQueueWithConcurrency(concurrency int) interfaces.TaskQueue {
	return task.NewTaskQueue(concurrency)
}

// TaskQueueOption is a function that configures a task queue.
type TaskQueueOption func(interfaces.TaskQueue)

// WithConcurrency sets the concurrency for a task queue.
func WithConcurrency(concurrency int) TaskQueueOption {
	return func(queue interfaces.TaskQueue) {
		queue.SetConcurrency(concurrency)
	}
}

// WithTaskHandler registers a task handler for a specific task type.
func WithTaskHandler(taskType string, handler interfaces.TaskHandler) TaskQueueOption {
	return func(queue interfaces.TaskQueue) {
		queue.RegisterTaskHandler(taskType, handler)
	}
}

// WithTaskListener registers a task event listener.
func WithTaskListener(listener interfaces.TaskEventListener) TaskQueueOption {
	return func(queue interfaces.TaskQueue) {
		queue.RegisterTaskListener(listener)
	}
}

// CreateTaskQueueWithOptions creates a new task queue with the specified options.
func CreateTaskQueueWithOptions(options ...TaskQueueOption) interfaces.TaskQueue {
	queue := CreateTaskQueue()
	
	for _, option := range options {
		option(queue)
	}
	
	return queue
}