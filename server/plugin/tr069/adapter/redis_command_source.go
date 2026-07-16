package adapter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/config"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type RedisCommandSourceConfig struct {
	LockTTL              time.Duration
	DedupTTL             time.Duration
	MaxScan              int
	InstanceID           string
	MaxPendingPerSession int
}

type commandDeviceLocker interface {
	Lock(ctx context.Context, key, owner string, ttl time.Duration) (bool, error)
	Unlock(ctx context.Context, key, owner string) error
}

type redisCommandDeviceLocker struct {
	client redis.UniversalClient
}

var ErrRedisLockOwnershipLost = errors.New("redis device lock ownership lost")

func (l redisCommandDeviceLocker) Lock(ctx context.Context, key, owner string, ttl time.Duration) (bool, error) {
	return l.client.SetNX(ctx, key, owner, ttl).Result()
}

func (l redisCommandDeviceLocker) Unlock(ctx context.Context, key, owner string) error {
	deleted, err := unlockLockScript.Run(ctx, l.client, []string{key}, owner).Int64()
	if err != nil {
		return fmt.Errorf("release Redis device lock %q: %w", key, err)
	}
	return validateRedisLockRelease(key, deleted)
}

func validateRedisLockRelease(key string, deleted int64) error {
	if deleted != 1 {
		return fmt.Errorf("%w: %s", ErrRedisLockOwnershipLost, key)
	}
	return nil
}

type RedisCommandSource struct {
	db     *gorm.DB
	store  *service.CommandStore
	locker commandDeviceLocker
	cfg    RedisCommandSourceConfig
}

func NewRedisCommandSource(cfg RedisCommandSourceConfig) (*RedisCommandSource, error) {
	if global.GVA_REDIS == nil {
		return nil, errors.New("Redis client not initialized")
	}
	if global.GVA_DB == nil {
		return nil, errors.New("database not initialized")
	}
	return newRedisCommandSource(global.GVA_DB, redisCommandDeviceLocker{client: global.GVA_REDIS}, cfg), nil
}

func newRedisCommandSource(db *gorm.DB, locker commandDeviceLocker, cfg RedisCommandSourceConfig) *RedisCommandSource {
	if cfg.InstanceID == "" {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		cfg.InstanceID = "gva-" + hostname + "-" + strconv.Itoa(os.Getpid())
	}
	if cfg.LockTTL <= 0 {
		cfg.LockTTL = 30 * time.Second
	}
	return &RedisCommandSource{db: db, store: service.NewCommandStore(db), locker: locker, cfg: cfg}
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
	if s == nil || s.db == nil || s.store == nil || s.locker == nil {
		return nil, nil, nil, errors.New("RedisCommandSource not initialized")
	}
	if deviceKey == "" {
		return nil, nil, nil, errors.New("deviceKey cannot be empty")
	}

	lockKey := RedisDeviceLockPrefix + deviceKey
	owner := s.cfg.InstanceID + ":" + uuid.NewString()
	locked, err := s.locker.Lock(ctx, lockKey, owner, s.cfg.LockTTL)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to acquire device lock: %w", err)
	}
	if !locked {
		return nil, nil, nil, nil
	}
	unlock := func(ctx context.Context) error {
		if ctx == nil {
			ctx = context.Background()
		}
		return s.locker.Unlock(ctx, lockKey, owner)
	}

	var head model.Command
	err = s.db.WithContext(ctx).
		Where("device_key = ? AND status IN ?", deviceKey, model.NonTerminalCommandStatuses()).
		Order("created_at ASC").
		Order("command_id ASC").
		First(&head).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, nil, errors.Join(unlock(ctx))
	}
	if err != nil {
		return nil, nil, nil, errors.Join(err, unlock(ctx))
	}
	if head.Status != model.CommandStatusWaitingDevice {
		return nil, nil, nil, errors.Join(unlock(ctx))
	}

	buildingAt := time.Now()
	building, err := s.store.Transition(ctx, service.CommandTransition{
		CommandID:       head.CommandID,
		FromStatuses:    []string{model.CommandStatusWaitingDevice},
		ToStatus:        model.CommandStatusBuilding,
		ExpectedVersion: head.Version,
		EventType:       "BUILDING_STARTED",
		Stage:           "core.build",
		Updates: map[string]any{
			"building_at":       buildingAt,
			"phase_deadline_at": nil,
		},
	})
	if errors.Is(err, service.ErrCommandTransitionConflict) {
		return nil, nil, nil, errors.Join(unlock(ctx))
	}
	if err != nil {
		return nil, nil, nil, errors.Join(err, unlock(ctx))
	}

	ack := core.AckFunc(func(ctx context.Context) error {
		return unlock(ctx)
	})
	nack := core.NackFunc(func(ctx context.Context, reason string) error {
		return s.nackBuilding(ctx, building, reason, unlock)
	})

	params, err := service.DecodeRPCParams(building.Operation, building.ParamsJSON)
	if err != nil {
		return nil, nil, nil, errors.Join(err, nack(ctx, "decode persisted command params"))
	}
	if spec, ok := service.RPCSpecs[building.Operation]; ok && spec.Transfer && building.CommandKey != nil {
		params["commandKey"] = *building.CommandKey
	}

	return &core.Command{
		ID:        building.CommandID,
		DeviceKey: building.DeviceKey,
		Operation: building.Operation,
		Params:    params,
		DedupKey:  building.DedupKey,
		CreatedAt: building.CreatedAt,
	}, ack, nack, nil
}

func (s *RedisCommandSource) nackBuilding(ctx context.Context, building model.Command, reason string, unlock func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	var current model.Command
	readErr := s.db.WithContext(ctx).First(&current, "command_id = ?", building.CommandID).Error
	var transitionErr error
	if readErr == nil && current.Status == model.CommandStatusBuilding && current.Version == building.Version {
		now := time.Now()
		deadline := now.Add(config.CurrentRuntime().CommandQueueWaitTimeout)
		_, transitionErr = s.store.Transition(ctx, service.CommandTransition{
			CommandID:       current.CommandID,
			FromStatuses:    []string{model.CommandStatusBuilding},
			ToStatus:        model.CommandStatusWaitingDevice,
			ExpectedVersion: building.Version,
			EventType:       "BUILDING_REQUEUED",
			Stage:           "core.build",
			Message:         reason,
			Updates: map[string]any{
				"building_at":       nil,
				"waiting_at":        now,
				"phase_deadline_at": deadline,
			},
		})
	}
	return errors.Join(readErr, transitionErr, unlock(ctx))
}

var _ core.CommandSource = (*RedisCommandSource)(nil)
