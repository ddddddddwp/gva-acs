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

type RedisCommandIngest struct {
	client    redis.UniversalClient
	streamKey string
}

func NewRedisCommandIngest(streamKey string) *RedisCommandIngest {
	if streamKey == "" {
		streamKey = RedisIngestStreamKey
	}
	return &RedisCommandIngest{
		client:    global.GVA_REDIS,
		streamKey: streamKey,
	}
}

func (i *RedisCommandIngest) Enqueue(ctx context.Context, cmd *core.Command) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if i == nil || i.client == nil {
		return errors.New("redis not initialized")
	}
	if cmd == nil || cmd.DeviceKey == "" || cmd.ID == "" || cmd.Operation == "" {
		return errors.New("invalid command")
	}

	params := ""
	if cmd.Params != nil {
		b, err := json.Marshal(cmd.Params)
		if err != nil {
			return err
		}
		params = string(b)
	}
	createdAt := cmd.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	return i.client.XAdd(ctx, &redis.XAddArgs{
		Stream: i.streamKey,
		Values: map[string]interface{}{
			"deviceKey": cmd.DeviceKey,
			"commandId": cmd.ID,
			"dedupKey":  cmd.DedupKey,
			"op":        cmd.Operation,
			"params":    params,
			"createdAt": createdAt.UnixMilli(),
		},
	}).Err()
}

var _ core.CommandIngest = (*RedisCommandIngest)(nil)
