// Package task provides implementation for task queue management.
package task

import (
	"context"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// RetryManager manages task retries and cancellations.
type RetryManager struct {
	taskQueue      *taskQueue
	retryDelays    map[int]time.Duration
	maxRetries     int
	retryMutex     sync.RWMutex
	cancelContexts map[string]context.CancelFunc
	cancelMutex    sync.RWMutex
}

// NewRetryManager creates a new retry manager.
func NewRetryManager(taskQueue *taskQueue) *RetryManager {
	// Default retry delays with exponential backoff
	retryDelays := map[int]time.Duration{
		1: 5 * time.Second,
		2: 15 * time.Second,
		3: 30 * time.Second,
		4: 1 * time.Minute,
		5: 5 * time.Minute,
	}

	return &RetryManager{
		taskQueue:      taskQueue,
		retryDelays:    retryDelays,
		maxRetries:     5, // Default max retries
		cancelContexts: make(map[string]context.CancelFunc),
	}
}

// SetMaxRetries sets the maximum number of retries.
func (rm *RetryManager) SetMaxRetries(maxRetries int) {
	if maxRetries < 0 {
		maxRetries = 0
	}
	
	rm.retryMutex.Lock()
	defer rm.retryMutex.Unlock()
	
	rm.maxRetries = maxRetries
}

// GetMaxRetries gets the maximum number of retries.
func (rm *RetryManager) GetMaxRetries() int {
	rm.retryMutex.RLock()
	defer rm.retryMutex.RUnlock()
	
	return rm.maxRetries
}

// SetRetryDelay sets the retry delay for a specific retry count.
func (rm *RetryManager) SetRetryDelay(retryCount int, delay time.Duration) {
	if retryCount <= 0 || delay < 0 {
		return
	}
	
	rm.retryMutex.Lock()
	defer rm.retryMutex.Unlock()
	
	rm.retryDelays[retryCount] = delay
}

// GetRetryDelay gets the retry delay for a specific retry count.
func (rm *RetryManager) GetRetryDelay(retryCount int) time.Duration {
	rm.retryMutex.RLock()
	defer rm.retryMutex.RUnlock()
	
	if delay, exists := rm.retryDelays[retryCount]; exists {
		return delay
	}
	
	// Default delay for unknown retry counts
	return time.Duration(retryCount) * 10 * time.Second
}

// RegisterCancelFunc registers a cancel function for a task.
func (rm *RetryManager) RegisterCancelFunc(taskID string, cancel context.CancelFunc) {
	rm.cancelMutex.Lock()
	defer rm.cancelMutex.Unlock()
	
	rm.cancelContexts[taskID] = cancel
}

// UnregisterCancelFunc unregisters a cancel function for a task.
func (rm *RetryManager) UnregisterCancelFunc(taskID string) {
	rm.cancelMutex.Lock()
	defer rm.cancelMutex.Unlock()
	
	delete(rm.cancelContexts, taskID)
}

// CancelTask cancels a task.
func (rm *RetryManager) CancelTask(taskID string) bool {
	rm.cancelMutex.Lock()
	cancel, exists := rm.cancelContexts[taskID]
	rm.cancelMutex.Unlock()
	
	if !exists {
		return false
	}
	
	// Call the cancel function
	cancel()
	
	// Update task status
	rm.taskQueue.tasksMutex.Lock()
	task, exists := rm.taskQueue.tasks[taskID]
	if exists {
		task.Status = interfaces.TaskStatusCancelled
	}
	rm.taskQueue.tasksMutex.Unlock()
	
	if exists {
		rm.taskQueue.emitTaskEvent(interfaces.TaskCancelled, task)
	}
	
	return exists
}

// ScheduleRetry schedules a task for retry.
func (rm *RetryManager) ScheduleRetry(task *interfaces.TaskInfo) {
	if task.RetryCount >= rm.GetMaxRetries() {
		// Max retries reached, mark as failed
		rm.taskQueue.tasksMutex.Lock()
		task.Status = interfaces.TaskStatusFailed
		rm.taskQueue.tasksMutex.Unlock()
		
		rm.taskQueue.emitTaskEvent(interfaces.TaskFailed, task)
		return
	}
	
	// Increment retry count
	task.RetryCount++
	
	// Get retry delay
	delay := rm.GetRetryDelay(task.RetryCount)
	
	// Schedule retry
	go func() {
		time.Sleep(delay)
		
		// Check if task is still retrying
		rm.taskQueue.tasksMutex.RLock()
		if task.Status != interfaces.TaskStatusRetrying {
			rm.taskQueue.tasksMutex.RUnlock()
			return
		}
		rm.taskQueue.tasksMutex.RUnlock()
		
		// Add task back to pending queue
		rm.taskQueue.pendingTaskMutex.Lock()
		task.Status = interfaces.TaskStatusPending
		rm.taskQueue.pendingTasks = append(rm.taskQueue.pendingTasks, task)
		rm.taskQueue.sortPendingTasksByPriority()
		rm.taskQueue.pendingTaskMutex.Unlock()
	}()
}

// HandleTaskError handles a task error and determines if it should be retried.
func (rm *RetryManager) HandleTaskError(task *interfaces.TaskInfo, err error) {
	// Check if task is retriable
	retriable := true
	
	// Check if task has metadata indicating it's not retriable
	if value, exists := task.Metadata["retriable"]; exists {
		if b, ok := value.(bool); ok {
			retriable = b
		}
	}
	
	// Store error
	task.Error = err
	
	if retriable && task.RetryCount < rm.GetMaxRetries() {
		// Mark task for retry
		rm.taskQueue.tasksMutex.Lock()
		task.Status = interfaces.TaskStatusRetrying
		rm.taskQueue.tasksMutex.Unlock()
		
		rm.taskQueue.emitTaskEvent(interfaces.TaskRetrying, task)
		
		// Schedule retry
		rm.ScheduleRetry(task)
	} else {
		// Mark task as failed
		rm.taskQueue.tasksMutex.Lock()
		task.Status = interfaces.TaskStatusFailed
		rm.taskQueue.tasksMutex.Unlock()
		
		rm.taskQueue.emitTaskEvent(interfaces.TaskFailed, task)
	}
}