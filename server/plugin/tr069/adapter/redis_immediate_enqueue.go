package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/redis/go-redis/v9"
)

type ImmediateEnqueueConfig struct {
	TTL time.Duration
}

func EnqueueImmediate(ctx context.Context, p DispatcherPayload, cfg ImmediateEnqueueConfig) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if global.GVA_REDIS == nil {
		return errors.New("redis not initialized")
	}
	if p.DeviceKey == "" || p.CommandID == "" || p.Op == "" {
		return errors.New("invalid payload")
	}
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	key := RedisImmediateListPrefix + p.DeviceKey
	if err := global.GVA_REDIS.RPush(ctx, key, b).Err(); err != nil {
		return err
	}
	if cfg.TTL > 0 {
		_ = global.GVA_REDIS.Expire(ctx, key, cfg.TTL).Err()
	}
	return nil
}

func IsRedisNil(err error) bool { return errors.Is(err, redis.Nil) }
