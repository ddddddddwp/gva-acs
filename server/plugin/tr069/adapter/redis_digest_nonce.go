package adapter

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisDigestNonceStore struct {
	client redis.UniversalClient
}

func NewRedisDigestNonceStore(client redis.UniversalClient) *RedisDigestNonceStore {
	return &RedisDigestNonceStore{client: client}
}

func (s *RedisDigestNonceStore) Issue(ctx context.Context, ttl time.Duration) (string, error) {
	if s == nil || s.client == nil {
		return "", errors.New("digest nonce Redis client is required")
	}
	if ttl <= 0 {
		return "", errors.New("digest nonce TTL must be positive")
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	nonce := hex.EncodeToString(random)
	if err := s.client.Set(ctx, digestNonceKey(nonce), "1", ttl).Err(); err != nil {
		return "", err
	}
	return nonce, nil
}

func (s *RedisDigestNonceStore) Consume(ctx context.Context, nonce, username, cnonce string, count uint32) (bool, error) {
	if s == nil || s.client == nil {
		return false, errors.New("digest nonce Redis client is required")
	}
	if nonce == "" || username == "" || cnonce == "" || count == 0 {
		return false, nil
	}
	result, err := consumeDigestNonceScript.Run(ctx, s.client,
		[]string{digestNonceKey(nonce), digestNonceCountKey(nonce, username, cnonce)},
		strconv.FormatUint(uint64(count), 10),
	).Int64()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}

var consumeDigestNonceScript = redis.NewScript(`
if redis.call("EXISTS", KEYS[1]) == 0 then
  return 0
end
local current = redis.call("GET", KEYS[2])
if current and tonumber(ARGV[1]) <= tonumber(current) then
  return -1
end
local ttl = redis.call("PTTL", KEYS[1])
if ttl <= 0 then
  return 0
end
redis.call("SET", KEYS[2], ARGV[1], "PX", ttl)
return 1
`)

func digestNonceKey(nonce string) string {
	return uploadIdentityKeyPrefix + ":digest:nonce:" + nonce
}

func digestNonceCountKey(nonce, username, cnonce string) string {
	sum := sha256.Sum256([]byte(nonce + "\x00" + username + "\x00" + cnonce))
	return uploadIdentityKeyPrefix + ":digest:nc:" + hex.EncodeToString(sum[:])
}
