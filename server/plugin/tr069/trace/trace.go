package trace

import (
	"context"
	"sync"
	"time"
)

type Entry struct {
	At      time.Time         `json:"at"`
	Stage   string            `json:"stage"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type Store struct {
	mu         sync.Mutex
	byRequest  map[string][]Entry
	maxEntries int
	maxAge     time.Duration
}

func NewStore(maxEntries int, maxAge time.Duration) *Store {
	if maxEntries <= 0 {
		maxEntries = 200
	}
	if maxAge <= 0 {
		maxAge = 30 * time.Minute
	}
	return &Store{
		byRequest:  map[string][]Entry{},
		maxEntries: maxEntries,
		maxAge:     maxAge,
	}
}

func (s *Store) Add(requestID string, e Entry) {
	if s == nil || requestID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := e.At
	if now.IsZero() {
		now = time.Now()
		e.At = now
	}

	for id, entries := range s.byRequest {
		if len(entries) == 0 {
			delete(s.byRequest, id)
			continue
		}
		last := entries[len(entries)-1].At
		if last.IsZero() {
			last = now
		}
		if now.Sub(last) > s.maxAge {
			delete(s.byRequest, id)
		}
	}

	entries := append(s.byRequest[requestID], e)
	if len(entries) > s.maxEntries {
		entries = entries[len(entries)-s.maxEntries:]
	}
	s.byRequest[requestID] = entries
}

func (s *Store) Get(requestID string) []Entry {
	if s == nil || requestID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	entries := s.byRequest[requestID]
	if len(entries) == 0 {
		return nil
	}
	out := make([]Entry, len(entries))
	copy(out, entries)
	return out
}

var Default = NewStore(200, 30*time.Minute)

type requestIDKey struct{}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if requestID == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDKey{}, requestID)
}

func RequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v := ctx.Value(requestIDKey{})
	id, _ := v.(string)
	return id
}

func Add(ctx context.Context, stage, message string, fields map[string]string) {
	id := RequestID(ctx)
	if id == "" {
		return
	}
	Default.Add(id, Entry{
		At:      time.Now(),
		Stage:   stage,
		Message: message,
		Fields:  fields,
	})
}

func Get(ctx context.Context, requestID string) []Entry {
	return Default.Get(requestID)
}

