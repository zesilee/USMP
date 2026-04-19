package cache

import (
	"sync"
	"time"
)

type entry struct {
	key       string
	value     interface{}
	createdAt time.Time
	lastUsed  time.Time
}

// TTLLRUCache implements a thread-safe TTL-based LRU cache
type TTLLRUCache struct {
	capacity int
	ttl      time.Duration
	entries  map[string]*entry
	mu       sync.RWMutex
	stopChan chan struct{}
}

var globalCache *TTLLRUCache

// NewTTLLRUCache creates a new TTL+LRU cache
func NewTTLLRUCache(capacity int, ttl time.Duration, cleanupInterval time.Duration) *TTLLRUCache {
	c := &TTLLRUCache{
		capacity: capacity,
		ttl:      ttl,
		entries:  make(map[string]*entry),
		stopChan: make(chan struct{}),
	}

	// Start background cleanup
	if cleanupInterval > 0 {
		go c.cleanupLoop(cleanupInterval)
	}

	return c
}

// InitGlobalCache initializes the global cache used by the whole application
func InitGlobalCache() {
	// Default: 10000 entries, 30s TTL, 1min cleanup interval
	globalCache = NewTTLLRUCache(10000, 30*time.Second, 1*time.Minute)
}

// GetGlobalCache returns the global cache instance
func GetGlobalCache() *TTLLRUCache {
	return globalCache
}

// Set adds or updates a cache entry
func (c *TTLLRUCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	// If key exists, update it
	if e, exists := c.entries[key]; exists {
		e.value = value
		e.createdAt = now
		e.lastUsed = now
		return
	}

	// If at capacity, evict LRU entry
	if len(c.entries) >= c.capacity {
		c.evictLRU()
	}

	// Add new entry
	c.entries[key] = &entry{
		key:       key,
		value:     value,
		createdAt: now,
		lastUsed:  now,
	}
}

// Get retrieves a cache entry, returns (value, found)
func (c *TTLLRUCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	e, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Since(e.createdAt) > c.ttl {
		c.Delete(key)
		return nil, false
	}

	// Update last used time
	c.mu.Lock()
	e.lastUsed = time.Now()
	c.mu.Unlock()

	return e.value, true
}

// Invalidate explicitly invalidates a cache entry
func (c *TTLLRUCache) Invalidate(key string) {
	c.Delete(key)
}

// Delete removes an entry from the cache
func (c *TTLLRUCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// ClearExpired removes all expired entries
func (c *TTLLRUCache) ClearExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for k, e := range c.entries {
		if now.Sub(e.createdAt) > c.ttl {
			delete(c.entries, k)
		}
	}
}

// Stop stops the background cleanup
func (c *TTLLRUCache) Stop() {
	close(c.stopChan)
}

// Size returns the current number of entries
func (c *TTLLRUCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

func (c *TTLLRUCache) evictLRU() {
	var lruKey string
	var oldestTime time.Time

	// Find the entry with the oldest lastUsed
	for _, e := range c.entries {
		if lruKey == "" || e.lastUsed.Before(oldestTime) {
			lruKey = e.key
			oldestTime = e.lastUsed
		}
	}

	if lruKey != "" {
		delete(c.entries, lruKey)
	}
}

func (c *TTLLRUCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.ClearExpired()
		case <-c.stopChan:
			return
		}
	}
}
