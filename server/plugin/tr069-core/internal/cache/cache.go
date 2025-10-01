// Package cache implements the TR069 cache mechanism for storing frequently accessed parameters.
// 包 cache 实现了 TR069 缓存机制，用于存储频繁访问的参数。
package cache

import (
	"sync"
	"time"

	"github.com/root/demo/tr069/interfaces"
)

// cacheItem represents an item in the cache.
// cacheItem 表示缓存中的一个项目。
type cacheItem struct {
	key        string
	value      interface{}
	expiration int64 // Unix timestamp in nanoseconds
	lastAccess int64 // Unix timestamp in nanoseconds
	accessCount int64 // Number of times this item has been accessed
}

// cache implements the Cache interface.
// cache 实现了 Cache 接口。
type cache struct {
	name            string
	items           map[string]*cacheItem
	mutex           sync.RWMutex
	stats           interfaces.CacheStats
	capacity        int
	evictionPolicy  interfaces.EvictionPolicy
	cleanupInterval time.Duration
	defaultTTL      time.Duration
	stopCleanup     chan struct{}
	eventListeners  []interfaces.CacheEventListener
	listenerMutex   sync.RWMutex
}

// NewCache creates a new cache with the specified name and options.
// NewCache 创建一个具有指定名称和选项的新缓存。
func NewCache(name string, options ...interfaces.CacheOption) interfaces.Cache {
	c := &cache{
		name:            name,
		items:           make(map[string]*cacheItem),
		capacity:        1000, // Default capacity
		evictionPolicy:  interfaces.EvictionPolicyLRU, // Default policy
		cleanupInterval: 5 * time.Minute, // Default cleanup interval
		defaultTTL:      0, // Default TTL (0 means no expiration)
		stopCleanup:     make(chan struct{}),
		eventListeners:  make([]interfaces.CacheEventListener, 0),
		stats: interfaces.CacheStats{
			MaxSize: 1000,
		},
	}

	// Apply options
	for _, option := range options {
		option(c)
	}

	// Start cleanup routine
	go c.startCleanupRoutine()

	return c
}

// startCleanupRoutine starts a goroutine that periodically cleans up expired items.
// startCleanupRoutine 启动一个定期清理过期项目的 goroutine。
func (c *cache) startCleanupRoutine() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupExpired()
		case <-c.stopCleanup:
			return
		}
	}
}

// cleanupExpired removes expired items from the cache.
// cleanupExpired 从缓存中删除过期的项目。
func (c *cache) cleanupExpired() {
	now := time.Now().UnixNano()
	
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	for key, item := range c.items {
		if item.expiration > 0 && item.expiration <= now {
			// Item has expired
			delete(c.items, key)
			c.stats.Evictions++
			
			// Emit event
			c.emitEvent(interfaces.CacheEventItemExpired, key, item.value)
		}
	}
}

// Get retrieves a value from the cache.
// Get 从缓存中检索值。
func (c *cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	item, found := c.items[key]
	c.mutex.RUnlock()
	
	if !found {
		c.mutex.Lock()
		c.stats.Misses++
		c.mutex.Unlock()
		return nil, false
	}
	
	// Check if the item has expired
	if item.expiration > 0 && item.expiration <= time.Now().UnixNano() {
		c.mutex.Lock()
		delete(c.items, key)
		c.stats.Misses++
		c.stats.Evictions++
		c.mutex.Unlock()
		
		// Emit event
		c.emitEvent(interfaces.CacheEventItemExpired, key, item.value)
		
		return nil, false
	}
	
	// Update access time and count
	c.mutex.Lock()
	item.lastAccess = time.Now().UnixNano()
	item.accessCount++
	c.stats.Hits++
	c.mutex.Unlock()
	
	return item.value, true
}

// Set stores a value in the cache.
// Set 在缓存中存储值。
func (c *cache) Set(key string, value interface{}, ttl time.Duration) error {
	var expiration int64
	if ttl > 0 {
		expiration = time.Now().Add(ttl).UnixNano()
	}
	
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	// Check if we need to evict an item
	if len(c.items) >= c.capacity && c.items[key] == nil {
		c.evict()
	}
	
	// Check if the item already exists
	if item, found := c.items[key]; found {
		// Update existing item
		item.value = value
		item.expiration = expiration
		item.lastAccess = time.Now().UnixNano()
		
		// Emit event
		c.emitEvent(interfaces.CacheEventItemUpdated, key, value)
	} else {
		// Add new item
		c.items[key] = &cacheItem{
			key:        key,
			value:      value,
			expiration: expiration,
			lastAccess: time.Now().UnixNano(),
			accessCount: 0,
		}
		
		// Update stats
		if c.stats.Size < c.capacity {
			c.stats.Size++
		}
		
		// Emit event
		c.emitEvent(interfaces.CacheEventItemAdded, key, value)
	}
	
	return nil
}

// Delete removes a value from the cache.
// Delete 从缓存中删除值。
func (c *cache) Delete(key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	if item, found := c.items[key]; found {
		delete(c.items, key)
		c.stats.Size--
		
		// Emit event
		c.emitEvent(interfaces.CacheEventItemRemoved, key, item.value)
	}
	
	return nil
}

// Exists checks if a key exists in the cache.
// Exists 检查键是否存在于缓存中。
func (c *cache) Exists(key string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	item, found := c.items[key]
	if !found {
		return false
	}
	
	// Check if the item has expired
	if item.expiration > 0 && item.expiration <= time.Now().UnixNano() {
		return false
	}
	
	return true
}

// Clear removes all values from the cache.
// Clear 从缓存中删除所有值。
func (c *cache) Clear() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.items = make(map[string]*cacheItem)
	c.stats.Size = 0
	
	// Emit event
	c.emitEvent(interfaces.CacheEventCleared, "", nil)
	
	return nil
}

// Size returns the number of items in the cache.
// Size 返回缓存中的项目数。
func (c *cache) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	return len(c.items)
}

// Keys returns all keys in the cache.
// Keys 返回缓存中的所有键。
func (c *cache) Keys() []string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	keys := make([]string, 0, len(c.items))
	for key := range c.items {
		keys = append(keys, key)
	}
	
	return keys
}

// GetStats returns cache statistics.
// GetStats 返回缓存统计信息。
func (c *cache) GetStats() interfaces.CacheStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	// Calculate hit rate
	totalAccess := c.stats.Hits + c.stats.Misses
	hitRate := 0.0
	if totalAccess > 0 {
		hitRate = float64(c.stats.Hits) / float64(totalAccess)
	}
	
	// Create a copy of stats to avoid race conditions
	stats := interfaces.CacheStats{
		Hits:          c.stats.Hits,
		Misses:        c.stats.Misses,
		Size:          c.stats.Size,
		MaxSize:       c.stats.MaxSize,
		Evictions:     c.stats.Evictions,
		AvgAccessTime: c.stats.AvgAccessTime,
		HitRate:       hitRate,
	}
	
	return stats
}

// ResetStats resets cache statistics.
// ResetStats 重置缓存统计信息。
func (c *cache) ResetStats() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	c.stats.Hits = 0
	c.stats.Misses = 0
	c.stats.Evictions = 0
	c.stats.AvgAccessTime = 0
	// Don't reset Size and MaxSize
}

// Close releases resources used by the cache.
// Close 释放缓存使用的资源。
func (c *cache) Close() error {
	// Stop the cleanup routine
	close(c.stopCleanup)
	
	// Clear the cache
	c.Clear()
	
	return nil
}

// AddEventListener adds a listener for cache events.
// AddEventListener 添加缓存事件的监听器。
func (c *cache) AddEventListener(listener interfaces.CacheEventListener) {
	c.listenerMutex.Lock()
	defer c.listenerMutex.Unlock()
	
	c.eventListeners = append(c.eventListeners, listener)
}

// RemoveEventListener removes a listener for cache events.
// RemoveEventListener 移除缓存事件的监听器。
func (c *cache) RemoveEventListener(listener interfaces.CacheEventListener) {
	c.listenerMutex.Lock()
	defer c.listenerMutex.Unlock()
	
	for i, l := range c.eventListeners {
		if l == listener {
			// Remove the listener by replacing it with the last element and truncating the slice
			c.eventListeners[i] = c.eventListeners[len(c.eventListeners)-1]
			c.eventListeners = c.eventListeners[:len(c.eventListeners)-1]
			break
		}
	}
}

// emitEvent emits a cache event to all listeners.
// emitEvent 向所有监听器发送缓存事件。
func (c *cache) emitEvent(eventType interfaces.CacheEventType, key string, value interface{}) {
	event := interfaces.CacheEvent{
		Type:      eventType,
		Key:       key,
		Value:     value,
		Timestamp: time.Now(),
	}
	
	// Notify listeners
	c.listenerMutex.RLock()
	listeners := make([]interfaces.CacheEventListener, len(c.eventListeners))
	copy(listeners, c.eventListeners)
	c.listenerMutex.RUnlock()
	
	for _, listener := range listeners {
		go listener.OnCacheEvent(event)
	}
}

// evict evicts an item from the cache based on the eviction policy.
// evict 根据驱逐策略从缓存中驱逐一个项目。
func (c *cache) evict() {
	if len(c.items) == 0 {
		return
	}
	
	var keyToEvict string
	
	switch c.evictionPolicy {
	case interfaces.EvictionPolicyLRU:
		// Least Recently Used
		var oldest int64 = time.Now().UnixNano()
		for key, item := range c.items {
			if item.lastAccess < oldest {
				oldest = item.lastAccess
				keyToEvict = key
			}
		}
	
	case interfaces.EvictionPolicyLFU:
		// Least Frequently Used
		var leastCount int64 = 1<<63 - 1 // Max int64
		for key, item := range c.items {
			if item.accessCount < leastCount {
				leastCount = item.accessCount
				keyToEvict = key
			}
		}
	
	case interfaces.EvictionPolicyFIFO:
		// First In First Out (we'll use the oldest item)
		var oldest int64 = time.Now().UnixNano()
		for key, item := range c.items {
			if item.lastAccess < oldest {
				oldest = item.lastAccess
				keyToEvict = key
			}
		}
	
	default:
		// Default to LRU
		var oldest int64 = time.Now().UnixNano()
		for key, item := range c.items {
			if item.lastAccess < oldest {
				oldest = item.lastAccess
				keyToEvict = key
			}
		}
	}
	
	// Evict the selected item
	if keyToEvict != "" {
		item := c.items[keyToEvict]
		delete(c.items, keyToEvict)
		c.stats.Evictions++
		
		// Emit event
		c.emitEvent(interfaces.CacheEventEvicted, keyToEvict, item.value)
	}
}

// updateAccessTime updates the average access time statistic.
// updateAccessTime 更新平均访问时间统计信息。
func (c *cache) updateAccessTime(duration time.Duration) {
	// Simple moving average
	if c.stats.Hits == 1 {
		c.stats.AvgAccessTime = duration
	} else {
		c.stats.AvgAccessTime = (c.stats.AvgAccessTime + duration) / 2
	}
}