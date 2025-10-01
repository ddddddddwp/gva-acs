// Package interfaces defines the interfaces for TR069 protocol components.
package interfaces

import (
	"context"
	"time"
)

// SessionState represents the current state of a TR069 session.
type SessionState string

const (
	// SessionStateNew indicates a newly created session.
	SessionStateNew SessionState = "new"
	// SessionStateActive indicates an active session.
	SessionStateActive SessionState = "active"
	// SessionStateIdle indicates an idle session.
	SessionStateIdle SessionState = "idle"
	// SessionStateExpired indicates an expired session.
	SessionStateExpired SessionState = "expired"
	// SessionStateClosed indicates a closed session.
	SessionStateClosed SessionState = "closed"
	// SessionStateError indicates a session in error state.
	SessionStateError SessionState = "error"
)

// SessionInfo contains information about a TR069 session.
type SessionInfo struct {
	// ID is the unique identifier for the session.
	ID string
	// DeviceID is the identifier of the device associated with the session.
	DeviceID string
	// State is the current state of the session.
	State SessionState
	// CreatedAt is the time when the session was created.
	CreatedAt time.Time
	// LastActiveAt is the time when the session was last active.
	LastActiveAt time.Time
	// ExpiresAt is the time when the session will expire.
	ExpiresAt time.Time
	// Metadata contains additional session metadata.
	Metadata map[string]interface{}
}

// SessionManager defines the interface for managing TR069 sessions.
type SessionManager interface {
	// CreateSession creates a new session for the specified device.
	CreateSession(ctx context.Context, deviceID string, options ...SessionOption) (*SessionInfo, error)
	
	// GetSession retrieves a session by its ID.
	GetSession(ctx context.Context, sessionID string) (*SessionInfo, error)
	
	// UpdateSession updates an existing session.
	UpdateSession(ctx context.Context, sessionID string, options ...SessionOption) (*SessionInfo, error)
	
	// CloseSession closes a session.
	CloseSession(ctx context.Context, sessionID string) error
	
	// ListSessions lists all active sessions.
	ListSessions(ctx context.Context) ([]*SessionInfo, error)
	
	// ListSessionsByDevice lists all sessions for a specific device.
	ListSessionsByDevice(ctx context.Context, deviceID string) ([]*SessionInfo, error)
	
	// CleanupExpiredSessions removes all expired sessions.
	CleanupExpiredSessions(ctx context.Context) (int, error)
	
	// SetSessionTimeout sets the default timeout duration for new sessions.
	SetSessionTimeout(duration time.Duration)
	
	// GetSessionTimeout gets the current default timeout duration.
	GetSessionTimeout() time.Duration
	
	// RegisterSessionListener registers a listener for session events.
	RegisterSessionListener(listener SessionEventListener)
	
	// UnregisterSessionListener unregisters a session event listener.
	UnregisterSessionListener(listener SessionEventListener)
}

// SessionOption is a function that configures a session.
type SessionOption func(*SessionInfo)

// WithSessionTimeout sets the timeout duration for a session.
func WithSessionTimeout(duration time.Duration) SessionOption {
	return func(info *SessionInfo) {
		info.ExpiresAt = time.Now().Add(duration)
	}
}

// WithSessionMetadata adds metadata to a session.
func WithSessionMetadata(key string, value interface{}) SessionOption {
	return func(info *SessionInfo) {
		if info.Metadata == nil {
			info.Metadata = make(map[string]interface{})
		}
		info.Metadata[key] = value
	}
}

// SessionEventType represents the type of session event.
type SessionEventType string

const (
	// SessionCreated indicates a session was created.
	SessionCreated SessionEventType = "created"
	// SessionUpdated indicates a session was updated.
	SessionUpdated SessionEventType = "updated"
	// SessionClosed indicates a session was closed.
	SessionClosed SessionEventType = "closed"
	// SessionExpired indicates a session has expired.
	SessionExpired SessionEventType = "expired"
	// SessionError indicates a session encountered an error.
	SessionError SessionEventType = "error"
	// SessionEventTimeout indicates a session timed out.
	SessionEventTimeout SessionEventType = "timeout"
	// SessionEventInactive indicates a session became inactive.
	SessionEventInactive SessionEventType = "inactive"
	// SessionEventHighLatency indicates a session has high latency.
	SessionEventHighLatency SessionEventType = "high_latency"
)

// SessionEvent represents an event related to a session.
type SessionEvent struct {
	// Type is the type of the event.
	Type SessionEventType
	// SessionID is the ID of the session.
	SessionID string
	// DeviceID is the ID of the device.
	DeviceID string
	// Timestamp is the time when the event occurred.
	Timestamp time.Time
	// Data contains additional event data.
	Data map[string]interface{}
}

// SessionEventListener is the interface for objects that listen to session events.
type SessionEventListener interface {
	// OnSessionEvent is called when a session event occurs.
	OnSessionEvent(event *SessionEvent)
}