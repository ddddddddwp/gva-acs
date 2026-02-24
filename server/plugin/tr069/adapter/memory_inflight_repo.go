package adapter

import (
	"context"
	"sync"
	"time"

	"github.com/ddddddddwp/tr069-core-only/pkg/core"
)

type MemoryInflightRepo struct {
	mu  sync.Mutex
	ttl time.Duration
	now func() time.Time
	m   map[string]core.InflightRequest
}

func NewMemoryInflightRepo(ttl time.Duration) *MemoryInflightRepo {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &MemoryInflightRepo{
		ttl: ttl,
		now: time.Now,
		m:   map[string]core.InflightRequest{},
	}
}

func (r *MemoryInflightRepo) Save(ctx context.Context, req core.InflightRequest) error {
	_, _ = ctx, req
	if req.DeviceKey == "" || req.CwmpID == "" {
		return nil // 空key不做处理
	}
	if req.SentAt.IsZero() {
		req.SentAt = r.now()
	}
	r.mu.Lock()
	r.m[r.key(req.DeviceKey, req.CwmpID)] = req
	r.mu.Unlock()
	return nil
}

func (r *MemoryInflightRepo) GetByCwmpID(ctx context.Context, deviceKey string, cwmpID string) (core.InflightRequest, bool, error) {
	_ = ctx
	if deviceKey == "" || cwmpID == "" {
		return core.InflightRequest{}, false, nil
	}
	k := r.key(deviceKey, cwmpID)
	now := r.now()

	r.mu.Lock()
	req, ok := r.m[k]
	if ok && r.ttl > 0 && !req.SentAt.IsZero() && now.Sub(req.SentAt) > r.ttl {
		delete(r.m, k)
		ok = false
		req = core.InflightRequest{}
	}
	r.mu.Unlock()
	return req, ok, nil
}

func (r *MemoryInflightRepo) DeleteByCwmpID(ctx context.Context, deviceKey string, cwmpID string) error {
	_ = ctx
	if deviceKey == "" || cwmpID == "" {
		return nil
	}
	r.mu.Lock()
	delete(r.m, r.key(deviceKey, cwmpID))
	r.mu.Unlock()
	return nil
}

func (r *MemoryInflightRepo) key(deviceKey string, cwmpID string) string {
	// 使用 Unicode Record Separator (0x1e) 作为分隔符，避免键冲突
	// 避免使用 "|" 等常见字符作为分隔符
	return deviceKey + "\x1e" + cwmpID
}

var _ core.InflightRepo = (*MemoryInflightRepo)(nil)
