---
name: go-ttl-lru-memory-cache
description: 无数据库架构，设备YANG配置内存缓存，TTL自动过期、LRU淘汰、协程安全，作为 ConfigStore 后端给 Manager 使用
---

# 技能详情（补全激活时机+核心原则+使用样例）
## 一、激活时机（何时自动触发）
1.  当用户需求包含「内存缓存」「TTL过期」「LRU淘汰」「缓存清理」等关键词时，自动激活。
2.  开发流程中，涉及「设备配置读取」「缓存管理」「配置下发后缓存失效」时，自动启用。
3.  与 Manager 联动，Manager 启动时自动初始化缓存，作为 ConfigStore 存储 desired 配置。

## 二、核心原则（底层设计逻辑）
1.  高性能原则：纯内存操作，无磁盘IO，确保配置读取响应速度，降低NETCONF请求频率。
2.  时效性原则：TTL自动过期，确保缓存数据与交换机实时配置一致，避免脏数据。
3.  安全原则：使用 sync 包保证协程安全，避免并发读写竞态；LRU淘汰机制防止内存溢出。
4.  联动原则：与 Manager、NETCONF 技能深度联动，配置下发后主动失效缓存，确保数据一致性。

## 三、使用样例（实操指令+输出效果）
### 样例1：触发技能指令
实现 TTL+LRU 内存缓存，Key 为设备 IP+YANG 路径，TTL 默认 30 秒，支持主动失效
plaintext

### 样例2：技能输出效果（符合规则，≤500行）
```go
package cache

import (
	"sync"
	"time"
)

// 缓存项结构体
type cacheItem struct {
	value      interface{}
	expireTime time.Time
}

// TTL+LRU缓存
type TTLLRUCache struct {
	mu         sync.RWMutex
	cache      map[string]*cacheItem
	lruList    []string          // LRU淘汰队列
	maxSize    int               // 最大缓存容量
	ttl        time.Duration     // 默认TTL
}

// 初始化缓存
func NewTTLLRUCache(maxSize int, cleanupInterval time.Duration, ttl time.Duration) *TTLLRUCache {
	c := &TTLLRUCache{
		cache:   make(map[string]*cacheItem),
		maxSize: maxSize,
		ttl:     ttl,
	}
	// 启动定时清理过期缓存
	go c.cleanExpired(cleanupInterval)
	return c
}

// 设置缓存
func (c *TTLLruCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 新增缓存项
	c.cache[key] = &cacheItem{
		value:      value,
		expireTime: time.Now().Add(c.ttl),
	}
	// 更新LRU队列
	c.updateLRU(key)
	// 超过最大容量，淘汰LRU末尾项
	if len(c.cache) > c.maxSize {
		淘汰Key := c.lruList[0]
		delete(c.cache, 淘汰Key)
		c.lruList = c.lruList[1:]
	}
}

// 获取缓存（自动判断过期）
func (c *TTLLRUCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.cache[key]
	if !ok {
		return nil, false
	}
	// 检查是否过期
	if time.Now().After(item.expireTime) {
		delete(c.cache, key)
		return nil, false
	}
	// 更新LRU队列
	c.updateLRU(key)
	return item.value, true
}

// 主动失效缓存
func (c *TTLLRUCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, key)
}

// 定时清理过期缓存
func (c *TTLLRUCache) cleanExpired(cleanupInterval time.Duration) {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		for key, item := range c.cache {
			if time.Now().After(item.expireTime) {
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}

// 更新LRU队列
func (c *TTLLRUCache) updateLRU(key string) {
	// 移除原有位置
	for i, k := range c.lruList {
		if k == key {
			c.lruList = append(c.lruList[:i], c.lruList[i+1:]...)
			break
		}
	}
	// 加入队尾
	c.lruList = append(c.lruList, key)
}
```

### 样例 3：联动其他技能

Manager 的 ConfigStore 获取设备 VLAN 配置，desired 状态存储在 TTL LRU 缓存，controller  reconcile 时读取。