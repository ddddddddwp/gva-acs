// Package task provides implementation for task queue management.
package task

import (
	"container/heap"
	"sync"
	"time"

	"github.com/root/demo/tr069/interfaces"
)

// priorityQueue implements a priority queue for tasks.
type priorityQueue struct {
	tasks []*interfaces.TaskInfo
	mutex sync.RWMutex
}

// Implement heap.Interface for priorityQueue
func (pq *priorityQueue) Len() int { 
	return len(pq.tasks) 
}

func (pq *priorityQueue) Less(i, j int) bool {
	// Higher priority tasks come first
	if pq.tasks[i].Priority != pq.tasks[j].Priority {
		return pq.tasks[i].Priority > pq.tasks[j].Priority
	}
	
	// For tasks with the same priority, older tasks come first (FIFO)
	return pq.tasks[i].CreatedAt.Before(pq.tasks[j].CreatedAt)
}

func (pq *priorityQueue) Swap(i, j int) {
	pq.tasks[i], pq.tasks[j] = pq.tasks[j], pq.tasks[i]
}

func (pq *priorityQueue) Push(x interface{}) {
	pq.tasks = append(pq.tasks, x.(*interfaces.TaskInfo))
}

func (pq *priorityQueue) Pop() interface{} {
	old := pq.tasks
	n := len(old)
	item := old[n-1]
	pq.tasks = old[0 : n-1]
	return item
}

// PriorityManager manages task priorities.
type PriorityManager struct {
	queue          *priorityQueue
	taskQueue      *taskQueue
	dynamicEnabled bool
	mutex          sync.RWMutex
}

// NewPriorityManager creates a new priority manager.
func NewPriorityManager(taskQueue *taskQueue) *PriorityManager {
	pq := &priorityQueue{
		tasks: make([]*interfaces.TaskInfo, 0),
	}
	heap.Init(pq)
	
	return &PriorityManager{
		queue:          pq,
		taskQueue:      taskQueue,
		dynamicEnabled: false,
	}
}

// AddTask adds a task to the priority queue.
func (pm *PriorityManager) AddTask(task *interfaces.TaskInfo) {
	pm.queue.mutex.Lock()
	defer pm.queue.mutex.Unlock()
	
	heap.Push(pm.queue, task)
}

// GetNextTask gets the next task from the priority queue.
func (pm *PriorityManager) GetNextTask() *interfaces.TaskInfo {
	pm.queue.mutex.Lock()
	defer pm.queue.mutex.Unlock()
	
	if pm.queue.Len() == 0 {
		return nil
	}
	
	return heap.Pop(pm.queue).(*interfaces.TaskInfo)
}

// UpdateTaskPriority updates the priority of a task.
func (pm *PriorityManager) UpdateTaskPriority(taskID string, priority interfaces.TaskPriority) bool {
	pm.taskQueue.tasksMutex.Lock()
	task, exists := pm.taskQueue.tasks[taskID]
	if !exists {
		pm.taskQueue.tasksMutex.Unlock()
		return false
	}
	
	// Update task priority
	task.Priority = priority
	pm.taskQueue.tasksMutex.Unlock()
	
	// Re-sort pending tasks
	pm.taskQueue.pendingTaskMutex.Lock()
	pm.taskQueue.sortPendingTasksByPriority()
	pm.taskQueue.pendingTaskMutex.Unlock()
	
	return true
}

// EnableDynamicPriorities enables dynamic priority adjustment.
func (pm *PriorityManager) EnableDynamicPriorities() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	if !pm.dynamicEnabled {
		pm.dynamicEnabled = true
		go pm.dynamicPriorityLoop()
	}
}

// DisableDynamicPriorities disables dynamic priority adjustment.
func (pm *PriorityManager) DisableDynamicPriorities() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	pm.dynamicEnabled = false
}

// IsDynamicPrioritiesEnabled checks if dynamic priorities are enabled.
func (pm *PriorityManager) IsDynamicPrioritiesEnabled() bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	
	return pm.dynamicEnabled
}

// dynamicPriorityLoop periodically adjusts task priorities.
func (pm *PriorityManager) dynamicPriorityLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		<-ticker.C
		
		pm.mutex.RLock()
		if !pm.dynamicEnabled {
			pm.mutex.RUnlock()
			return
		}
		pm.mutex.RUnlock()
		
		pm.adjustPriorities()
	}
}

// adjustPriorities adjusts task priorities based on various factors.
func (pm *PriorityManager) adjustPriorities() {
	pm.taskQueue.tasksMutex.RLock()
	tasks := make([]*interfaces.TaskInfo, 0, len(pm.taskQueue.tasks))
	for _, task := range pm.taskQueue.tasks {
		if task.Status == interfaces.TaskStatusPending {
			tasks = append(tasks, task)
		}
	}
	pm.taskQueue.tasksMutex.RUnlock()
	
	now := time.Now()
	
	for _, task := range tasks {
		// Skip tasks that are not pending
		if task.Status != interfaces.TaskStatusPending {
			continue
		}
		
		// Calculate age factor (older tasks get higher priority)
		ageMinutes := now.Sub(task.CreatedAt).Minutes()
		
		// Boost priority for tasks waiting too long
		if ageMinutes > 10 && task.Priority < interfaces.TaskPriorityHigh {
			pm.UpdateTaskPriority(task.ID, task.Priority+1)
		}
		
		// Boost priority for tasks with retry attempts
		if task.RetryCount > 0 && task.Priority < interfaces.TaskPriorityCritical {
			pm.UpdateTaskPriority(task.ID, task.Priority+1)
		}
	}
}