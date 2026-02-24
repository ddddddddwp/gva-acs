package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RedisCommandSourceConfig struct {
	LockTTL              time.Duration
	DedupTTL             time.Duration
	MaxScan              int
	InstanceID           string
	MaxPendingPerSession int // 限制每次 session 处理的 pending 消息数量，避免饥饿
}

type RedisCommandSource struct {
	client redis.UniversalClient
	cfg    RedisCommandSourceConfig
}

func NewRedisCommandSource(cfg RedisCommandSourceConfig) (*RedisCommandSource, error) {
	if global.GVA_REDIS == nil {
		return nil, errors.New("Redis client not initialized")
	}

	instanceID := cfg.InstanceID
	if instanceID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
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
	if cfg.MaxPendingPerSession <= 0 {
		cfg.MaxPendingPerSession = 5
	}
	cfg.InstanceID = instanceID
	return &RedisCommandSource{
		client: global.GVA_REDIS,
		cfg:    cfg,
	}, nil
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
	if s == nil || s.client == nil {
		return nil, nil, nil, errors.New("RedisCommandSource not initialized")
	}
	if deviceKey == "" {
		return nil, nil, nil, errors.New("deviceKey cannot be empty")
	}

	lockKey := RedisDeviceLockPrefix + deviceKey
	ok, err := s.client.SetNX(ctx, lockKey, s.cfg.InstanceID, s.cfg.LockTTL).Result()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to acquire device lock: %w", err)
	}
	if !ok {
		return nil, nil, nil, nil // 锁已被其他实例持有
	}

	immediateKey := RedisImmediateListPrefix + deviceKey
	pendingKey := RedisPendingListPrefix + deviceKey

	// 1. 优先处理 immediate 队列（主动下发的命令）
	cmd := s.pullOneFromQueue(ctx, immediateKey, deviceKey)
	if cmd != nil {
		global.GVA_LOG.Info("TR069 REDIS PULL",
			zap.String("deviceKey", deviceKey),
			zap.String("source", "immediate"),
			zap.String("commandId", cmd.ID),
			zap.String("operation", cmd.Operation))
		ack := func(ctx context.Context) error {
			if ctx == nil {
				ctx = context.Background()
			}
			return unlockLockScript.Run(ctx, s.client, []string{lockKey}, s.cfg.InstanceID).Err()
		}
		nack := func(ctx context.Context, reason string) error {
			if ctx == nil {
				ctx = context.Background()
			}
			if err := s.client.RPush(ctx, immediateKey, s.marshalCommand(cmd)).Err(); err != nil {
				return err
			}
			return unlockLockScript.Run(ctx, s.client, []string{lockKey}, s.cfg.InstanceID).Err()
		}
		return cmd, ack, nack, nil
	}

	// 2. 限制处理 pending 队列的数量，避免饥饿
	pendingCount := 0
	for pendingCount < s.cfg.MaxPendingPerSession {
		cmd = s.pullOneFromQueue(ctx, pendingKey, deviceKey)
		if cmd == nil {
			break
		}
		pendingCount++
		global.GVA_LOG.Info("TR069 REDIS PULL",
			zap.String("deviceKey", deviceKey),
			zap.String("source", "pending"),
			zap.String("commandId", cmd.ID),
			zap.String("operation", cmd.Operation),
			zap.Int("batch", pendingCount),
			zap.Int("maxBatch", s.cfg.MaxPendingPerSession))

		ack := func(ctx context.Context) error {
			if ctx == nil {
				ctx = context.Background()
			}
			return unlockLockScript.Run(ctx, s.client, []string{lockKey}, s.cfg.InstanceID).Err()
		}
		nack := func(ctx context.Context, reason string) error {
			if ctx == nil {
				ctx = context.Background()
			}
			if err := s.client.RPush(ctx, pendingKey, s.marshalCommand(cmd)).Err(); err != nil {
				return err
			}
			return unlockLockScript.Run(ctx, s.client, []string{lockKey}, s.cfg.InstanceID).Err()
		}
		return cmd, ack, nack, nil
	}

	// 没有命令
	_ = unlockLockScript.Run(ctx, s.client, []string{lockKey}, s.cfg.InstanceID).Err()
	return nil, nil, nil, nil
}

func (s *RedisCommandSource) pullOneFromQueue(ctx context.Context, queueKey, deviceKey string) *core.Command {
	val, err := s.client.LPop(ctx, queueKey).Bytes()
	if errors.Is(err, redis.Nil) || len(val) == 0 {
		return nil
	}
	if err != nil {
		global.GVA_LOG.Error("failed to pop from queue", zap.Error(err), zap.String("queueKey", queueKey))
		return nil
	}

	var parsed DispatcherPayload
	if err := json.Unmarshal(val, &parsed); err != nil {
		return nil
	}

	if parsed.DeviceKey == "" {
		parsed.DeviceKey = deviceKey
	}

	// 去重检查
	if parsed.DedupKey != "" {
		dedupKey := RedisDedupPrefix + parsed.DedupKey
		seen, err := s.client.SetNX(ctx, dedupKey, "1", s.cfg.DedupTTL).Result()
		if err == nil && !seen {
			return nil // 已存在，跳过
		}
	}

	cmd := &core.Command{
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
	return cmd
}

func (s *RedisCommandSource) marshalCommand(cmd *core.Command) string {
	p := DispatcherPayload{
		DeviceKey: cmd.DeviceKey,
		CommandID: cmd.ID,
		Op:        cmd.Operation,
		DedupKey:  cmd.DedupKey,
	}
	if cmd.Params != nil {
		b, _ := json.Marshal(cmd.Params)
		p.Params = string(b)
	}
	b, _ := json.Marshal(p)
	return string(b)
}

var _ core.CommandSource = (*RedisCommandSource)(nil)
