// Package cache implements the TR069 cache mechanism for storing frequently accessed parameters.
// 包 cache 实现了 TR069 缓存机制，用于存储频繁访问的参数。
package cache

import (
	"testing"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// TestCache_SetAndGet tests the Set and Get methods of the cache.
// TestCache_SetAndGet 测试缓存的 Set 和 Get 方法。
func TestCache_SetAndGet(t *testing.T) {
	// Create a cache
	// 创建缓存
	cache := NewCache("test-cache", interfaces.EvictionPolicyLRU)
	
	// Set a value
	// 设置值
	key := "test-key"
	value := "test-value"
	err := cache.Set(key, value, 0)
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}
	
	// Get the value
	// 获取值
	result, err := cache.Get(key)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	
	// Check the value
	// 检查值
	if result != value {
		t.Errorf("Expected %v, got %v", value, result)
	}
}

// TestCache_Delete tests the Delete method of the cache.
// TestCache_Delete 测试缓存的 Delete 方法。
func TestCache_Delete(t *testing.T) {
	// Create a cache
	// 创建缓存
	cache := NewCache("test-cache", interfaces.EvictionPolicyLRU)
	
	// Set a value
	// 设置值
	key := "test-key"
	value := "test-value"
	cache.Set(key, value, 0)
	
	// Delete the value
	// 删除值
	err := cache.Delete(key)
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}
	
	// Try to get the deleted value
	// 尝试获取已删除的值
	result, err := cache.Get(key)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	
	// Check that the value is nil
	// 检查值是否为 nil
	if result != nil {
		t.Errorf("Expected nil, got %v", result)
	}
}

// TestCache_Clear tests the Clear method of the cache.
// TestCache_Clear 测试缓存的 Clear 方法。
func TestCache_Clear(t *testing.T) {
	// Create a cache
	// 创建缓存
	cache := NewCache("test-cache", interfaces.EvictionPolicyLRU)
	
	// Set some values
	// 设置一些值
	cache.Set("key1", "value1", 0)
	cache.Set("key2", "value2", 0)
	cache.Set("key3", "value3", 0)
	
	// Check that the cache is not empty
	// 检查缓存不为空
	if cache.IsEmpty() {
		t.Error("Expected cache to not be empty")
	}
	
	// Clear the cache
	// 清空缓存
	err := cache.Clear()
	if err != nil {
		t.Errorf("Clear failed: %v", err)
	}
	
	// Check that the cache is empty
	// 检查缓存为空
	if !cache.IsEmpty() {
		t.Error("Expected cache to be empty")
	}
}

// TestCache_Size tests the Size method of the cache.
// TestCache_Size 测试缓存的 Size 方法。
func TestCache_Size(t *testing.T) {
	// Create a cache
	// 创建缓存
	cache := NewCache("test-cache", interfaces.EvictionPolicyLRU)
	
	// Check initial size
	// 检查初始大小
	if cache.Size() != 0 {
		t.Errorf("Expected size 0, got %d", cache.Size())
	}
	
	// Set some values
	// 设置一些值
	cache.Set("key1", "value1", 0)
	cache.Set("key2", "value2", 0)
	
	// Check size after adding values
	// 添加值后检查大小
	if cache.Size() != 2 {
		t.Errorf("Expected size 2, got %d", cache.Size())
	}
}

// TestCache_Stats tests the Stats method of the cache.
// TestCache_Stats 测试缓存的 Stats 方法。
func TestCache_Stats(t *testing.T) {
	// Create a cache
	// 创建缓存
	cache := NewCache("test-cache", interfaces.EvictionPolicyLRU)
	
	// Get initial stats
	// 获取初始统计信息
	stats := cache.Stats()
	
	// Check initial stats
	// 检查初始统计信息
	if stats.Hits != 0 || stats.Misses != 0 || stats.Evictions != 0 {
		t.Errorf("Expected all stats to be 0, got %+v", stats)
	}
	
	// Try to get a non-existent key (should be a miss)
	// 尝试获取不存在的键（应该是一个未命中）
	cache.Get("non-existent")
	
	// Get stats after a miss
	// 未命中后获取统计信息
	stats = cache.Stats()
	if stats.Misses != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.Misses)
	}
	
	// Set a value
	// 设置值
	cache.Set("key", "value", 0)
	
	// Get the value (should be a hit)
	// 获取值（应该是一个命中）
	cache.Get("key")
	
	// Get stats after a hit
	// 命中后获取统计信息
	stats = cache.Stats()
	if stats.Hits != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.Hits)
	}
}

// TestCache_TTL tests the TTL (time-to-live) functionality of the cache.
// TestCache_TTL 测试缓存的 TTL（生存时间）功能。
func TestCache_TTL(t *testing.T) {
	// Create a cache
	// 创建缓存
	cache := NewCache("test-cache", interfaces.EvictionPolicyLRU)
	
	// Set a value with a short TTL
	// 设置一个具有短 TTL 的值
	key := "test-key"
	value := "test-value"
	ttl := 100 * time.Millisecond
	cache.Set(key, value, ttl)
	
	// Get the value immediately (should succeed)
	// 立即获取值（应该成功）
	result, err := cache.Get(key)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if result != value {
		t.Errorf("Expected %v, got %v", value, result)
	}
	
	// Wait for the TTL to expire
	// 等待 TTL 过期
	time.Sleep(ttl)
	
	// Try to get the expired value (should return nil)
	// 尝试获取过期的值（应该返回 nil）
	result, err = cache.Get(key)
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if result != nil {
		t.Errorf("Expected nil for expired value, got %v", result)
	}
}

// TestCache_EvictionPolicy tests the eviction policy functionality of the cache.
// TestCache_EvictionPolicy 测试缓存的驱逐策略功能。
func TestCache_EvictionPolicy(t *testing.T) {
	// Create a cache with a small max size
	// 创建一个最大大小较小的缓存
	cache := NewCache("test-cache", interfaces.EvictionPolicyLRU)
	cache.SetMaxSize(2)
	
	// Set values to exceed the max size
	// 设置值以超过最大大小
	cache.Set("key1", "value1", 0)
	cache.Set("key2", "value2", 0)
	cache.Set("key3", "value3", 0) // This should trigger eviction
	// 这应该触发驱逐
	
	// Check that one of the original values was evicted
	// 检查其中一个原始值已被驱逐
	stats := cache.Stats()
	if stats.Evictions == 0 {
		t.Error("Expected an eviction to occur")
	}
}