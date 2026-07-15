package adapter

import (
	"context"
	"reflect"
	"testing"

	"github.com/redis/go-redis/v9"
)

type wakeTokenPush struct {
	key    string
	values []interface{}
}

type wakeTokenClient struct {
	pushes []wakeTokenPush
}

func (c *wakeTokenClient) RPush(_ context.Context, key string, values ...interface{}) *redis.IntCmd {
	c.pushes = append(c.pushes, wakeTokenPush{key: key, values: append([]interface{}(nil), values...)})
	return redis.NewIntResult(int64(len(c.pushes)), nil)
}

func TestEnqueueImmediateStoresOnlyDuplicateSafeDeviceWakeTokens(t *testing.T) {
	client := new(wakeTokenClient)
	for range 2 {
		if err := enqueueImmediate(context.Background(), client, "001122-WAKE"); err != nil {
			t.Fatalf("enqueueImmediate() error: %v", err)
		}
	}
	want := []wakeTokenPush{
		{key: "tr069:command:wakeup", values: []interface{}{"001122-WAKE"}},
		{key: "tr069:command:wakeup", values: []interface{}{"001122-WAKE"}},
	}
	if !reflect.DeepEqual(client.pushes, want) {
		t.Fatalf("wake pushes = %#v, want %#v", client.pushes, want)
	}
}
