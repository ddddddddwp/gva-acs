package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/ddddddddwp/gva-acs/server/global"
	"github.com/ddddddddwp/tr069-core-only/pkg/core"
	"github.com/redis/go-redis/v9"
)

type RedisSessionStore struct {
	client     redis.UniversalClient
	defaultTTL time.Duration
	lockTTL    time.Duration
	instanceID string
	now        func() time.Time
}

func NewRedisSessionStore(defaultTTL time.Duration) *RedisSessionStore {
	if defaultTTL <= 0 {
		defaultTTL = 5 * time.Minute
	}
	hostname, _ := os.Hostname()
	return &RedisSessionStore{
		client:     global.GVA_REDIS,
		defaultTTL: defaultTTL,
		lockTTL:    30 * time.Second,
		instanceID: "gva-" + hostname + "-" + strconv.Itoa(os.Getpid()),
		now:        time.Now,
	}
}

func (s *RedisSessionStore) GetByTempKey(ctx context.Context, ipKey string) (*core.Session, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if s == nil || s.client == nil || ipKey == "" {
		return nil, nil
	}
	deviceKey, err := s.client.Get(ctx, redisTempKey(ipKey)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.GetByDeviceKey(ctx, deviceKey)
}

func (s *RedisSessionStore) GetByDeviceKey(ctx context.Context, deviceKey string) (*core.Session, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if s == nil || s.client == nil || deviceKey == "" {
		return nil, nil
	}
	b, err := s.client.Get(ctx, redisSessionKey(deviceKey)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var sess core.Session
	if err := json.Unmarshal(b, &sess); err != nil {
		_ = s.client.Del(ctx, redisSessionKey(deviceKey)).Err()
		return nil, nil
	}
	return &sess, nil
}

func (s *RedisSessionStore) Save(ctx context.Context, session *core.Session, ttl time.Duration) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if s == nil || s.client == nil {
		return errors.New("redis not initialized")
	}
	if session == nil {
		return errors.New("session is nil")
	}
	if session.DeviceKey == "" {
		return errors.New("session DeviceKey is empty")
	}
	if ttl <= 0 {
		ttl = s.defaultTTL
	}
	b, err := json.Marshal(session)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, redisSessionKey(session.DeviceKey), b, ttl).Err()
}

func (s *RedisSessionStore) Delete(ctx context.Context, key string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if s == nil || s.client == nil || key == "" {
		return nil
	}
	_, err := s.client.Del(ctx, redisSessionKey(key), redisTempKey(key), redisLockKey(key)).Result()
	return err
}

func (s *RedisSessionStore) Migrate(ctx context.Context, tempKey, deviceKey string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if s == nil || s.client == nil {
		return errors.New("redis not initialized")
	}
	if tempKey == "" || deviceKey == "" {
		return nil
	}
	ttl := s.defaultTTL
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return s.client.Set(ctx, redisTempKey(tempKey), deviceKey, ttl).Err()
}

var unlockSessionLockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

func (s *RedisSessionStore) Lock(ctx context.Context, deviceKey string) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if s == nil || s.client == nil || deviceKey == "" {
		return false, nil
	}
	return s.client.SetNX(ctx, redisLockKey(deviceKey), s.instanceID, s.lockTTL).Result()
}

func (s *RedisSessionStore) Unlock(ctx context.Context, deviceKey string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if s == nil || s.client == nil || deviceKey == "" {
		return nil
	}
	return unlockSessionLockScript.Run(ctx, s.client, []string{redisLockKey(deviceKey)}, s.instanceID).Err()
}

func redisSessionKey(deviceKey string) string { return "tr069:sess:" + deviceKey }
func redisTempKey(tempKey string) string      { return "tr069:sess:tmp:" + tempKey }
func redisLockKey(deviceKey string) string    { return "tr069:sess:lock:" + deviceKey }

var _ core.SessionStore = (*RedisSessionStore)(nil)
