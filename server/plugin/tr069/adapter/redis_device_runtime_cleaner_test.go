package adapter

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/redis/go-redis/v9"
)

func TestRedisDeviceRuntimeCleanerPurgesOnlyTargetDevice(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()
	target := service.DeviceRuntimeIdentity{DeviceID: 41, DeviceKey: "001122-BS-41", IP: "192.0.2.41"}
	peer := service.DeviceRuntimeIdentity{DeviceID: 42, DeviceKey: "334455-BS-42", IP: target.IP}

	if err := client.RPush(ctx, RedisCommandWakeupKey, target.DeviceKey, peer.DeviceKey, target.DeviceKey).Err(); err != nil {
		t.Fatalf("seed wake list: %v", err)
	}
	for _, key := range []string{
		RedisPendingListPrefix + target.DeviceKey,
		RedisDeviceLockPrefix + target.DeviceKey,
		RedisInflightHashPrefix + target.DeviceKey,
		redisSessionKey(target.DeviceKey),
		redisLockKey(target.DeviceKey),
		RedisPendingListPrefix + peer.DeviceKey,
		RedisDeviceLockPrefix + peer.DeviceKey,
		RedisInflightHashPrefix + peer.DeviceKey,
		redisSessionKey(peer.DeviceKey),
		redisLockKey(peer.DeviceKey),
	} {
		if err := client.Set(ctx, key, "state", time.Hour).Err(); err != nil {
			t.Fatalf("seed %s: %v", key, err)
		}
	}
	if err := client.Set(ctx, redisTempKey(target.IP), target.DeviceKey, time.Hour).Err(); err != nil {
		t.Fatalf("seed temp session: %v", err)
	}
	identities := NewUploadIdentityStore(client)
	for _, identity := range []service.UploadDeviceIdentity{
		{DeviceID: target.DeviceID, IP: target.IP, OUI: "001122", SerialNumber: "BS-41"},
		{DeviceID: peer.DeviceID, IP: peer.IP, OUI: "334455", SerialNumber: "BS-42"},
	} {
		if err := identities.Bind(ctx, identity, time.Hour); err != nil {
			t.Fatalf("bind identity: %v", err)
		}
	}

	cleaner := NewRedisDeviceRuntimeCleaner(client)
	if err := cleaner.Purge(ctx, target); err != nil {
		t.Fatalf("Purge() error = %v", err)
	}

	wakeValues, err := client.LRange(ctx, RedisCommandWakeupKey, 0, -1).Result()
	if err != nil || len(wakeValues) != 1 || wakeValues[0] != peer.DeviceKey {
		t.Fatalf("wake values = %#v, error = %v", wakeValues, err)
	}
	for _, key := range []string{
		RedisPendingListPrefix + target.DeviceKey,
		RedisDeviceLockPrefix + target.DeviceKey,
		RedisInflightHashPrefix + target.DeviceKey,
		redisSessionKey(target.DeviceKey),
		redisLockKey(target.DeviceKey),
		redisTempKey(target.IP),
	} {
		if exists := client.Exists(ctx, key).Val(); exists != 0 {
			t.Fatalf("target key %s still exists", key)
		}
	}
	for _, key := range []string{
		RedisPendingListPrefix + peer.DeviceKey,
		RedisDeviceLockPrefix + peer.DeviceKey,
		RedisInflightHashPrefix + peer.DeviceKey,
		redisSessionKey(peer.DeviceKey),
		redisLockKey(peer.DeviceKey),
	} {
		if exists := client.Exists(ctx, key).Val(); exists != 1 {
			t.Fatalf("peer key %s was removed", key)
		}
	}
	candidates, err := identities.Resolve(ctx, target.IP)
	if err != nil || len(candidates) != 1 || candidates[0].DeviceID != peer.DeviceID {
		t.Fatalf("remaining upload identities = %#v, error = %v", candidates, err)
	}
}
