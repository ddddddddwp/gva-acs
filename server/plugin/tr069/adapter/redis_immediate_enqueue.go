package adapter

import (
	"context"
	"errors"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/redis/go-redis/v9"
)

type commandWakeClient interface {
	RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
}

// EnqueueImmediate records only a duplicate-safe device wake-up. The database
// command head remains the authoritative source selected by RedisCommandSource.
func EnqueueImmediate(ctx context.Context, deviceKey string) error {
	if global.GVA_REDIS == nil {
		return errors.New("redis not initialized")
	}
	return enqueueImmediate(ctx, global.GVA_REDIS, deviceKey)
}

func enqueueImmediate(ctx context.Context, client commandWakeClient, deviceKey string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if client == nil {
		return errors.New("redis not initialized")
	}
	if deviceKey == "" {
		return errors.New("deviceKey cannot be empty")
	}
	return client.RPush(ctx, RedisCommandWakeupKey, deviceKey).Err()
}

func IsRedisNil(err error) bool { return errors.Is(err, redis.Nil) }
