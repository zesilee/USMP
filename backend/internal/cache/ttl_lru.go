package cache

import (
	"strings"
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

// Get retrieves a cache entry, returns (value, found).
// The whole read is under the write lock: Set mutates *entry fields in place,
// so createdAt/value must not be read after unlocking (R09 data race).
func (c *TTLLRUCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, exists := c.entries[key]
	if !exists {
		return nil, false
	}
	if time.Since(e.createdAt) > c.ttl {
		delete(c.entries, key)
		return nil, false
	}
	e.lastUsed = time.Now()
	return e.value, true
}

// GetWithAge retrieves a cache entry together with how long it has been cached.
// Returns (value, age, true) on a fresh hit; (nil, 0, false) on miss or expiry
// (expired entries are deleted, consistent with Get). Used to surface cache-age
// to API consumers (e.g. the freshness indicator).
func (c *TTLLRUCache) GetWithAge(key string) (interface{}, time.Duration, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, exists := c.entries[key]
	if !exists {
		return nil, 0, false
	}
	age := time.Since(e.createdAt)
	if age > c.ttl {
		delete(c.entries, key)
		return nil, 0, false
	}
	e.lastUsed = time.Now()
	return e.value, age, true
}

// TTL returns the configured time-to-live for entries.
func (c *TTLLRUCache) TTL() time.Duration { return c.ttl }

// Invalidate explicitly invalidates a cache entry
func (c *TTLLRUCache) Invalidate(key string) {
	c.Delete(key)
}

// InvalidatePrefix removes all entries whose key starts with prefix. Used to
// evict every cached path of a device at once (e.g. after a config push).
func (c *TTLLRUCache) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.entries {
		if strings.HasPrefix(k, prefix) {
			delete(c.entries, k)
		}
	}
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

// Keys returns a snapshot of the keys of all non-expired entries. Expired
// entries are excluded (consistent with Get) but not evicted here; the cleanup
// loop reclaims them. Safe for concurrent use.
func (c *TTLLRUCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	now := time.Now()
	keys := make([]string, 0, len(c.entries))
	for k, e := range c.entries {
		if now.Sub(e.createdAt) <= c.ttl {
			keys = append(keys, k)
		}
	}
	return keys
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
