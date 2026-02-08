package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/redis/go-redis/v9"
)

type RedisDispatcherConfig struct {
	IngestStream string
	Group        string
	Consumer     string
	Block        time.Duration
	BatchSize    int64
	ClaimIdle    time.Duration
}

type DispatcherPayload struct {
	StreamID  string `json:"streamId"`
	DeviceKey string `json:"deviceKey"`
	CommandID string `json:"commandId"`
	DedupKey  string `json:"dedupKey,omitempty"`
	Op        string `json:"op"`
	Params    string `json:"params,omitempty"`
	CreatedAt int64  `json:"createdAt,omitempty"`
}

func (p DispatcherPayload) pendingKey() string { return RedisPendingListPrefix + p.DeviceKey }

func defaultDispatcherConfig() RedisDispatcherConfig {
	hostname, _ := os.Hostname()
	consumer := "gva-" + hostname + "-" + strconv.Itoa(os.Getpid())
	return RedisDispatcherConfig{
		IngestStream: RedisIngestStreamKey,
		Group:        RedisDispatcherGroup,
		Consumer:     consumer,
		Block:        2 * time.Second,
		BatchSize:    128,
		ClaimIdle:    60 * time.Second,
	}
}

var (
	dispatcherOnce   sync.Once
	dispatcherCancel context.CancelFunc
)

func StartRedisDispatcher(ctx context.Context, cfg RedisDispatcherConfig) {
	dispatcherOnce.Do(func() {
		if global.GVA_REDIS == nil {
			return
		}
		if ctx == nil {
			ctx = context.Background()
		}
		if cfg.IngestStream == "" {
			cfg = defaultDispatcherConfig()
		}
		cctx, cancel := context.WithCancel(ctx)
		dispatcherCancel = cancel
		go runRedisDispatcher(cctx, cfg)
	})
}

func StopRedisDispatcher() {
	if dispatcherCancel != nil {
		dispatcherCancel()
	}
}

func runRedisDispatcher(ctx context.Context, cfg RedisDispatcherConfig) {
	client := global.GVA_REDIS
	if client == nil {
		return
	}

	_ = client.XGroupCreateMkStream(ctx, cfg.IngestStream, cfg.Group, "$").Err()

	lastClaim := time.Time{}

	for {
		if err := ctx.Err(); err != nil {
			return
		}

		if cfg.ClaimIdle > 0 && time.Since(lastClaim) > cfg.ClaimIdle {
			lastClaim = time.Now()
			claimAndDispatch(ctx, client, cfg)
		}

		streams, err := client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    cfg.Group,
			Consumer: cfg.Consumer,
			Streams:  []string{cfg.IngestStream, ">"},
			Count:    cfg.BatchSize,
			Block:    cfg.Block,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || ctx.Err() != nil {
				continue
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, s := range streams {
			for _, msg := range s.Messages {
				dispatchOne(ctx, client, cfg, msg)
			}
		}
	}
}

func claimAndDispatch(ctx context.Context, client redis.UniversalClient, cfg RedisDispatcherConfig) {
	if cfg.ClaimIdle <= 0 {
		return
	}

	pending, err := client.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: cfg.IngestStream,
		Group:  cfg.Group,
		Start:  "-",
		End:    "+",
		Count:  cfg.BatchSize,
		Idle:   cfg.ClaimIdle,
	}).Result()
	if err != nil || len(pending) == 0 {
		return
	}
	ids := make([]string, 0, len(pending))
	for _, p := range pending {
		ids = append(ids, p.ID)
	}
	claimed, err := client.XClaim(ctx, &redis.XClaimArgs{
		Stream:   cfg.IngestStream,
		Group:    cfg.Group,
		Consumer: cfg.Consumer,
		MinIdle:  cfg.ClaimIdle,
		Messages: ids,
	}).Result()
	if err != nil || len(claimed) == 0 {
		return
	}
	for _, msg := range claimed {
		dispatchOne(ctx, client, cfg, msg)
	}
}

func dispatchOne(ctx context.Context, client redis.UniversalClient, cfg RedisDispatcherConfig, msg redis.XMessage) {
	p := DispatcherPayload{
		StreamID: msg.ID,
	}
	if v, ok := msg.Values["deviceKey"]; ok {
		p.DeviceKey = toString(v)
	}
	if p.DeviceKey == "" {
		_ = client.XAck(ctx, cfg.IngestStream, cfg.Group, msg.ID).Err()
		return
	}
	p.CommandID = toString(msg.Values["commandId"])
	p.DedupKey = toString(msg.Values["dedupKey"])
	p.Op = toString(msg.Values["op"])
	p.Params = toString(msg.Values["params"])
	p.CreatedAt = toInt64(msg.Values["createdAt"])

	b, err := json.Marshal(p)
	if err != nil {
		return
	}

	if err := client.RPush(ctx, p.pendingKey(), b).Err(); err != nil {
		return
	}
	_ = client.XAck(ctx, cfg.IngestStream, cfg.Group, msg.ID).Err()
}

func toString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return ""
	}
}

func toInt64(v interface{}) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case string:
		n, _ := strconv.ParseInt(x, 10, 64)
		return n
	default:
		return 0
	}
}
