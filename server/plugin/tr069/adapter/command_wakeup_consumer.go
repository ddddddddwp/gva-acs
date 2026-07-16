package adapter

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/model"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type wakeTokenSource interface {
	Pop(ctx context.Context) (string, error)
}

type redisWakeTokenSource struct {
	client redis.UniversalClient
	block  time.Duration
}

func (s redisWakeTokenSource) Pop(ctx context.Context) (string, error) {
	values, err := s.client.BLPop(ctx, s.block, RedisCommandWakeupKey).Result()
	if err != nil {
		return "", err
	}
	if len(values) != 2 {
		return "", nil
	}
	return values[1], nil
}

type connectionRequestTrigger func(context.Context, uint, ConnectionRequestConfig) ConnectionRequestResult

type CommandWakeConsumerConfig struct {
	BlockTimeout      time.Duration
	ErrorBackoff      time.Duration
	FailureBackoff    time.Duration
	ConnectionRequest ConnectionRequestConfig
}

type CommandWakeConsumer struct {
	db      *gorm.DB
	store   *service.CommandStore
	source  wakeTokenSource
	trigger connectionRequestTrigger
	cfg     CommandWakeConsumerConfig
}

var commandWakeConsumerLifecycle struct {
	sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}
}

// StartCommandWakeConsumer starts the Redis wake-token consumer once for this process.
func StartCommandWakeConsumer(ctx context.Context, cfg CommandWakeConsumerConfig) error {
	commandWakeConsumerLifecycle.Lock()
	defer commandWakeConsumerLifecycle.Unlock()
	if commandWakeConsumerLifecycle.cancel != nil {
		return nil
	}
	if global.GVA_DB == nil {
		return errors.New("start command wake consumer: database is not initialized")
	}
	if global.GVA_REDIS == nil {
		return errors.New("start command wake consumer: Redis is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cfg = normalizeCommandWakeConsumerConfig(cfg)
	consumer := newCommandWakeConsumer(
		global.GVA_DB,
		redisWakeTokenSource{client: global.GVA_REDIS, block: cfg.BlockTimeout},
		TriggerConnectionRequest,
		cfg,
	)
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	commandWakeConsumerLifecycle.cancel = cancel
	commandWakeConsumerLifecycle.done = done
	go func() {
		defer close(done)
		consumer.Run(runCtx)
	}()
	return nil
}

// StopCommandWakeConsumer cancels the consumer and waits for its BLPOP loop to exit.
func StopCommandWakeConsumer(ctx context.Context) error {
	commandWakeConsumerLifecycle.Lock()
	cancel := commandWakeConsumerLifecycle.cancel
	done := commandWakeConsumerLifecycle.done
	commandWakeConsumerLifecycle.Unlock()
	if cancel == nil || done == nil {
		return nil
	}
	cancel()
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-done:
		commandWakeConsumerLifecycle.Lock()
		if commandWakeConsumerLifecycle.done == done {
			commandWakeConsumerLifecycle.cancel = nil
			commandWakeConsumerLifecycle.done = nil
		}
		commandWakeConsumerLifecycle.Unlock()
		return nil
	case <-ctx.Done():
		return fmt.Errorf("stop command wake consumer: %w", ctx.Err())
	}
}

func newCommandWakeConsumer(db *gorm.DB, source wakeTokenSource, trigger connectionRequestTrigger, cfg CommandWakeConsumerConfig) *CommandWakeConsumer {
	cfg = normalizeCommandWakeConsumerConfig(cfg)
	return &CommandWakeConsumer{
		db: db, store: service.NewCommandStore(db), source: source, trigger: trigger, cfg: cfg,
	}
}

func normalizeCommandWakeConsumerConfig(cfg CommandWakeConsumerConfig) CommandWakeConsumerConfig {
	if cfg.BlockTimeout <= 0 {
		cfg.BlockTimeout = time.Second
	}
	if cfg.ErrorBackoff <= 0 {
		cfg.ErrorBackoff = 250 * time.Millisecond
	}
	if cfg.FailureBackoff <= 0 {
		cfg.FailureBackoff = 250 * time.Millisecond
	}
	if cfg.ConnectionRequest.Timeout <= 0 {
		cfg.ConnectionRequest.Timeout = 5 * time.Second
	}
	if cfg.ConnectionRequest.Retries <= 0 {
		cfg.ConnectionRequest.Retries = 1
	}
	return cfg
}

func (c *CommandWakeConsumer) Run(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}
	if c == nil || c.db == nil || c.store == nil || c.source == nil || c.trigger == nil {
		return
	}
	for {
		deviceKey, err := c.source.Pop(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			if errors.Is(err, redis.Nil) {
				continue
			}
			if !waitCommandWakeConsumer(ctx, c.cfg.ErrorBackoff) {
				return
			}
			continue
		}
		if deviceKey == "" {
			continue
		}
		if err := c.handleToken(ctx, deviceKey); err != nil {
			if !waitCommandWakeConsumer(ctx, c.cfg.FailureBackoff) {
				return
			}
		}
	}
}

func (c *CommandWakeConsumer) handleToken(ctx context.Context, deviceKey string) error {
	var head model.Command
	err := c.db.WithContext(ctx).
		Where("device_key = ? AND status IN ?", deviceKey, model.NonTerminalCommandStatuses()).
		Order("created_at ASC").
		Order("command_id ASC").
		First(&head).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if head.Status != model.CommandStatusWaitingDevice {
		return nil
	}

	result := c.trigger(ctx, head.DeviceID, c.cfg.ConnectionRequest)
	event := &model.CommandEvent{
		CommandID:  head.CommandID,
		FromStatus: head.Status,
		ToStatus:   head.Status,
		Stage:      "connection_request",
		CreatedAt:  time.Now(),
	}
	if result.Err != nil {
		event.EventType = "WAKE_FAILED"
		event.Message = result.Err.Error()
		if err := c.store.AppendEvent(ctx, event); err != nil {
			return errors.Join(result.Err, err)
		}
		return result.Err
	}
	event.EventType = "WAKE_TRIGGERED"
	return c.store.AppendEvent(ctx, event)
}

func waitCommandWakeConsumer(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
