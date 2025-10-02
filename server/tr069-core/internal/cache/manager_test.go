// Package cache implements the TR069 cache mechanism for storing frequently accessed parameters.
// 包 cache 实现了 TR069 缓存机制，用于存储频繁访问的参数。
package cache

import (
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// TestCacheManager_GetCache tests the GetCache method of the cache manager.
// TestCacheManager_GetCache 测试缓存管理器的 GetCache 方法。
func TestCacheManager_GetCache(t *testing.T) {
	// Create a cache manager
	// 创建缓存管理器
	manager := NewCacheManager(interfaces.EvictionPolicyLRU)

	// Get a cache
	// 获取缓存
	cacheName := "test-cache"
	cache := manager.GetCache(cacheName)

	// Check that the cache is not nil
	// 检查缓存不为 nil
	if cache == nil {
		t.Error("Expected cache to not be nil")
	}

	// Get the same cache again
	// 再次获取相同的缓存
	sameCache := manager.GetCache(cacheName)

	// Check that it's the same cache
	// 检查它是相同的缓存
	if cache != sameCache {
		t.Error("Expected to get the same cache instance")
	}
}

// TestCacheManager_CreateCache tests the CreateCache method of the cache manager.
// TestCacheManager_CreateCache 测试缓存管理器的 CreateCache 方法。
func TestCacheManager_CreateCache(t *testing.T) {
	// Create a cache manager
	// 创建缓存管理器
	manager := NewCacheManager(interfaces.EvictionPolicyLRU)

	// Create a cache with a specific policy
	// 使用特定策略创建缓存
	cacheName := "test-cache"
	cache := manager.CreateCache(cacheName, interfaces.EvictionPolicyLFU)

	// Check that the cache is not nil
	// 检查缓存不为 nil
	if cache == nil {
		t.Error("Expected cache to not be nil")
	}

	// Check that the cache has the correct policy
	// 检查缓存具有正确的策略
	if cache.GetEvictionPolicy() != interfaces.EvictionPolicyLFU {
		t.Errorf("Expected LFU policy, got %v", cache.GetEvictionPolicy())
	}
}

// TestCacheManager_DeleteCache tests the DeleteCache method of the cache manager.
// TestCacheManager_DeleteCache 测试缓存管理器的 DeleteCache 方法。
func TestCacheManager_DeleteCache(t *testing.T) {
	// Create a cache manager
	// 创建缓存管理器
	manager := NewCacheManager(interfaces.EvictionPolicyLRU)

	// Create a cache
	// 创建缓存
	cacheName := "test-cache"
	manager.CreateCache(cacheName, interfaces.EvictionPolicyLRU)

	// Check that the cache exists
	// 检查缓存存在
	caches := manager.ListCaches()
	if len(caches) != 1 {
		t.Errorf("Expected 1 cache, got %d", len(caches))
	}

	// Delete the cache
	// 删除缓存
	err := manager.DeleteCache(cacheName)
	if err != nil {
		t.Errorf("DeleteCache failed: %v", err)
	}

	// Check that the cache no longer exists
	// 检查缓存不再存在
	caches = manager.ListCaches()
	if len(caches) != 0 {
		t.Errorf("Expected 0 caches, got %d", len(caches))
	}
}

// TestCacheManager_ListCaches tests the ListCaches method of the cache manager.
// TestCacheManager_ListCaches 测试缓存管理器的 ListCaches 方法。
func TestCacheManager_ListCaches(t *testing.T) {
	// Create a cache manager
	// 创建缓存管理器
	manager := NewCacheManager(interfaces.EvictionPolicyLRU)

	// Check that the list is initially empty
	// 检查列表最初为空
	caches := manager.ListCaches()
	if len(caches) != 0 {
		t.Errorf("Expected empty list, got %v", caches)
	}

	// Create some caches
	// 创建一些缓存
	manager.CreateCache("cache1", interfaces.EvictionPolicyLRU)
	manager.CreateCache("cache2", interfaces.EvictionPolicyLFU)
	manager.CreateCache("cache3", interfaces.EvictionPolicyFIFO)

	// Check that all caches are listed
	// 检查所有缓存都已列出
	caches = manager.ListCaches()
	if len(caches) != 3 {
		t.Errorf("Expected 3 caches, got %d", len(caches))
	}

	// Check that all cache names are present
	// 检查所有缓存名称都存在
	expectedCaches := map[string]bool{
		"cache1": true,
		"cache2": true,
		"cache3": true,
	}

	for _, cacheName := range caches {
		if !expectedCaches[cacheName] {
			t.Errorf("Unexpected cache name: %s", cacheName)
		}
		delete(expectedCaches, cacheName)
	}

	// Check that all expected caches were found
	// 检查所有预期的缓存都已找到
	if len(expectedCaches) != 0 {
		t.Errorf("Missing cache names: %v", expectedCaches)
	}
}

// TestCacheManager_GetCacheStats tests the GetCacheStats method of the cache manager.
// TestCacheManager_GetCacheStats 测试缓存管理器的 GetCacheStats 方法。
func TestCacheManager_GetCacheStats(t *testing.T) {
	// Create a cache manager
	// 创建缓存管理器
	manager := NewCacheManager(interfaces.EvictionPolicyLRU)

	// Try to get stats for a non-existent cache
	// 尝试获取不存在缓存的统计信息
	stats, err := manager.GetCacheStats("non-existent")
	if err != nil {
		t.Errorf("GetCacheStats failed: %v", err)
	}
	if stats != nil {
		t.Error("Expected nil stats for non-existent cache")
	}

	// Create a cache and get its stats
	// 创建缓存并获取其统计信息
	cacheName := "test-cache"
	manager.CreateCache(cacheName, interfaces.EvictionPolicyLRU)
	stats, err = manager.GetCacheStats(cacheName)
	if err != nil {
		t.Errorf("GetCacheStats failed: %v", err)
	}
	if stats == nil {
		t.Error("Expected stats for existing cache")
	}
}

// TestCacheManager_GetAllCacheStats tests the GetAllCacheStats method of the cache manager.
// TestCacheManager_GetAllCacheStats 测试缓存管理器的 GetAllCacheStats 方法。
func TestCacheManager_GetAllCacheStats(t *testing.T) {
	// Create a cache manager
	// 创建缓存管理器
	manager := NewCacheManager(interfaces.EvictionPolicyLRU)

	// Get stats when there are no caches
	// 在没有缓存时获取统计信息
	stats := manager.GetAllCacheStats()
	if len(stats) != 0 {
		t.Errorf("Expected empty stats map, got %v", stats)
	}

	// Create some caches
	// 创建一些缓存
	manager.CreateCache("cache1", interfaces.EvictionPolicyLRU)
	manager.CreateCache("cache2", interfaces.EvictionPolicyLFU)

	// Get all stats
	// 获取所有统计信息
	stats = manager.GetAllCacheStats()
	if len(stats) != 2 {
		t.Errorf("Expected stats for 2 caches, got %d", len(stats))
	}

	// Check that stats exist for both caches
	// 检查两个缓存的统计信息都存在
	if stats["cache1"] == nil {
		t.Error("Expected stats for cache1")
	}
	if stats["cache2"] == nil {
		t.Error("Expected stats for cache2")
	}
}

// TestCacheManager_SetEvictionPolicy tests the SetEvictionPolicy method of the cache manager.
// TestCacheManager_SetEvictionPolicy 测试缓存管理器的 SetEvictionPolicy 方法。
func TestCacheManager_SetEvictionPolicy(t *testing.T) {
	// Create a cache manager
	// 创建缓存管理器
	manager := NewCacheManager(interfaces.EvictionPolicyLRU)

	// Create a cache
	// 创建缓存
	cacheName := "test-cache"
	manager.CreateCache(cacheName, interfaces.EvictionPolicyLRU)

	// Check initial policy
	// 检查初始策略
	policy, err := manager.GetEvictionPolicy(cacheName)
	if err != nil {
		t.Errorf("GetEvictionPolicy failed: %v", err)
	}
	if policy != interfaces.EvictionPolicyLRU {
		t.Errorf("Expected LRU policy, got %v", policy)
	}

	// Set a new policy
	// 设置新策略
	err = manager.SetEvictionPolicy(cacheName, interfaces.EvictionPolicyLFU)
	if err != nil {
		t.Errorf("SetEvictionPolicy failed: %v", err)
	}

	// Check that the policy was updated
	// 检查策略已更新
	policy, err = manager.GetEvictionPolicy(cacheName)
	if err != nil {
		t.Errorf("GetEvictionPolicy failed: %v", err)
	}
	if policy != interfaces.EvictionPolicyLFU {
		t.Errorf("Expected LFU policy, got %v", policy)
	}
}

// TestCacheManager_PreloadCache tests the PreloadCache method of the cache manager.
// TestCacheManager_PreloadCache 测试缓存管理器的 PreloadCache 方法。
func TestCacheManager_PreloadCache(t *testing.T) {
	// Create a cache manager
	// 创建缓存管理器
	manager := NewCacheManager(interfaces.EvictionPolicyLRU)

	// Preload data into a cache
	// 预加载数据到缓存
	cacheName := "test-cache"
	data := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	err := manager.PreloadCache(cacheName, data)
	if err != nil {
		t.Errorf("PreloadCache failed: %v", err)
	}

	// Check that the data was loaded
	// 检查数据已加载
	cache := manager.GetCache(cacheName)
	for key, expectedValue := range data {
		value, err := cache.Get(key)
		if err != nil {
			t.Errorf("Get failed for key %s: %v", key, err)
		}
		if value != expectedValue {
			t.Errorf("Expected %v for key %s, got %v", expectedValue, key, value)
		}
	}
}

// TestCacheManager_RefreshCache tests the RefreshCache method of the cache manager.
// TestCacheManager_RefreshCache 测试缓存管理器的 RefreshCache 方法。
func TestCacheManager_RefreshCache(t *testing.T) {
	// Create a cache manager
	// 创建缓存管理器
	manager := NewCacheManager(interfaces.EvictionPolicyLRU)

	// Preload some initial data
	// 预加载一些初始数据
	cacheName := "test-cache"
	initialData := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
	}
	manager.PreloadCache(cacheName, initialData)

	// Check that the initial data exists
	// 检查初始数据存在
	cache := manager.GetCache(cacheName)
	value, err := cache.Get("key1")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	// Refresh the cache with new data
	// 使用新数据刷新缓存
	newData := map[string]interface{}{
		"key3": "value3",
		"key4": "value4",
	}
	err = manager.RefreshCache(cacheName, newData)
	if err != nil {
		t.Errorf("RefreshCache failed: %v", err)
	}

	// Check that the old data is gone
	// 检查旧数据已消失
	value, err = cache.Get("key1")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if value != nil {
		t.Errorf("Expected nil for key1, got %v", value)
	}

	// Check that the new data is present
	// 检查新数据存在
	value, err = cache.Get("key3")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if value != "value3" {
		t.Errorf("Expected value3, got %v", value)
	}
}

// TestCacheManager_WarmupCache tests the WarmupCache method of the cache manager.
// TestCacheManager_WarmupCache 测试缓存管理器的 WarmupCache 方法。
func TestCacheManager_WarmupCache(t *testing.T) {
	// Create a cache manager
	// 创建缓存管理器
	manager := NewCacheManager(interfaces.EvictionPolicyLRU)

	// Define a loader function
	// 定义加载器函数
	loader := func() (map[string]interface{}, error) {
		return map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}, nil
	}

	// Warm up the cache
	// 预热缓存
	cacheName := "test-cache"
	err := manager.WarmupCache(cacheName, loader)
	if err != nil {
		t.Errorf("WarmupCache failed: %v", err)
	}

	// Check that the data was loaded
	// 检查数据已加载
	cache := manager.GetCache(cacheName)
	value, err := cache.Get("key1")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}
	if value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}
}
