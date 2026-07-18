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

// Store 是 TR069 调试用 Trace 存储（内存环形队列风格）：
// - key = traceId（来自 X-Request-ID 或自动生成的 UUID）
// - value = 按时间顺序追加的阶段记录（解析、入库、下发等）
// 删除/禁用：不影响核心业务，移除 trace.Add/trace.WithTraceID 调用以及 /tr069/debug/trace 接口即可。
type Store struct {
	mu         sync.Mutex
	byTrace    map[string][]Entry
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
		byTrace:    map[string][]Entry{},
		maxEntries: maxEntries,
		maxAge:     maxAge,
	}
}

func (s *Store) Add(traceID string, e Entry) {
	if s == nil || traceID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := e.At
	if now.IsZero() {
		now = time.Now()
		e.At = now
	}

	for id, entries := range s.byTrace {
		if len(entries) == 0 {
			delete(s.byTrace, id)
			continue
		}
		last := entries[len(entries)-1].At
		if last.IsZero() {
			last = now
		}
		if now.Sub(last) > s.maxAge {
			delete(s.byTrace, id)
		}
	}

	entries := append(s.byTrace[traceID], e)
	if len(entries) > s.maxEntries {
		entries = entries[len(entries)-s.maxEntries:]
	}
	s.byTrace[traceID] = entries
}

func (s *Store) Get(traceID string) []Entry {
	if s == nil || traceID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	entries := s.byTrace[traceID]
	if len(entries) == 0 {
		return nil
	}
	out := make([]Entry, len(entries))
	copy(out, entries)
	return out
}

var Default = NewStore(200, 30*time.Minute)

type traceIDKey struct{}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if traceID == "" {
		return ctx
	}
	return context.WithValue(ctx, traceIDKey{}, traceID)
}

func TraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v := ctx.Value(traceIDKey{})
	id, _ := v.(string)
	return id
}

func Add(ctx context.Context, stage, message string, fields map[string]string) {
	id := TraceID(ctx)
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

func Get(ctx context.Context, traceID string) []Entry {
	return Default.Get(traceID)
}
