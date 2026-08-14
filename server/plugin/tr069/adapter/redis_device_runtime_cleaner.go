package adapter

import (
	"context"
	"errors"
	"strings"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/redis/go-redis/v9"
)

type RedisDeviceRuntimeCleaner struct {
	client     redis.UniversalClient
	identities *UploadIdentityStore
}

func NewRedisDeviceRuntimeCleaner(client redis.UniversalClient) *RedisDeviceRuntimeCleaner {
	return &RedisDeviceRuntimeCleaner{client: client, identities: NewUploadIdentityStore(client)}
}

var deleteMatchingTempSession = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

func (c *RedisDeviceRuntimeCleaner) Purge(ctx context.Context, identity service.DeviceRuntimeIdentity) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil || c.client == nil {
		return errors.New("device runtime Redis client is required")
	}
	identity.DeviceKey = strings.TrimSpace(identity.DeviceKey)
	if identity.DeviceID == 0 || identity.DeviceKey == "" {
		return errors.New("device runtime identity is incomplete")
	}
	pipe := c.client.TxPipeline()
	pipe.LRem(ctx, RedisCommandWakeupKey, 0, identity.DeviceKey)
	pipe.Del(ctx,
		RedisPendingListPrefix+identity.DeviceKey,
		RedisDeviceLockPrefix+identity.DeviceKey,
		RedisInflightHashPrefix+identity.DeviceKey,
		redisSessionKey(identity.DeviceKey),
		redisLockKey(identity.DeviceKey),
	)
	_, pipelineErr := pipe.Exec(ctx)

	var tempErr error
	var identityErr error
	if strings.TrimSpace(identity.IP) != "" {
		tempErr = deleteMatchingTempSession.Run(ctx, c.client, []string{redisTempKey(strings.TrimSpace(identity.IP))}, identity.DeviceKey).Err()
		identityErr = c.identities.Unbind(ctx, identity.IP, identity.DeviceID)
	}
	return errors.Join(pipelineErr, tempErr, identityErr)
}

var _ service.DeviceRuntimeCleaner = (*RedisDeviceRuntimeCleaner)(nil)
