package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/redis/go-redis/v9"
)

type RedisCommandSourceConfig struct {
	LockTTL    time.Duration
	DedupTTL   time.Duration
	MaxScan    int
	InstanceID string
}

type RedisCommandSource struct {
	client redis.UniversalClient
	cfg    RedisCommandSourceConfig
}

func NewRedisCommandSource(cfg RedisCommandSourceConfig) *RedisCommandSource {
	instanceID := cfg.InstanceID
	if instanceID == "" {
		hostname, _ := os.Hostname()
		instanceID = "gva-" + hostname + "-" + strconv.Itoa(os.Getpid())
	}
	if cfg.LockTTL <= 0 {
		cfg.LockTTL = 30 * time.Second
	}
	if cfg.DedupTTL <= 0 {
		cfg.DedupTTL = 24 * time.Hour
	}
	if cfg.MaxScan <= 0 {
		cfg.MaxScan = 10
	}
	cfg.InstanceID = instanceID
	return &RedisCommandSource{
		client: global.GVA_REDIS,
		cfg:    cfg,
	}
}

var unlockLockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

func (s *RedisCommandSource) Pull(ctx context.Context, deviceKey string) (*core.Command, core.AckFunc, core.NackFunc, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if s == nil || s.client == nil || deviceKey == "" {
		return nil, nil, nil, nil
	}

	lockKey := RedisDeviceLockPrefix + deviceKey
	ok, err := s.client.SetNX(ctx, lockKey, s.cfg.InstanceID, s.cfg.LockTTL).Result()
	if err != nil || !ok {
		return nil, nil, nil, nil
	}

	immediateKey := RedisImmediateListPrefix + deviceKey
	pendingKey := RedisPendingListPrefix + deviceKey

	var payload []byte
	var parsed DispatcherPayload
	var cmd *core.Command
	selectedKey := pendingKey

	for i := 0; i < s.cfg.MaxScan; i++ {
		val, err := s.client.LPop(ctx, immediateKey).Bytes()
		selectedKey = immediateKey
		if errors.Is(err, redis.Nil) {
			val, err = s.client.LPop(ctx, pendingKey).Bytes()
			selectedKey = pendingKey
		}
		if errors.Is(err, redis.Nil) {
			_ = unlockLockScript.Run(ctx, s.client, []string{lockKey}, s.cfg.InstanceID).Err()
			return nil, nil, nil, nil
		}
		if err != nil {
			_ = unlockLockScript.Run(ctx, s.client, []string{lockKey}, s.cfg.InstanceID).Err()
			return nil, nil, nil, err
		}

		payload = val
		if err := json.Unmarshal(payload, &parsed); err != nil {
			continue
		}
		if parsed.DeviceKey == "" {
			parsed.DeviceKey = deviceKey
		}
		if parsed.DedupKey != "" {
			dedupKey := RedisDedupPrefix + parsed.DedupKey
			seen, err := s.client.SetNX(ctx, dedupKey, "1", s.cfg.DedupTTL).Result()
			if err == nil && !seen {
				payload = nil
				parsed = DispatcherPayload{}
				continue
			}
		}

		cmd = &core.Command{
			ID:        parsed.CommandID,
			DeviceKey: parsed.DeviceKey,
			Operation: parsed.Op,
			DedupKey:  parsed.DedupKey,
			Params:    map[string]interface{}{},
			CreatedAt: time.Now(),
		}
		if parsed.Params != "" {
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(parsed.Params), &m); err == nil {
				cmd.Params = m
			}
		}
		break
	}

	if cmd == nil {
		_ = unlockLockScript.Run(ctx, s.client, []string{lockKey}, s.cfg.InstanceID).Err()
		return nil, nil, nil, nil
	}

	ack := func(ctx context.Context) error {
		if ctx == nil {
			ctx = context.Background()
		}
		return unlockLockScript.Run(ctx, s.client, []string{lockKey}, s.cfg.InstanceID).Err()
	}
	nack := func(ctx context.Context, reason string) error {
		_, _ = reason, payload
		if ctx == nil {
			ctx = context.Background()
		}
		if len(payload) > 0 {
			_ = s.client.LPush(ctx, selectedKey, payload).Err()
		}
		return unlockLockScript.Run(ctx, s.client, []string{lockKey}, s.cfg.InstanceID).Err()
	}
	return cmd, ack, nack, nil
}

var _ core.CommandSource = (*RedisCommandSource)(nil)
