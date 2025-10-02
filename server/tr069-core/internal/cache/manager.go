// Package cache implements the TR069 cache mechanism for storing frequently accessed parameters.
// 包 cache 实现了 TR069 缓存机制，用于存储频繁访问的参数。
package cache

import (
	"context"
	"errors"
	"sync"

	"github.com/flipped-aurora/gin-vue-admin/server/plugin/tr069-core/interfaces"
)

// cacheManager implements the CacheManager interface.
// cacheManager 实现了 CacheManager 接口。
type cacheManager struct {
	caches         map[string]interfaces.Cache
	mutex          sync.RWMutex
	preloadEnabled bool
	preloadData    map[string]map[string]interface{}
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewCacheManager creates a new cache manager.
// NewCacheManager 创建一个新的缓存管理器。
func NewCacheManager() interfaces.CacheManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &cacheManager{
		caches:         make(map[string]interfaces.Cache),
		preloadEnabled: false,
		preloadData:    make(map[string]map[string]interface{}),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// CreateCache creates a new cache with the specified name and options.
// CreateCache 创建一个具有指定名称和选项的新缓存。
func (cm *cacheManager) CreateCache(name string, options ...interfaces.CacheOption) (interfaces.Cache, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Check if cache already exists
	// 检查缓存是否已存在
	if _, exists := cm.caches[name]; exists {
		return nil, errors.New("cache already exists")
	}

	// Create new cache
	// 创建新缓存
	cache := NewCache(name, options...)
	cm.caches[name] = cache

	// Preload data if enabled
	// 如果启用，则预加载数据
	if cm.preloadEnabled {
		if preloadData, exists := cm.preloadData[name]; exists {
			for key, value := range preloadData {
				cache.Set(key, value, 0)
			}
		}
	}

	return cache, nil
}

// GetCache returns the cache with the specified name.
// GetCache 返回具有指定名称的缓存。
func (cm *cacheManager) GetCache(name string) (interfaces.Cache, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	cache, exists := cm.caches[name]
	if !exists {
		return nil, errors.New("cache not found")
	}

	return cache, nil
}

// DeleteCache deletes the cache with the specified name.
// DeleteCache 删除具有指定名称的缓存。
func (cm *cacheManager) DeleteCache(name string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cache, exists := cm.caches[name]
	if !exists {
		return errors.New("cache not found")
	}

	// Close the cache to release resources
	// 关闭缓存以释放资源
	if err := cache.Close(); err != nil {
		return err
	}

	// Remove from caches map
	// 从缓存映射中删除
	delete(cm.caches, name)

	return nil
}

// CacheExists checks if a cache exists.
// CacheExists 检查缓存是否存在。
func (cm *cacheManager) CacheExists(name string) bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	_, exists := cm.caches[name]
	return exists
}

// GetCacheNames returns the names of all caches.
// GetCacheNames 返回所有缓存的名称。
func (cm *cacheManager) GetCacheNames() []string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	names := make([]string, 0, len(cm.caches))
	for name := range cm.caches {
		names = append(names, name)
	}

	return names
}

// GetAllCacheStats returns statistics for all caches.
// GetAllCacheStats 返回所有缓存的统计信息。
func (cm *cacheManager) GetAllCacheStats() map[string]interfaces.CacheStats {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	stats := make(map[string]interfaces.CacheStats)
	for name, cache := range cm.caches {
		stats[name] = cache.GetStats()
	}

	return stats
}

// Close releases all resources used by the cache manager.
// Close 释放缓存管理器使用的所有资源。
func (cm *cacheManager) Close() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Cancel context to stop background tasks
	// 取消上下文以停止后台任务
	cm.cancel()

	// Close all caches
	// 关闭所有缓存
	for _, cache := range cm.caches {
		if err := cache.Close(); err != nil {
			return err
		}
	}

	// Clear maps
	// 清除映射
	cm.caches = make(map[string]interfaces.Cache)
	cm.preloadData = make(map[string]map[string]interface{})

	return nil
}

// EnablePreloading enables or disables cache preloading.
// EnablePreloading 启用或禁用缓存预加载。
func (cm *cacheManager) EnablePreloading(enabled bool) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.preloadEnabled = enabled
}

// IsPreloadingEnabled returns whether cache preloading is enabled.
// IsPreloadingEnabled 返回缓存预加载是否已启用。
func (cm *cacheManager) IsPreloadingEnabled() bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return cm.preloadEnabled
}

// PreloadCache preloads data into a cache.
// PreloadCache 将数据预加载到缓存中。
func (cm *cacheManager) PreloadCache(name string, data map[string]interface{}) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Store preload data
	// 存储预加载数据
	cm.preloadData[name] = data

	// If cache exists and preloading is enabled, load data into cache
	// 如果缓存存在且预加载已启用，则将数据加载到缓存中
	if cm.preloadEnabled {
		if cache, exists := cm.caches[name]; exists {
			for key, value := range data {
				cache.Set(key, value, 0)
			}
		}
	}

	return nil
}
