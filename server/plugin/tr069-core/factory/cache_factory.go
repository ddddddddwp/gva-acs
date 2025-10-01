// Package factory provides factory functions for creating TR069 components.
// 包 factory 提供创建 TR069 组件的工厂函数。
package factory

import (
	"github.com/root/demo/tr069/internal/cache"
	"github.com/root/demo/tr069/interfaces"
)

// CreateCacheManager creates a new cache manager.
// CreateCacheManager 创建新的缓存管理器。
func CreateCacheManager() interfaces.CacheManager {
	return cache.NewCacheManager()
}

// CreateCache creates a new cache with the specified name and eviction policy.
// CreateCache 创建具有指定名称和驱逐策略的新缓存。
func CreateCache(name string, policy interfaces.EvictionPolicy) (interfaces.Cache, error) {
	cm := cache.NewCacheManager()
	return cm.CreateCache(name, interfaces.WithEvictionPolicy(policy))
}

// CreateCacheWithMaxSize creates a new cache with the specified name, eviction policy, and maximum size.
// CreateCacheWithMaxSize 创建具有指定名称、驱逐策略和最大容量的新缓存。
func CreateCacheWithMaxSize(name string, policy interfaces.EvictionPolicy, maxSize int) (interfaces.Cache, error) {
	cm := cache.NewCacheManager()
	return cm.CreateCache(name, 
		interfaces.WithEvictionPolicy(policy),
		interfaces.WithCapacity(maxSize),
	)
}

// PreloadCache preloads a cache with the specified data.
// PreloadCache 使用指定的数据预加载缓存。
func PreloadCache(cm interfaces.CacheManager, name string, data map[string]interface{}) error {
	cm.EnablePreloading(true)
	if !cm.CacheExists(name) {
		_, err := cm.CreateCache(name)
		if err != nil {
			return err
		}
	}
	return cm.PreloadCache(name, data)
}