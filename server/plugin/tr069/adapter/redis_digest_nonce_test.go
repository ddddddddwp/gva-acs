package adapter

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestDigestNonceStoreExpiresAndRejectsRepeatedCounts(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewRedisDigestNonceStore(client)
	nonce, err := store.Issue(context.Background(), 2*time.Minute)
	if err != nil || len(nonce) != 64 {
		t.Fatalf("nonce=%q err=%v", nonce, err)
	}
	accepted, err := store.Consume(context.Background(), nonce, "log-user", "client-nonce", 1)
	if err != nil || !accepted {
		t.Fatalf("first consume accepted=%v err=%v", accepted, err)
	}
	accepted, err = store.Consume(context.Background(), nonce, "log-user", "client-nonce", 1)
	if err != nil || accepted {
		t.Fatalf("repeated consume accepted=%v err=%v", accepted, err)
	}
	accepted, err = store.Consume(context.Background(), nonce, "log-user", "client-nonce", 2)
	if err != nil || !accepted {
		t.Fatalf("increased count accepted=%v err=%v", accepted, err)
	}

	server.FastForward(3 * time.Minute)
	accepted, err = store.Consume(context.Background(), nonce, "log-user", "new-client-nonce", 1)
	if err != nil || accepted {
		t.Fatalf("expired nonce accepted=%v err=%v", accepted, err)
	}
}
