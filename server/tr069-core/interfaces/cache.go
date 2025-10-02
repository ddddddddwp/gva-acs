// Package interfaces defines the interfaces for the TR069 protocol implementation.
package interfaces

import (
	"time"
)

// Cache defines the interface for a cache implementation.
type Cache interface {
	// Get retrieves a value from the cache.
	Get(key string) (interface{}, bool)

	// Set stores a value in the cache with an optional TTL.
	Set(key string, value interface{}, ttl time.Duration) error

	// Delete removes a value from the cache.
	Delete(key string) error

	// Exists checks if a key exists in the cache.
	Exists(key string) bool

	// Clear removes all values from the cache.
	Clear() error

	// Size returns the number of items in the cache.
	Size() int

	// Keys returns all keys in the cache.
	Keys() []string

	// GetStats returns cache statistics.
	GetStats() CacheStats

	// ResetStats resets cache statistics.
	ResetStats()

	// Close releases resources used by the cache.
	Close() error

	// AddEventListener adds a listener for cache events.
	AddEventListener(listener CacheEventListener)

	// RemoveEventListener removes a listener for cache events.
	RemoveEventListener(listener CacheEventListener)
}

// CacheManager defines the interface for managing multiple caches.
type CacheManager interface {
	// CreateCache creates a new cache with the specified name and options.
	CreateCache(name string, options ...CacheOption) (Cache, error)

	// GetCache returns a cache by name.
	GetCache(name string) (Cache, error)

	// DeleteCache removes a cache by name.
	DeleteCache(name string) error

	// CacheExists checks if a cache exists.
	CacheExists(name string) bool

	// GetCacheNames returns the names of all caches.
	GetCacheNames() []string

	// GetAllCacheStats returns statistics for all caches.
	GetAllCacheStats() map[string]CacheStats

	// Close releases resources used by all caches.
	Close() error

	// EnablePreloading enables or disables cache preloading.
	EnablePreloading(enabled bool)

	// IsPreloadingEnabled returns whether cache preloading is enabled.
	IsPreloadingEnabled() bool

	// PreloadCache preloads data into a cache.
	PreloadCache(name string, data map[string]interface{}) error
}

// EvictionPolicy defines the policy for evicting items from a cache.
type EvictionPolicy int

const (
	// EvictionPolicyLRU evicts the least recently used item.
	EvictionPolicyLRU EvictionPolicy = iota

	// EvictionPolicyLFU evicts the least frequently used item.
	EvictionPolicyLFU

	// EvictionPolicyFIFO evicts the first item added to the cache.
	EvictionPolicyFIFO
)

// CacheStats contains statistics about a cache.
type CacheStats struct {
	// Hits is the number of cache hits.
	Hits int64

	// Misses is the number of cache misses.
	Misses int64

	// Size is the current number of items in the cache.
	Size int

	// MaxSize is the maximum number of items the cache has held.
	MaxSize int

	// Evictions is the number of items evicted from the cache.
	Evictions int64

	// AvgAccessTime is the average time to access an item.
	AvgAccessTime time.Duration

	// HitRate is the ratio of hits to total accesses.
	HitRate float64
}

// CacheEventType defines the type of cache event.
type CacheEventType int

const (
	// CacheEventItemAdded is emitted when an item is added to the cache.
	CacheEventItemAdded CacheEventType = iota

	// CacheEventItemUpdated is emitted when an item in the cache is updated.
	CacheEventItemUpdated

	// CacheEventItemRemoved is emitted when an item is removed from the cache.
	CacheEventItemRemoved

	// CacheEventItemExpired is emitted when an item in the cache expires.
	CacheEventItemExpired

	// CacheEventEvicted is emitted when an item is evicted from the cache.
	CacheEventEvicted

	// CacheEventCleared is emitted when the cache is cleared.
	CacheEventCleared
)

// CacheEvent represents an event that occurred in a cache.
type CacheEvent struct {
	// Type is the type of event.
	Type CacheEventType

	// Key is the key of the item involved in the event.
	Key string

	// Value is the value of the item involved in the event.
	Value interface{}

	// Timestamp is when the event occurred.
	Timestamp time.Time
}

// CacheEventListener defines the interface for listening to cache events.
type CacheEventListener interface {
	// OnCacheEvent is called when a cache event occurs.
	OnCacheEvent(event CacheEvent)
}

// CacheOption is a function that configures a cache.
type CacheOption func(cache Cache)

// WithCapacity sets the maximum capacity of the cache.
func WithCapacity(capacity int) CacheOption {
	return func(cache Cache) {
		if c, ok := cache.(interface{ setCapacity(int) }); ok {
			c.setCapacity(capacity)
		}
	}
}

// WithEvictionPolicy sets the eviction policy of the cache.
func WithEvictionPolicy(policy EvictionPolicy) CacheOption {
	return func(cache Cache) {
		if c, ok := cache.(interface{ setEvictionPolicy(EvictionPolicy) }); ok {
			c.setEvictionPolicy(policy)
		}
	}
}

// WithCleanupInterval sets the interval at which expired items are cleaned up.
func WithCleanupInterval(interval time.Duration) CacheOption {
	return func(cache Cache) {
		if c, ok := cache.(interface{ setCleanupInterval(time.Duration) }); ok {
			c.setCleanupInterval(interval)
		}
	}
}

// WithDefaultTTL sets the default TTL for items added to the cache.
func WithDefaultTTL(ttl time.Duration) CacheOption {
	return func(cache Cache) {
		if c, ok := cache.(interface{ setDefaultTTL(time.Duration) }); ok {
			c.setDefaultTTL(ttl)
		}
	}
}

// WithEventListener adds an event listener to the cache.
func WithEventListener(listener CacheEventListener) CacheOption {
	return func(cache Cache) {
		cache.AddEventListener(listener)
	}
}
