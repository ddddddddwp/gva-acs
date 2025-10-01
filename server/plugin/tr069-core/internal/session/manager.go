// Package session implements the TR069 session management functionality.
package session

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/root/demo/tr069/interfaces"
)

// sessionManager implements the SessionManager interface.
type sessionManager struct {
	sessions        map[string]*interfaces.SessionInfo
	deviceSessions  map[string][]string
	listeners       []interfaces.SessionEventListener
	sessionTimeout  time.Duration
	cleanupInterval time.Duration
	sessionsMutex   sync.RWMutex
	stopCleanup     chan struct{}
	monitor         *SessionMonitor
	cleaner         *SessionCleaner
}

// NewSessionManager creates a new session manager.
func NewSessionManager() interfaces.SessionManager {
	sm := &sessionManager{
		sessions:        make(map[string]*interfaces.SessionInfo),
		deviceSessions:  make(map[string][]string),
		sessionTimeout:  30 * time.Minute, // Default timeout
		cleanupInterval: 5 * time.Minute,  // Default cleanup interval
		stopCleanup:     make(chan struct{}),
	}

	// Create and start the session monitor
	sm.monitor = NewSessionMonitor(sm)
	sm.monitor.Start()
	
	// Create and start the session cleaner
	sm.cleaner = NewSessionCleaner(sm)
	sm.cleaner.Start()

	return sm
}

// CreateSession creates a new session for the specified device.
func (sm *sessionManager) CreateSession(ctx context.Context, deviceID string, options ...interfaces.SessionOption) (*interfaces.SessionInfo, error) {
	sm.sessionsMutex.Lock()
	defer sm.sessionsMutex.Unlock()

	// Generate a unique session ID
	sessionID := uuid.New().String()

	// Create a new session with default values
	now := time.Now()
	session := &interfaces.SessionInfo{
		ID:           sessionID,
		DeviceID:     deviceID,
		State:        interfaces.SessionStateNew,
		CreatedAt:    now,
		LastActiveAt: now,
		ExpiresAt:    now.Add(sm.sessionTimeout),
		Metadata:     make(map[string]interface{}),
	}

	// Apply options
	for _, option := range options {
		option(session)
	}

	// Store the session
	sm.sessions[sessionID] = session

	// Add to device sessions
	if _, exists := sm.deviceSessions[deviceID]; !exists {
		sm.deviceSessions[deviceID] = []string{}
	}
	sm.deviceSessions[deviceID] = append(sm.deviceSessions[deviceID], sessionID)

	// Notify listeners
	sm.notifyListeners(&interfaces.SessionEvent{
		Type:      interfaces.SessionCreated,
		SessionID: sessionID,
		DeviceID:  deviceID,
		Timestamp: now,
		Data:      make(map[string]interface{}),
	})

	return session, nil
}

// GetSession retrieves a session by its ID.
func (sm *sessionManager) GetSession(ctx context.Context, sessionID string) (*interfaces.SessionInfo, error) {
	sm.sessionsMutex.RLock()
	defer sm.sessionsMutex.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		session.State = interfaces.SessionStateExpired
	}

	return session, nil
}

// UpdateSession updates an existing session.
func (sm *sessionManager) UpdateSession(ctx context.Context, sessionID string, options ...interfaces.SessionOption) (*interfaces.SessionInfo, error) {
	sm.sessionsMutex.Lock()
	defer sm.sessionsMutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Check if session is expired or closed
	if session.State == interfaces.SessionStateExpired || session.State == interfaces.SessionStateClosed {
		return nil, fmt.Errorf("cannot update %s session: %s", session.State, sessionID)
	}

	// Update last active time
	session.LastActiveAt = time.Now()
	
	// Set state to active
	session.State = interfaces.SessionStateActive

	// Apply options
	for _, option := range options {
		option(session)
	}

	// Notify listeners
	sm.notifyListeners(&interfaces.SessionEvent{
		Type:      interfaces.SessionUpdated,
		SessionID: sessionID,
		DeviceID:  session.DeviceID,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	})

	return session, nil
}

// CloseSession closes a session.
func (sm *sessionManager) CloseSession(ctx context.Context, sessionID string) error {
	sm.sessionsMutex.Lock()
	defer sm.sessionsMutex.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	// Update session state
	session.State = interfaces.SessionStateClosed

	// Notify listeners
	sm.notifyListeners(&interfaces.SessionEvent{
		Type:      interfaces.SessionClosed,
		SessionID: sessionID,
		DeviceID:  session.DeviceID,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	})

	// Remove from maps
	delete(sm.sessions, sessionID)
	
	// Remove from device sessions
	deviceID := session.DeviceID
	if sessions, exists := sm.deviceSessions[deviceID]; exists {
		for i, id := range sessions {
			if id == sessionID {
				sm.deviceSessions[deviceID] = append(sessions[:i], sessions[i+1:]...)
				break
			}
		}
		// If no more sessions for this device, remove the device entry
		if len(sm.deviceSessions[deviceID]) == 0 {
			delete(sm.deviceSessions, deviceID)
		}
	}

	return nil
}

// ListSessions lists all active sessions.
func (sm *sessionManager) ListSessions(ctx context.Context) ([]*interfaces.SessionInfo, error) {
	sm.sessionsMutex.RLock()
	defer sm.sessionsMutex.RUnlock()

	sessions := make([]*interfaces.SessionInfo, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		// Only include non-closed sessions
		if session.State != interfaces.SessionStateClosed {
			sessions = append(sessions, session)
		}
	}

	return sessions, nil
}

// ListSessionsByDevice lists all sessions for a specific device.
func (sm *sessionManager) ListSessionsByDevice(ctx context.Context, deviceID string) ([]*interfaces.SessionInfo, error) {
	sm.sessionsMutex.RLock()
	defer sm.sessionsMutex.RUnlock()

	sessionIDs, exists := sm.deviceSessions[deviceID]
	if !exists {
		return []*interfaces.SessionInfo{}, nil
	}

	sessions := make([]*interfaces.SessionInfo, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		if session, exists := sm.sessions[sessionID]; exists {
			// Only include non-closed sessions
			if session.State != interfaces.SessionStateClosed {
				sessions = append(sessions, session)
			}
		}
	}

	return sessions, nil
}

// CleanupExpiredSessions removes all expired sessions.
func (sm *sessionManager) CleanupExpiredSessions(ctx context.Context) (int, error) {
	sm.sessionsMutex.Lock()
	defer sm.sessionsMutex.Unlock()

	now := time.Now()
	count := 0

	// Find expired sessions
	expiredSessions := make([]string, 0)
	for sessionID, session := range sm.sessions {
		if now.After(session.ExpiresAt) {
			expiredSessions = append(expiredSessions, sessionID)
		}
	}

	// Process expired sessions
	for _, sessionID := range expiredSessions {
		session := sm.sessions[sessionID]
		
		// Update session state
		session.State = interfaces.SessionStateExpired

		// Notify listeners
		sm.notifyListeners(&interfaces.SessionEvent{
			Type:      interfaces.SessionExpired,
			SessionID: sessionID,
			DeviceID:  session.DeviceID,
			Timestamp: now,
			Data:      make(map[string]interface{}),
		})

		// Remove from maps
		delete(sm.sessions, sessionID)
		
		// Remove from device sessions
		deviceID := session.DeviceID
		if sessions, exists := sm.deviceSessions[deviceID]; exists {
			for i, id := range sessions {
				if id == sessionID {
					sm.deviceSessions[deviceID] = append(sessions[:i], sessions[i+1:]...)
					break
				}
			}
			// If no more sessions for this device, remove the device entry
			if len(sm.deviceSessions[deviceID]) == 0 {
				delete(sm.deviceSessions, deviceID)
			}
		}

		count++
	}

	return count, nil
}

// SetSessionTimeout sets the default timeout duration for new sessions.
func (sm *sessionManager) SetSessionTimeout(duration time.Duration) {
	sm.sessionsMutex.Lock()
	defer sm.sessionsMutex.Unlock()
	sm.sessionTimeout = duration
}

// GetSessionTimeout gets the current default timeout duration.
func (sm *sessionManager) GetSessionTimeout() time.Duration {
	sm.sessionsMutex.RLock()
	defer sm.sessionsMutex.RUnlock()
	return sm.sessionTimeout
}

// RegisterSessionListener registers a listener for session events.
func (sm *sessionManager) RegisterSessionListener(listener interfaces.SessionEventListener) {
	sm.sessionsMutex.Lock()
	defer sm.sessionsMutex.Unlock()
	sm.listeners = append(sm.listeners, listener)
}

// UnregisterSessionListener unregisters a session event listener.
func (sm *sessionManager) UnregisterSessionListener(listener interfaces.SessionEventListener) {
	sm.sessionsMutex.Lock()
	defer sm.sessionsMutex.Unlock()
	
	for i, l := range sm.listeners {
		if l == listener {
			sm.listeners = append(sm.listeners[:i], sm.listeners[i+1:]...)
			break
		}
	}
}

// notifyListeners notifies all registered listeners about a session event.
func (sm *sessionManager) notifyListeners(event *interfaces.SessionEvent) {
	for _, listener := range sm.listeners {
		go listener.OnSessionEvent(event)
	}
}

// cleanupRoutine periodically cleans up expired sessions.
func (sm *sessionManager) cleanupRoutine() {
	ticker := time.NewTicker(sm.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_, _ = sm.CleanupExpiredSessions(context.Background())
		case <-sm.stopCleanup:
			return
		}
	}
}

// getAllSessions returns all sessions (internal method for cleaner and monitor).
func (sm *sessionManager) getAllSessions() map[string]*interfaces.SessionInfo {
	sm.sessionsMutex.RLock()
	defer sm.sessionsMutex.RUnlock()
	
	// Return a copy to avoid race conditions
	sessions := make(map[string]*interfaces.SessionInfo)
	for id, session := range sm.sessions {
		sessions[id] = session
	}
	return sessions
}

// emitSessionEvent emits a session event to all listeners.
func (sm *sessionManager) emitSessionEvent(eventType interfaces.SessionEventType, sessionID string, deviceID string) {
	event := &interfaces.SessionEvent{
		Type:      eventType,
		SessionID: sessionID,
		DeviceID:  deviceID,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	}
	sm.notifyListeners(event)
}

// Stop stops the session manager and its components.
func (sm *sessionManager) Stop() {
	close(sm.stopCleanup)
	if sm.monitor != nil {
		sm.monitor.Stop()
	}
	if sm.cleaner != nil {
		sm.cleaner.Stop()
	}
}