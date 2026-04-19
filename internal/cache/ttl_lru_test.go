package cache

import (
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
