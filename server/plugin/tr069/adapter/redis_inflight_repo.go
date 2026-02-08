package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/redis/go-redis/v9"
)

type RedisInflightRepo struct {
	client redis.UniversalClient
	ttl    time.Duration
	now    func() time.Time
}

func NewRedisInflightRepo(ttl time.Duration) *RedisInflightRepo {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &RedisInflightRepo{
		client: global.GVA_REDIS,
		ttl:    ttl,
		now:    time.Now,
	}
}

func (r *RedisInflightRepo) Save(ctx context.Context, req core.InflightRequest) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if r == nil || r.client == nil {
		return errors.New("redis not initialized")
	}
	if req.DeviceKey == "" || req.CwmpID == "" {
		return nil
	}
	if req.SentAt.IsZero() {
		req.SentAt = r.now()
	}
	key := RedisInflightHashPrefix + req.DeviceKey
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	pipe := r.client.TxPipeline()
	pipe.HSet(ctx, key, req.CwmpID, b)
	if r.ttl > 0 {
		pipe.Expire(ctx, key, r.ttl)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisInflightRepo) GetByCwmpID(ctx context.Context, deviceKey string, cwmpID string) (core.InflightRequest, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if r == nil || r.client == nil {
		return core.InflightRequest{}, false, errors.New("redis not initialized")
	}
	if deviceKey == "" || cwmpID == "" {
		return core.InflightRequest{}, false, nil
	}
	key := RedisInflightHashPrefix + deviceKey
	val, err := r.client.HGet(ctx, key, cwmpID).Bytes()
	if errors.Is(err, redis.Nil) {
		return core.InflightRequest{}, false, nil
	}
	if err != nil {
		return core.InflightRequest{}, false, err
	}
	var out core.InflightRequest
	if err := json.Unmarshal(val, &out); err != nil {
		_ = r.client.HDel(ctx, key, cwmpID).Err()
		return core.InflightRequest{}, false, nil
	}
	return out, true, nil
}

func (r *RedisInflightRepo) DeleteByCwmpID(ctx context.Context, deviceKey string, cwmpID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if r == nil || r.client == nil {
		return errors.New("redis not initialized")
	}
	if deviceKey == "" || cwmpID == "" {
		return nil
	}
	key := RedisInflightHashPrefix + deviceKey
	return r.client.HDel(ctx, key, cwmpID).Err()
}

var _ core.InflightRepo = (*RedisInflightRepo)(nil)
