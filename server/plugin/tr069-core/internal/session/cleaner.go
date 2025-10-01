// Package session provides implementation for TR069 session management.
package session

import (
	"context"
	"sync"
	"time"

	"github.com/root/demo/tr069/interfaces"
)

// SessionCleaner handles automatic cleanup of expired sessions.
type SessionCleaner struct {
	manager           *sessionManager
	cleanupInterval   time.Duration
	idleTimeout       time.Duration
	maxSessionAge     time.Duration
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	cleanupInProgress bool
	cleanupMutex      sync.Mutex
}

// NewSessionCleaner creates a new session cleaner.
func NewSessionCleaner(manager *sessionManager) *SessionCleaner {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &SessionCleaner{
		manager:         manager,
		cleanupInterval: 5 * time.Minute,  // Default cleanup interval
		idleTimeout:     30 * time.Minute, // Default idle timeout
		maxSessionAge:   24 * time.Hour,   // Default max session age
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start starts the session cleaner.
func (c *SessionCleaner) Start() {
	c.wg.Add(1)
	go c.cleanupLoop()
}

// Stop stops the session cleaner.
func (c *SessionCleaner) Stop() {
	c.cancel()
	c.wg.Wait()
}

// SetCleanupInterval sets the interval for session cleanup.
func (c *SessionCleaner) SetCleanupInterval(interval time.Duration) {
	if interval < time.Minute {
		interval = time.Minute
	}
	c.cleanupInterval = interval
}

// SetIdleTimeout sets the idle timeout for sessions.
func (c *SessionCleaner) SetIdleTimeout(timeout time.Duration) {
	if timeout < time.Minute {
		timeout = time.Minute
	}
	c.idleTimeout = timeout
}

// SetMaxSessionAge sets the maximum age for sessions.
func (c *SessionCleaner) SetMaxSessionAge(age time.Duration) {
	if age < time.Hour {
		age = time.Hour
	}
	c.maxSessionAge = age
}

// RunCleanup manually runs a cleanup cycle.
func (c *SessionCleaner) RunCleanup() int {
	c.cleanupMutex.Lock()
	if c.cleanupInProgress {
		c.cleanupMutex.Unlock()
		return 0
	}
	c.cleanupInProgress = true
	c.cleanupMutex.Unlock()
	
	defer func() {
		c.cleanupMutex.Lock()
		c.cleanupInProgress = false
		c.cleanupMutex.Unlock()
	}()
	
	return c.cleanupExpiredSessions()
}

// cleanupLoop periodically cleans up expired sessions.
func (c *SessionCleaner) cleanupLoop() {
	defer c.wg.Done()
	
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.RunCleanup()
		}
	}
}

// cleanupExpiredSessions cleans up expired sessions.
func (c *SessionCleaner) cleanupExpiredSessions() int {
	now := time.Now()
	idleDeadline := now.Add(-c.idleTimeout)
	ageDeadline := now.Add(-c.maxSessionAge)
	
	// Get all sessions
	sessions := c.manager.getAllSessions()
	
	// Track sessions to close
	sessionsToClose := make([]string, 0)
	
	// Check each session
	for sessionID, session := range sessions {
		// Skip already closed sessions
		if session.State == interfaces.SessionStateClosed {
			continue
		}
		
		// Check for idle timeout
		if session.LastActiveAt.Before(idleDeadline) {
			sessionsToClose = append(sessionsToClose, sessionID)
			continue
		}
		
		// Check for max age
		if session.CreatedAt.Before(ageDeadline) {
			sessionsToClose = append(sessionsToClose, sessionID)
			continue
		}
	}
	
	// Close expired sessions
	for _, sessionID := range sessionsToClose {
		// Emit timeout event before closing
		if session, err := c.manager.GetSession(context.Background(), sessionID); err == nil {
			c.manager.emitSessionEvent(interfaces.SessionEventTimeout, sessionID, session.DeviceID)
		}
		
		// Close the session
		_ = c.manager.CloseSession(context.Background(), sessionID)
	}
	
	return len(sessionsToClose)
}