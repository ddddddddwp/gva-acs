package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ddddddddwp/gva-acs/server/plugin/tr069/service"
	"github.com/redis/go-redis/v9"
)

const uploadIdentityKeyPrefix = "tr069:file-ingress"

type UploadIdentityBinding = service.UploadDeviceIdentity

type UploadIdentityStore struct {
	client redis.UniversalClient
	now    func() time.Time
}

func NewUploadIdentityStore(client redis.UniversalClient) *UploadIdentityStore {
	return &UploadIdentityStore{client: client, now: time.Now}
}

func (s *UploadIdentityStore) Bind(ctx context.Context, binding UploadIdentityBinding, ttl time.Duration) error {
	if s == nil || s.client == nil {
		return errors.New("upload identity Redis client is required")
	}
	if binding.DeviceID == 0 {
		return errors.New("upload identity device ID is required")
	}
	ip, err := normalizeUploadIdentityIP(binding.IP)
	if err != nil {
		return err
	}
	if ttl <= 0 {
		return errors.New("upload identity TTL must be positive")
	}
	now := s.now().UTC()
	binding.IP = ip
	binding.BoundAt = now
	binding.ExpiresAt = now.Add(ttl)
	payload, err := json.Marshal(binding)
	if err != nil {
		return err
	}
	member := strconv.FormatUint(uint64(binding.DeviceID), 10)
	zsetKey := uploadIdentityIPKey(ip)
	pipe := s.client.TxPipeline()
	pipe.ZAdd(ctx, zsetKey, redis.Z{Score: float64(binding.ExpiresAt.UnixMilli()), Member: member})
	pipe.Set(ctx, uploadIdentityMemberKey(ip, binding.DeviceID), payload, ttl)
	pipe.Expire(ctx, zsetKey, ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *UploadIdentityStore) Resolve(ctx context.Context, ipValue string) ([]UploadIdentityBinding, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("upload identity Redis client is required")
	}
	ip, err := normalizeUploadIdentityIP(ipValue)
	if err != nil {
		return nil, err
	}
	nowMillis := s.now().UTC().UnixMilli()
	zsetKey := uploadIdentityIPKey(ip)
	if err := s.client.ZRemRangeByScore(ctx, zsetKey, "-inf", strconv.FormatInt(nowMillis, 10)).Err(); err != nil {
		return nil, err
	}
	members, err := s.client.ZRangeByScore(ctx, zsetKey, &redis.ZRangeBy{Min: "(" + strconv.FormatInt(nowMillis, 10), Max: "+inf"}).Result()
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(members))
	deviceIDs := make([]uint, 0, len(members))
	for _, member := range members {
		parsed, parseErr := strconv.ParseUint(member, 10, 64)
		if parseErr != nil || parsed == 0 {
			continue
		}
		deviceID := uint(parsed)
		deviceIDs = append(deviceIDs, deviceID)
		keys = append(keys, uploadIdentityMemberKey(ip, deviceID))
	}
	if len(keys) == 0 {
		return nil, nil
	}
	values, err := s.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	bindings := make([]UploadIdentityBinding, 0, len(values))
	for index, raw := range values {
		text, ok := raw.(string)
		if !ok || strings.TrimSpace(text) == "" {
			continue
		}
		var binding UploadIdentityBinding
		if err := json.Unmarshal([]byte(text), &binding); err != nil || binding.DeviceID != deviceIDs[index] || !binding.ExpiresAt.After(s.now()) {
			continue
		}
		bindings = append(bindings, binding)
	}
	sort.Slice(bindings, func(left, right int) bool { return bindings[left].DeviceID < bindings[right].DeviceID })
	return bindings, nil
}

func normalizeUploadIdentityIP(value string) (string, error) {
	address, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("invalid upload identity IP: %w", err)
	}
	return address.Unmap().String(), nil
}

func uploadIdentityIPKey(ip string) string {
	return uploadIdentityKeyPrefix + ":ip:" + ip
}

func uploadIdentityMemberKey(ip string, deviceID uint) string {
	return uploadIdentityKeyPrefix + ":identity:" + ip + ":" + strconv.FormatUint(uint64(deviceID), 10)
}
