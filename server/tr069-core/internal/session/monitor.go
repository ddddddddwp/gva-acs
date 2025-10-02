// Package session provides implementation for TR069 session management.
package session

import (
	"context"
	"sync"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// SessionMonitor implements session monitoring functionality.
type SessionMonitor struct {
	manager       *sessionManager
	listeners     []interfaces.SessionEventListener
	listenerMutex sync.RWMutex
	statsMap      map[string]*SessionStats
	statsMutex    sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	interval      time.Duration
}

// SessionStats contains statistics for a session.
type SessionStats struct {
	SessionID       string
	DeviceID  string
	State     interfaces.SessionState
	CreatedAt time.Time
	LastActivityAt  time.Time
	RequestCount    int
	ResponseCount   int
	ErrorCount      int
	BytesReceived   int64
	BytesSent       int64
	AverageLatency  time.Duration
	LatencySamples  []time.Duration
	ActiveRequests  int
	CompletedTasks  int
	FailedTasks     int
	PendingTasks    int
	ConnectionCount int
}

// NewSessionMonitor creates a new session monitor.
func NewSessionMonitor(manager *sessionManager) *SessionMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &SessionMonitor{
		manager:    manager,
		listeners:  make([]interfaces.SessionEventListener, 0),
		statsMap:   make(map[string]*SessionStats),
		ctx:        ctx,
		cancel:     cancel,
		interval:   5 * time.Second, // Default monitoring interval
	}
}

// Start starts the session monitor.
func (m *SessionMonitor) Start() {
	m.wg.Add(1)
	go m.monitorLoop()
}

// Stop stops the session monitor.
func (m *SessionMonitor) Stop() {
	m.cancel()
	m.wg.Wait()
}

// SetMonitoringInterval sets the interval for session monitoring.
func (m *SessionMonitor) SetMonitoringInterval(interval time.Duration) {
	if interval < time.Second {
		interval = time.Second
	}
	m.interval = interval
}

// RegisterListener registers a session event listener.
func (m *SessionMonitor) RegisterListener(listener interfaces.SessionEventListener) {
	m.listenerMutex.Lock()
	defer m.listenerMutex.Unlock()
	
	m.listeners = append(m.listeners, listener)
}

// UnregisterListener unregisters a session event listener.
func (m *SessionMonitor) UnregisterListener(listener interfaces.SessionEventListener) {
	m.listenerMutex.Lock()
	defer m.listenerMutex.Unlock()
	
	for i, l := range m.listeners {
		if l == listener {
			m.listeners = append(m.listeners[:i], m.listeners[i+1:]...)
			break
		}
	}
}

// GetSessionStats gets statistics for a session.
func (m *SessionMonitor) GetSessionStats(sessionID string) (*SessionStats, bool) {
	m.statsMutex.RLock()
	defer m.statsMutex.RUnlock()
	
	stats, exists := m.statsMap[sessionID]
	return stats, exists
}

// GetAllSessionStats gets statistics for all sessions.
func (m *SessionMonitor) GetAllSessionStats() []*SessionStats {
	m.statsMutex.RLock()
	defer m.statsMutex.RUnlock()
	
	stats := make([]*SessionStats, 0, len(m.statsMap))
	for _, s := range m.statsMap {
		stats = append(stats, s)
	}
	
	return stats
}

// UpdateSessionActivity updates the activity timestamp for a session.
func (m *SessionMonitor) UpdateSessionActivity(sessionID string) {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()
	
	if stats, exists := m.statsMap[sessionID]; exists {
		stats.LastActivityAt = time.Now()
	}
}

// RecordRequest records a request for a session.
func (m *SessionMonitor) RecordRequest(sessionID string, size int) {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()
	
	if stats, exists := m.statsMap[sessionID]; exists {
		stats.RequestCount++
		stats.BytesReceived += int64(size)
		stats.ActiveRequests++
		stats.LastActivityAt = time.Now()
	}
}

// RecordResponse records a response for a session.
func (m *SessionMonitor) RecordResponse(sessionID string, size int, latency time.Duration) {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()
	
	if stats, exists := m.statsMap[sessionID]; exists {
		stats.ResponseCount++
		stats.BytesSent += int64(size)
		stats.ActiveRequests--
		stats.LastActivityAt = time.Now()
		
		// Update latency statistics
		stats.LatencySamples = append(stats.LatencySamples, latency)
		if len(stats.LatencySamples) > 100 {
			stats.LatencySamples = stats.LatencySamples[1:]
		}
		
		// Calculate average latency
		var totalLatency time.Duration
		for _, l := range stats.LatencySamples {
			totalLatency += l
		}
		stats.AverageLatency = totalLatency / time.Duration(len(stats.LatencySamples))
	}
}

// RecordError records an error for a session.
func (m *SessionMonitor) RecordError(sessionID string) {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()
	
	if stats, exists := m.statsMap[sessionID]; exists {
		stats.ErrorCount++
		stats.LastActivityAt = time.Now()
	}
}

// RecordTaskCompleted records a completed task for a session.
func (m *SessionMonitor) RecordTaskCompleted(sessionID string) {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()
	
	if stats, exists := m.statsMap[sessionID]; exists {
		stats.CompletedTasks++
		stats.PendingTasks--
		stats.LastActivityAt = time.Now()
	}
}

// RecordTaskFailed records a failed task for a session.
func (m *SessionMonitor) RecordTaskFailed(sessionID string) {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()
	
	if stats, exists := m.statsMap[sessionID]; exists {
		stats.FailedTasks++
		stats.PendingTasks--
		stats.LastActivityAt = time.Now()
	}
}

// RecordTaskPending records a pending task for a session.
func (m *SessionMonitor) RecordTaskPending(sessionID string) {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()
	
	if stats, exists := m.statsMap[sessionID]; exists {
		stats.PendingTasks++
		stats.LastActivityAt = time.Now()
	}
}

// RecordConnection records a connection for a session.
func (m *SessionMonitor) RecordConnection(sessionID string) {
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()
	
	if stats, exists := m.statsMap[sessionID]; exists {
		stats.ConnectionCount++
		stats.LastActivityAt = time.Now()
	}
}

// monitorLoop periodically monitors sessions.
func (m *SessionMonitor) monitorLoop() {
	defer m.wg.Done()
	
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.updateSessionStats()
			m.detectStateChanges()
		}
	}
}

// updateSessionStats updates statistics for all sessions.
func (m *SessionMonitor) updateSessionStats() {
	// Get all sessions
	sessions := m.manager.getAllSessions()
	
	// Update stats map
	m.statsMutex.Lock()
	defer m.statsMutex.Unlock()
	
	// Remove stats for closed sessions
	for sessionID := range m.statsMap {
		found := false
		for _, session := range sessions {
			if session.ID == sessionID {
				found = true
				break
			}
		}
		if !found {
			delete(m.statsMap, sessionID)
		}
	}
	
	// Add or update stats for current sessions
	for _, session := range sessions {
		if _, exists := m.statsMap[session.ID]; !exists {
			m.statsMap[session.ID] = &SessionStats{
				SessionID:      session.ID,
				DeviceID:       session.DeviceID,
				State:          session.State,
				CreatedAt:      session.CreatedAt,
				LastActivityAt: session.LastActiveAt,
				LatencySamples: make([]time.Duration, 0, 100),
			}
		} else {
			m.statsMap[session.ID].State = session.State
		}
	}
}

// detectStateChanges detects changes in session states.
func (m *SessionMonitor) detectStateChanges() {
	m.manager.sessionsMutex.RLock()
	sessions := make([]*interfaces.SessionInfo, 0, len(m.manager.sessions))
	for _, session := range m.manager.sessions {
		sessions = append(sessions, session)
	}
	m.manager.sessionsMutex.RUnlock()
	
	for _, session := range sessions {
		m.statsMutex.RLock()
		stats, exists := m.statsMap[session.ID]
		m.statsMutex.RUnlock()
		
		if !exists {
			continue
		}
		
		// Check for inactivity
		if session.State == interfaces.SessionStateActive {
			inactiveThreshold := 5 * time.Minute // Configurable
			if time.Since(stats.LastActivityAt) > inactiveThreshold {
				m.emitSessionEvent(interfaces.SessionEventInactive, session)
			}
		}
		
		// Check for high error rate
		if stats.RequestCount > 0 {
			errorRate := float64(stats.ErrorCount) / float64(stats.RequestCount)
			if errorRate > 0.2 { // 20% error rate threshold
				m.emitSessionEvent(interfaces.SessionEventHighLatency, session)
			}
		}
		
		// Check for high latency
		if stats.AverageLatency > 500*time.Millisecond {
			m.emitSessionEvent(interfaces.SessionEventHighLatency, session)
		}
	}
}

// emitSessionEvent emits a session event to all registered listeners.
func (m *SessionMonitor) emitSessionEvent(eventType interfaces.SessionEventType, session *interfaces.SessionInfo) {
	event := &interfaces.SessionEvent{
		Type:      eventType,
		SessionID: session.ID,
		DeviceID:  session.DeviceID,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	}
	
	// Add session state to event data
	event.Data["state"] = session.State
	
	// Add stats to event data if available
	m.statsMutex.RLock()
	if stats, exists := m.statsMap[session.ID]; exists {
		event.Data["requestCount"] = stats.RequestCount
		event.Data["errorCount"] = stats.ErrorCount
		event.Data["averageLatency"] = stats.AverageLatency
	}
	m.statsMutex.RUnlock()
	
	m.listenerMutex.RLock()
	listeners := make([]interfaces.SessionEventListener, len(m.listeners))
	copy(listeners, m.listeners)
	m.listenerMutex.RUnlock()
	
	for _, listener := range listeners {
		go listener.OnSessionEvent(event)
	}
}