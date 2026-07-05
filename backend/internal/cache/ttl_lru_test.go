package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacheSetGet(t *testing.T) {
	cache := NewTTLLRUCache(100, 5*time.Second, 10*time.Second)
	defer cache.Stop()

	cache.Set("key1", "value1")

	val, ok := cache.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)
}

func TestCacheExpire(t *testing.T) {
	cache := NewTTLLRUCache(100, 100*time.Millisecond, 10*time.Second)
	defer cache.Stop()

	cache.Set("key1", "value1")

	// 立即获取应该命中
	val, ok := cache.Get("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", val)

	// 等待过期
	time.Sleep(150 * time.Millisecond)

	// 应该过期未命中
	_, ok = cache.Get("key1")
	assert.False(t, ok)
}

func TestCacheInvalidate(t *testing.T) {
	cache := NewTTLLRUCache(100, 5*time.Second, 10*time.Second)
	defer cache.Stop()

	cache.Set("key1", "value1")
	cache.Invalidate("key1")

	_, ok := cache.Get("key1")
	assert.False(t, ok)
}

func TestCacheLRUEviction(t *testing.T) {
	// 容量为3，存入4个会淘汰最久未使用
	cache := NewTTLLRUCache(3, 5*time.Second, 10*time.Second)
	defer cache.Stop()

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// 访问key1，更新使用顺序
	cache.Get("key1")

	// 存入第四个，应该淘汰key2（最久未使用）
	cache.Set("key4", "value4")

	// key1, key3, key4 应该存在
	_, ok := cache.Get("key1")
	assert.True(t, ok)
	_, ok = cache.Get("key3")
	assert.True(t, ok)
	_, ok = cache.Get("key4")
	assert.True(t, ok)

	// key2 应该被淘汰
	_, ok = cache.Get("key2")
	assert.False(t, ok)
}

func TestCacheConcurrency(t *testing.T) {
	cache := NewTTLLRUCache(1000, 5*time.Second, 10*time.Second)
	defer cache.Stop()

	done := make(chan bool)
	iterations := 1000

	for i := 0; i < 100; i++ {
		go func(i int) {
			for j := 0; j < iterations; j++ {
				cache.Set("key"+string(rune(i)), "value"+string(rune(i)))
				cache.Get("key" + string(rune(i)))
			}
			done <- true
		}(i)
	}

	// 等待所有goroutine完成
	for i := 0; i < 100; i++ {
		<-done
	}

	// 如果没有死锁就是成功
	assert.True(t, true)
}

func TestGlobalCache(t *testing.T) {
	InitGlobalCache()

	globalCache := GetGlobalCache()
	assert.NotNil(t, globalCache)

	globalCache.Set("test", "global")
	val, ok := globalCache.Get("test")
	assert.True(t, ok)
	assert.Equal(t, "global", val)
}

func TestClearExpired(t *testing.T) {
	cache := NewTTLLRUCache(100, 50*time.Millisecond, 10*time.Millisecond)
	defer cache.Stop()

	cache.Set("key1", "value1")
	cache.Set("key2", "value2")

	time.Sleep(100 * time.Millisecond)

	// 手动触发清理
	cache.ClearExpired()

	_, ok1 := cache.Get("key1")
	_, ok2 := cache.Get("key2")
	assert.False(t, ok1)
	assert.False(t, ok2)
}

func TestGetWithAge_HitFreshMissExpired(t *testing.T) {
	c := NewTTLLRUCache(10, 60*time.Millisecond, 0)
	c.Set("k", "v")

	val, age, ok := c.GetWithAge("k")
	if !ok || val != "v" {
		t.Fatalf("hit: got (%v,%v), want (v,true)", val, ok)
	}
	if age < 0 || age > 40*time.Millisecond {
		t.Errorf("fresh age = %v, want ~0 (<40ms)", age)
	}

	if _, _, ok := c.GetWithAge("missing"); ok {
		t.Errorf("miss should report found=false")
	}

	time.Sleep(120 * time.Millisecond) // > TTL
	if _, _, ok := c.GetWithAge("k"); ok {
		t.Errorf("expired entry should report found=false")
	}
}

func TestGetWithAge_Monotonic(t *testing.T) {
	c := NewTTLLRUCache(10, 500*time.Millisecond, 0)
	c.Set("k", "v")
	time.Sleep(60 * time.Millisecond)
	_, age, ok := c.GetWithAge("k")
	if !ok {
		t.Fatalf("expected hit")
	}
	if age <= 0 { // lower bound only, avoid upper-bound flakiness under load
		t.Errorf("age = %v, want > 0 after sleep", age)
	}
}

func TestInvalidatePrefix(t *testing.T) {
	c := NewTTLLRUCache(10, time.Minute, 0)
	c.Set("10.0.0.1|/vlans", "a")
	c.Set("10.0.0.1|/ifm", "b")
	c.Set("10.0.0.2|/vlans", "c")

	c.InvalidatePrefix("10.0.0.1|")

	if _, ok := c.Get("10.0.0.1|/vlans"); ok {
		t.Errorf("10.0.0.1|/vlans should be invalidated")
	}
	if _, ok := c.Get("10.0.0.1|/ifm"); ok {
		t.Errorf("10.0.0.1|/ifm should be invalidated")
	}
	if _, ok := c.Get("10.0.0.2|/vlans"); !ok {
		t.Errorf("other device 10.0.0.2 must not be invalidated")
	}
}

func TestTTLGetter(t *testing.T) {
	c := NewTTLLRUCache(10, 30*time.Second, 0)
	if c.TTL() != 30*time.Second {
		t.Errorf("TTL() = %v, want 30s", c.TTL())
	}
}

// TestGetWithAge_SameKeyConcurrent reproduces the race where concurrent GET
// (miss -> Set) on the SAME key mutates one *entry in place while other
// goroutines read it via GetWithAge/Get. Must be clean under -race.
func TestGetWithAge_SameKeyConcurrent(t *testing.T) {
	c := NewTTLLRUCache(10, time.Minute, 0)
	c.Set("k", 0)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func(n int) { defer wg.Done(); c.Set("k", n) }(i)
		go func() { defer wg.Done(); c.GetWithAge("k") }()
		go func() { defer wg.Done(); c.Get("k") }()
	}
	wg.Wait()
}

// TestGetWithAge_AfterLRUEviction: an evicted entry must report found=false.
func TestGetWithAge_AfterLRUEviction(t *testing.T) {
	c := NewTTLLRUCache(1, time.Minute, 0) // capacity 1
	c.Set("a", "va")
	c.Set("b", "vb") // evicts "a" (LRU)
	if _, _, ok := c.GetWithAge("a"); ok {
		t.Errorf("evicted key 'a' should report found=false")
	}
	if _, _, ok := c.GetWithAge("b"); !ok {
		t.Errorf("key 'b' should be present")
	}
}
