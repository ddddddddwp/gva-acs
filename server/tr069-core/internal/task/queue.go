// Package task provides implementation for task queue management.
package task

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// taskQueue implements the interfaces.TaskQueue interface.
type taskQueue struct {
	tasks            map[string]*interfaces.TaskInfo
	handlers         map[string]interfaces.TaskHandler
	listeners        []interfaces.TaskEventListener
	pendingTasks     []*interfaces.TaskInfo
	concurrency      int
	workerPool       chan struct{}
	tasksMutex       sync.RWMutex
	handlersMutex    sync.RWMutex
	listenersMutex   sync.RWMutex
	pendingTaskMutex sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	priorityManager  *PriorityManager
	retryManager     *RetryManager
}

// NewTaskQueue creates a new task queue with the specified concurrency.
func NewTaskQueue(concurrency int) interfaces.TaskQueue {
	if concurrency <= 0 {
		concurrency = 10 // Default concurrency
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	queue := &taskQueue{
		tasks:        make(map[string]*interfaces.TaskInfo),
		handlers:     make(map[string]interfaces.TaskHandler),
		listeners:    make([]interfaces.TaskEventListener, 0),
		pendingTasks: make([]*interfaces.TaskInfo, 0),
		concurrency:  concurrency,
		workerPool:   make(chan struct{}, concurrency),
		ctx:          ctx,
		cancel:       cancel,
	}
	
	// Initialize priority manager
	queue.priorityManager = NewPriorityManager(queue)
	// Initialize retry manager
	queue.retryManager = NewRetryManager(queue)
	
	return queue
}

// EnqueueTask adds a task to the queue.
func (q *taskQueue) EnqueueTask(ctx context.Context, taskType string, data interface{}, options ...interfaces.TaskOption) (*interfaces.TaskInfo, error) {
	q.handlersMutex.RLock()
	_, handlerExists := q.handlers[taskType]
	q.handlersMutex.RUnlock()

	if !handlerExists {
		return nil, fmt.Errorf("no handler registered for task type: %s", taskType)
	}

	taskID := uuid.New().String()
	now := time.Now()

	task := &interfaces.TaskInfo{
		ID:          taskID,
		Type:        taskType,
		Status:      interfaces.TaskStatusPending,
		Priority:    interfaces.TaskPriorityNormal, // Default priority
		Data:        data,
		CreatedAt:   now,
		RetryCount:  0,
		MaxRetries:  3, // Default to 3 retries
		Metadata:    make(map[string]interface{}),
		Cancellable: true, // Default to cancellable
	}

	// Apply options
	for _, option := range options {
		option(task)
	}

	q.tasksMutex.Lock()
	q.tasks[taskID] = task
	q.tasksMutex.Unlock()

	q.pendingTaskMutex.Lock()
	q.pendingTasks = append(q.pendingTasks, task)
	q.sortPendingTasksByPriority()
	q.pendingTaskMutex.Unlock()

	q.emitTaskEvent(interfaces.TaskCreated, task)

	return task, nil
}

// GetTask retrieves a task by its ID.
func (q *taskQueue) GetTask(ctx context.Context, taskID string) (*interfaces.TaskInfo, error) {
	q.tasksMutex.RLock()
	defer q.tasksMutex.RUnlock()

	task, exists := q.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	return task, nil
}

// GetNextTask gets the next task from the queue.
func (q *taskQueue) GetNextTask() *interfaces.TaskInfo {
	q.pendingTaskMutex.Lock()
	defer q.pendingTaskMutex.Unlock()

	if len(q.pendingTasks) == 0 {
		return nil
	}

	// Get the highest priority task
	task := q.pendingTasks[0]
	q.pendingTasks = q.pendingTasks[1:]

	// Update task status
	q.tasksMutex.Lock()
	task.Status = interfaces.TaskStatusRunning
	task.StartedAt = time.Now()
	q.tasksMutex.Unlock()

	// Create a cancellation context for this task
	_, cancel := context.WithCancel(context.Background())
	q.retryManager.RegisterCancelFunc(task.ID, cancel)

	q.emitTaskEvent(interfaces.TaskStarted, task)

	return task
}

// CancelTask cancels a task.
func (q *taskQueue) CancelTask(ctx context.Context, taskID string) error {
	q.tasksMutex.RLock()
	task, exists := q.tasks[taskID]
	q.tasksMutex.RUnlock()

	if !exists {
		return interfaces.ErrTaskNotFound
	}

	// Check if task is already completed or cancelled
	if task.Status == interfaces.TaskStatusCompleted || 
	   task.Status == interfaces.TaskStatusFailed || 
	   task.Status == interfaces.TaskStatusCancelled {
		return interfaces.ErrTaskAlreadyCompleted
	}

	// Check if task is cancellable
	if !task.Cancellable {
		return interfaces.ErrTaskNotCancellable
	}

	// Use retry manager to cancel the task
	if q.retryManager.CancelTask(taskID) {
		return nil
	}

	// If task is pending, remove it from pending queue
	q.pendingTaskMutex.Lock()
	for i, pendingTask := range q.pendingTasks {
		if pendingTask.ID == taskID {
			q.pendingTasks = append(q.pendingTasks[:i], q.pendingTasks[i+1:]...)
			break
		}
	}
	q.pendingTaskMutex.Unlock()

	// Update task status
	q.tasksMutex.Lock()
	task.Status = interfaces.TaskStatusCancelled
	q.tasksMutex.Unlock()

	q.emitTaskEvent(interfaces.TaskCancelled, task)

	return nil
}

// ListTasks lists all tasks.
func (q *taskQueue) ListTasks(ctx context.Context) ([]*interfaces.TaskInfo, error) {
	q.tasksMutex.RLock()
	defer q.tasksMutex.RUnlock()

	tasks := make([]*interfaces.TaskInfo, 0, len(q.tasks))
	for _, task := range q.tasks {
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// ListTasksByStatus lists tasks with the specified status.
func (q *taskQueue) ListTasksByStatus(ctx context.Context, status interfaces.TaskStatus) ([]*interfaces.TaskInfo, error) {
	q.tasksMutex.RLock()
	defer q.tasksMutex.RUnlock()

	tasks := make([]*interfaces.TaskInfo, 0)
	for _, task := range q.tasks {
		if task.Status == status {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// ListTasksByDevice lists tasks for a specific device.
func (q *taskQueue) ListTasksByDevice(ctx context.Context, deviceID string) ([]*interfaces.TaskInfo, error) {
	q.tasksMutex.RLock()
	defer q.tasksMutex.RUnlock()

	tasks := make([]*interfaces.TaskInfo, 0)
	for _, task := range q.tasks {
		if task.DeviceID == deviceID {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// ListTasksBySession lists tasks for a specific session.
func (q *taskQueue) ListTasksBySession(ctx context.Context, sessionID string) ([]*interfaces.TaskInfo, error) {
	q.tasksMutex.RLock()
	defer q.tasksMutex.RUnlock()

	tasks := make([]*interfaces.TaskInfo, 0)
	for _, task := range q.tasks {
		if task.SessionID == sessionID {
			tasks = append(tasks, task)
		}
	}

	return tasks, nil
}

// RegisterTaskHandler registers a handler for a specific task type.
func (q *taskQueue) RegisterTaskHandler(taskType string, handler interfaces.TaskHandler) {
	q.handlersMutex.Lock()
	defer q.handlersMutex.Unlock()

	q.handlers[taskType] = handler
}

// UnregisterTaskHandler unregisters a handler for a specific task type.
func (q *taskQueue) UnregisterTaskHandler(taskType string) {
	q.handlersMutex.Lock()
	defer q.handlersMutex.Unlock()

	delete(q.handlers, taskType)
}

// SetConcurrency sets the maximum number of concurrent tasks.
func (q *taskQueue) SetConcurrency(concurrency int) {
	if concurrency <= 0 {
		return
	}

	q.tasksMutex.Lock()
	defer q.tasksMutex.Unlock()

	// Create a new worker pool with the new concurrency
	oldPool := q.workerPool
	q.workerPool = make(chan struct{}, concurrency)
	q.concurrency = concurrency

	// Close the old pool
	close(oldPool)
}

// GetConcurrency gets the current maximum number of concurrent tasks.
func (q *taskQueue) GetConcurrency() int {
	q.tasksMutex.RLock()
	defer q.tasksMutex.RUnlock()
	
	return q.concurrency
}

// Start starts the task queue processing.
func (q *taskQueue) Start() {
	q.wg.Add(1)
	go q.processTasksLoop()
}

// Stop stops the task queue processing.
func (q *taskQueue) Stop() {
	q.cancel()
	q.wg.Wait()
}

// RegisterTaskListener registers a listener for task events.
func (q *taskQueue) RegisterTaskListener(listener interfaces.TaskEventListener) {
	q.listenersMutex.Lock()
	defer q.listenersMutex.Unlock()

	q.listeners = append(q.listeners, listener)
}

// UnregisterTaskListener unregisters a task event listener.
func (q *taskQueue) UnregisterTaskListener(listener interfaces.TaskEventListener) {
	q.listenersMutex.Lock()
	defer q.listenersMutex.Unlock()

	for i, l := range q.listeners {
		if l == listener {
			q.listeners = append(q.listeners[:i], q.listeners[i+1:]...)
			break
		}
	}
}

// processTasksLoop continuously processes tasks from the queue.
func (q *taskQueue) processTasksLoop() {
	defer q.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-q.ctx.Done():
			return
		case <-ticker.C:
			q.processPendingTasks()
		}
	}
}

// processPendingTasks processes pending tasks.
func (q *taskQueue) processPendingTasks() {
	q.pendingTaskMutex.Lock()
	if len(q.pendingTasks) == 0 {
		q.pendingTaskMutex.Unlock()
		return
	}

	// Get the next task
	task := q.pendingTasks[0]
	q.pendingTasks = q.pendingTasks[1:]
	q.pendingTaskMutex.Unlock()

	// Check if the task is still pending
	q.tasksMutex.RLock()
	if task.Status != interfaces.TaskStatusPending {
		q.tasksMutex.RUnlock()
		return
	}
	q.tasksMutex.RUnlock()

	// Try to acquire a worker
	select {
	case q.workerPool <- struct{}{}:
		// Worker acquired, process the task
		q.wg.Add(1)
		go q.processTask(task)
	default:
		// No workers available, put the task back in the queue
		q.pendingTaskMutex.Lock()
		q.pendingTasks = append([]*interfaces.TaskInfo{task}, q.pendingTasks...)
		q.pendingTaskMutex.Unlock()
	}
}

// processTask processes a single task.
func (q *taskQueue) processTask(task *interfaces.TaskInfo) {
	defer func() {
		<-q.workerPool // Release the worker
		q.wg.Done()
	}()

	// Update task status to running
	q.tasksMutex.Lock()
	task.Status = interfaces.TaskStatusRunning
	task.StartedAt = time.Now()
	q.tasksMutex.Unlock()

	// Create a cancellation context for this task
	ctx, cancel := context.WithCancel(q.ctx)
	q.retryManager.RegisterCancelFunc(task.ID, cancel)
	defer q.retryManager.UnregisterCancelFunc(task.ID)

	q.emitTaskEvent(interfaces.TaskStarted, task)

	// Get the handler
	q.handlersMutex.RLock()
	handler, exists := q.handlers[task.Type]
	q.handlersMutex.RUnlock()

	if !exists {
		q.tasksMutex.Lock()
		task.Status = interfaces.TaskStatusFailed
		task.Error = errors.New("no handler registered for task type")
		q.tasksMutex.Unlock()

		q.emitTaskEvent(interfaces.TaskFailed, task)
		return
	}

	// Create a context with timeout if needed
	taskCtx := ctx
	if deadline, ok := task.Metadata["deadline"].(time.Time); ok {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithDeadline(taskCtx, deadline)
		defer cancel()
	}

	// Execute the handler
	result, err := handler(taskCtx, task)

	q.tasksMutex.Lock()
	task.CompletedAt = time.Now()
	task.Result = result

	if err != nil {
		task.Error = err
		if task.RetryCount < task.MaxRetries {
			task.Status = interfaces.TaskStatusRetrying
			task.RetryCount++
			
			// Put the task back in the queue for retry
			q.pendingTaskMutex.Lock()
			q.pendingTasks = append(q.pendingTasks, task)
			q.sortPendingTasksByPriority()
			q.pendingTaskMutex.Unlock()
			
			q.tasksMutex.Unlock()
			q.emitTaskEvent(interfaces.TaskRetrying, task)
		} else {
			task.Status = interfaces.TaskStatusFailed
			q.tasksMutex.Unlock()
			q.emitTaskEvent(interfaces.TaskFailed, task)
		}
	} else {
		task.Status = interfaces.TaskStatusCompleted
		q.tasksMutex.Unlock()
		q.emitTaskEvent(interfaces.TaskCompleted, task)
	}
}

// emitTaskEvent emits a task event to all registered listeners.
func (q *taskQueue) emitTaskEvent(eventType interfaces.TaskEventType, task *interfaces.TaskInfo) {
	event := &interfaces.TaskEvent{
		Type:      eventType,
		TaskID:    task.ID,
		TaskType:  task.Type,
		SessionID: task.SessionID,
		DeviceID:  task.DeviceID,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	}

	// Add task status to event data
	event.Data["status"] = task.Status

	// Add error to event data if present
	if task.Error != nil {
		event.Data["error"] = task.Error.Error()
	}

	q.listenersMutex.RLock()
	listeners := make([]interfaces.TaskEventListener, len(q.listeners))
	copy(listeners, q.listeners)
	q.listenersMutex.RUnlock()

	for _, listener := range listeners {
		go listener.OnTaskEvent(event)
	}
}

// sortPendingTasksByPriority sorts the pending tasks by priority.
func (q *taskQueue) sortPendingTasksByPriority() {
	// Simple bubble sort for now, can be optimized later
	n := len(q.pendingTasks)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if q.pendingTasks[j].Priority < q.pendingTasks[j+1].Priority {
				q.pendingTasks[j], q.pendingTasks[j+1] = q.pendingTasks[j+1], q.pendingTasks[j]
			}
		}
	}
}