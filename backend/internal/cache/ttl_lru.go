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
	// gen 写代：每次 Set/SetPending 递增，供 MarkExpiring 做「读到的还是这一版」守卫（CC-08）。
	gen uint64
	// pending 待同步：为 true 时不计 TTL（CC-02 例外），直到 MarkExpiring 转为已同步。
	pending bool
}

// expired reports whether e is past its TTL at now. Pending entries never
// expire: their clock only starts at MarkExpiring (CC-08).
func (c *TTLLRUCache) expired(e *entry, now time.Time) bool {
	return !e.pending && now.Sub(e.createdAt) > c.ttl
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

	if cleanupInterval > 0 {
		go c.cleanupLoop(cleanupInterval)
	}

	return c
}

// InitGlobalCache initializes the global cache used by the whole application
func InitGlobalCache() {
	// capacity / TTL / cleanup interval：TTL 30s 对齐 CLAUDE.md §8 的配置缓存口径
	// （下发后另由 Invalidate 主动失效，不等过期）。
	globalCache = NewTTLLRUCache(10000, 30*time.Second, 1*time.Minute)
}

// GetGlobalCache returns the global cache instance
func GetGlobalCache() *TTLLRUCache {
	return globalCache
}

// Set adds or updates a cache entry. The TTL clock starts now (CC-02).
func (c *TTLLRUCache) Set(key string, value interface{}) {
	c.put(key, value, false)
}

// SetPending adds or updates a cache entry in the pending state: it does not
// expire until MarkExpiring is called with the generation returned by Track
// (CC-08). Used by the desired-config store so an intent survives however long
// delivery to the device takes.
func (c *TTLLRUCache) SetPending(key string, value interface{}) {
	c.put(key, value, true)
}

// put is the shared write path: an existing entry is overwritten in place with
// its generation bumped; a new entry starts at generation 1.
func (c *TTLLRUCache) put(key string, value interface{}, pending bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	if e, exists := c.entries[key]; exists {
		e.value = value
		e.createdAt = now
		e.lastUsed = now
		e.gen++
		e.pending = pending
		return
	}

	if len(c.entries) >= c.capacity {
		c.evictLRU()
	}

	c.entries[key] = &entry{
		key:       key,
		value:     value,
		createdAt: now,
		lastUsed:  now,
		gen:       1,
		pending:   pending,
	}
}

// MarkExpiring turns a pending entry into an expiring one, starting its TTL
// clock now — but only if the entry is still at generation gen. A mismatch
// means the key was rewritten after the caller read it, so the newer value
// stays pending. Returns whether the mark was applied (CC-08).
func (c *TTLLRUCache) MarkExpiring(key string, gen uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	e, exists := c.entries[key]
	if !exists || e.gen != gen {
		return false
	}
	e.pending = false
	e.createdAt = time.Now()
	return true
}

// Track reports the write generation of key, when it was last written
// (pendingSince, meaningful while pending) and whether it is still pending.
// ok is false for a missing or expired key. Read-only: no LRU touch.
func (c *TTLLRUCache) Track(key string) (gen uint64, pendingSince time.Time, pending, ok bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	e, exists := c.entries[key]
	if !exists || c.expired(e, time.Now()) {
		return 0, time.Time{}, false, false
	}
	return e.gen, e.createdAt, e.pending, true
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
	now := time.Now()
	if c.expired(e, now) {
		delete(c.entries, key)
		return nil, false
	}
	e.lastUsed = now
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
	now := time.Now()
	age := now.Sub(e.createdAt)
	if c.expired(e, now) {
		delete(c.entries, key)
		return nil, 0, false
	}
	e.lastUsed = now
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

// DeleteIfGen removes the entry only if it is still at generation gen, so a
// caller deciding on a value it read earlier cannot delete a newer write.
// Returns whether an entry was removed.
func (c *TTLLRUCache) DeleteIfGen(key string, gen uint64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, exists := c.entries[key]
	if !exists || e.gen != gen {
		return false
	}
	delete(c.entries, key)
	return true
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
		if c.expired(e, now) {
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
		if !c.expired(e, now) {
			keys = append(keys, k)
		}
	}
	return keys
}

func (c *TTLLRUCache) evictLRU() {
	var lruKey string
	var oldestTime time.Time

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
