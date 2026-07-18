package adapter

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestUploadIdentityStorePreservesMultipleCandidatesForOneIP(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewUploadIdentityStore(client)
	now := time.Date(2026, 7, 19, 5, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }

	for _, binding := range []UploadIdentityBinding{
		{DeviceID: 11, IP: "192.0.2.10", OUI: "001122", SerialNumber: "BS-11"},
		{DeviceID: 12, IP: "192.0.2.10", OUI: "334455", SerialNumber: "BS-12"},
	} {
		if err := store.Bind(context.Background(), binding, 30*time.Minute); err != nil {
			t.Fatalf("bind device %d: %v", binding.DeviceID, err)
		}
	}
	candidates, err := store.Resolve(context.Background(), "192.0.2.10")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if len(candidates) != 2 || candidates[0].DeviceID != 11 || candidates[1].DeviceID != 12 {
		t.Fatalf("candidates=%#v", candidates)
	}
}

func TestUploadIdentityStoreRemovesExpiredBindings(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store := NewUploadIdentityStore(client)
	now := time.Date(2026, 7, 19, 5, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return now }
	if err := store.Bind(context.Background(), UploadIdentityBinding{DeviceID: 21, IP: "2001:db8::21", SerialNumber: "BS-21"}, time.Minute); err != nil {
		t.Fatalf("bind: %v", err)
	}
	server.FastForward(2 * time.Minute)
	store.now = func() time.Time { return now.Add(2 * time.Minute) }
	candidates, err := store.Resolve(context.Background(), "2001:db8::21")
	if err != nil {
		t.Fatalf("resolve expired: %v", err)
	}
	if len(candidates) != 0 {
		t.Fatalf("expired candidates=%#v", candidates)
	}
}
