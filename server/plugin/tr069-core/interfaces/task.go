// Package interfaces defines the interfaces for TR069 protocol components.
package interfaces

import (
	"context"
	"errors"
	"time"
)

// Task-related errors
var (
	ErrTaskNotFound         = errors.New("task not found")
	ErrTaskAlreadyCompleted = errors.New("task already completed")
	ErrTaskNotCancellable   = errors.New("task is not cancellable")
)

// TaskStatus represents the current status of a task.
type TaskStatus string

const (
	// TaskStatusPending indicates a task is waiting to be processed.
	TaskStatusPending TaskStatus = "pending"
	// TaskStatusRunning indicates a task is currently being processed.
	TaskStatusRunning TaskStatus = "running"
	// TaskStatusCompleted indicates a task has been completed successfully.
	TaskStatusCompleted TaskStatus = "completed"
	// TaskStatusFailed indicates a task has failed.
	TaskStatusFailed TaskStatus = "failed"
	// TaskStatusCancelled indicates a task has been cancelled.
	TaskStatusCancelled TaskStatus = "cancelled"
	// TaskStatusRetrying indicates a task is being retried after a failure.
	TaskStatusRetrying TaskStatus = "retrying"
)

// TaskPriority represents the priority level of a task.
type TaskPriority int

const (
	// TaskPriorityLow represents low priority tasks.
	TaskPriorityLow TaskPriority = 1
	// TaskPriorityNormal represents normal priority tasks.
	TaskPriorityNormal TaskPriority = 2
	// TaskPriorityHigh represents high priority tasks.
	TaskPriorityHigh TaskPriority = 3
	// TaskPriorityCritical represents critical priority tasks.
	TaskPriorityCritical TaskPriority = 4
)

// TaskInfo contains information about a task.
type TaskInfo struct {
	// ID is the unique identifier for the task.
	ID string
	// Type is the type of the task.
	Type string
	// Status is the current status of the task.
	Status TaskStatus
	// Priority is the priority of the task.
	Priority TaskPriority
	// Data contains the task data.
	Data interface{}
	// Result contains the task result.
	Result interface{}
	// Error contains any error that occurred during task execution.
	Error error
	// CreatedAt is the time when the task was created.
	CreatedAt time.Time
	// StartedAt is the time when the task was started.
	StartedAt time.Time
	// CompletedAt is the time when the task was completed.
	CompletedAt time.Time
	// RetryCount is the number of times the task has been retried.
	RetryCount int
	// MaxRetries is the maximum number of retries allowed.
	MaxRetries int
	// SessionID is the ID of the session associated with the task.
	SessionID string
	// DeviceID is the device ID associated with the task.
	DeviceID string
	// Cancellable indicates whether the task can be cancelled.
	Cancellable bool
	// Metadata contains additional task metadata.
	Metadata map[string]interface{}
}

// TaskHandler is a function that processes a task.
type TaskHandler func(ctx context.Context, task *TaskInfo) (interface{}, error)

// TaskQueue defines the interface for managing tasks.
type TaskQueue interface {
	// EnqueueTask adds a task to the queue.
	EnqueueTask(ctx context.Context, taskType string, data interface{}, options ...TaskOption) (*TaskInfo, error)
	
	// GetTask retrieves a task by its ID.
	GetTask(ctx context.Context, taskID string) (*TaskInfo, error)
	
	// CancelTask cancels a task.
	CancelTask(ctx context.Context, taskID string) error
	
	// ListTasks lists all tasks.
	ListTasks(ctx context.Context) ([]*TaskInfo, error)
	
	// ListTasksByStatus lists tasks with the specified status.
	ListTasksByStatus(ctx context.Context, status TaskStatus) ([]*TaskInfo, error)
	
	// ListTasksByDevice lists tasks for a specific device.
	ListTasksByDevice(ctx context.Context, deviceID string) ([]*TaskInfo, error)
	
	// ListTasksBySession lists tasks for a specific session.
	ListTasksBySession(ctx context.Context, sessionID string) ([]*TaskInfo, error)
	
	// RegisterTaskHandler registers a handler for a specific task type.
	RegisterTaskHandler(taskType string, handler TaskHandler)
	
	// UnregisterTaskHandler unregisters a handler for a specific task type.
	UnregisterTaskHandler(taskType string)
	
	// SetConcurrency sets the maximum number of concurrent tasks.
	SetConcurrency(concurrency int)
	
	// GetConcurrency gets the current maximum number of concurrent tasks.
	GetConcurrency() int
	
	// Start starts the task queue processing.
	Start()
	
	// Stop stops the task queue processing.
	Stop()
	
	// RegisterTaskListener registers a listener for task events.
	RegisterTaskListener(listener TaskEventListener)
	
	// UnregisterTaskListener unregisters a task event listener.
	UnregisterTaskListener(listener TaskEventListener)
}

// TaskOption is a function that configures a task.
type TaskOption func(*TaskInfo)

// WithTaskPriority sets the priority for a task.
func WithTaskPriority(priority TaskPriority) TaskOption {
	return func(info *TaskInfo) {
		info.Priority = priority
	}
}

// WithTaskRetries sets the maximum number of retries for a task.
func WithTaskRetries(maxRetries int) TaskOption {
	return func(info *TaskInfo) {
		info.MaxRetries = maxRetries
	}
}

// WithTaskSessionID associates a task with a session.
func WithTaskSessionID(sessionID string) TaskOption {
	return func(info *TaskInfo) {
		info.SessionID = sessionID
	}
}

// WithTaskDeviceID associates a task with a device.
func WithTaskDeviceID(deviceID string) TaskOption {
	return func(info *TaskInfo) {
		info.DeviceID = deviceID
	}
}

// WithTaskMetadata adds metadata to a task.
func WithTaskMetadata(key string, value interface{}) TaskOption {
	return func(info *TaskInfo) {
		if info.Metadata == nil {
			info.Metadata = make(map[string]interface{})
		}
		info.Metadata[key] = value
	}
}

// TaskEventType represents the type of task event.
type TaskEventType string

const (
	// TaskCreated indicates a task was created.
	TaskCreated TaskEventType = "created"
	// TaskStarted indicates a task was started.
	TaskStarted TaskEventType = "started"
	// TaskCompleted indicates a task was completed.
	TaskCompleted TaskEventType = "completed"
	// TaskFailed indicates a task has failed.
	TaskFailed TaskEventType = "failed"
	// TaskCancelled indicates a task was cancelled.
	TaskCancelled TaskEventType = "cancelled"
	// TaskRetrying indicates a task is being retried.
	TaskRetrying TaskEventType = "retrying"
)

// TaskEvent represents an event related to a task.
type TaskEvent struct {
	// Type is the type of the event.
	Type TaskEventType
	// TaskID is the ID of the task.
	TaskID string
	// TaskType is the type of the task.
	TaskType string
	// SessionID is the ID of the session associated with the task.
	SessionID string
	// DeviceID is the ID of the device associated with the task.
	DeviceID string
	// Timestamp is the time when the event occurred.
	Timestamp time.Time
	// Data contains additional event data.
	Data map[string]interface{}
}

// TaskEventListener is the interface for objects that listen to task events.
type TaskEventListener interface {
	// OnTaskEvent is called when a task event occurs.
	OnTaskEvent(event *TaskEvent)
}