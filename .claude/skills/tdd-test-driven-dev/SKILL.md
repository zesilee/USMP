---
name: tdd-test-driven-dev
description: 所有功能优先编写单元测试用例，再实现业务代码，覆盖正常、异常、边界、并发场景
---

# 技能详情
## 一、激活时机（何时自动触发）
1.  当用户需求包含「测试用例」「TDD」「单元测试」「异常测试」等关键词时，自动激活。
2.  开发流程中，任何新功能、BUG修复，在编写业务代码前，自动触发本技能，生成测试用例。
3.  与其他所有技能联动，每个技能的实现代码，均需先通过本技能生成测试用例，再编写代码。

## 二、核心原则（底层设计逻辑）
1.  测试先行原则：测试用例定义功能需求和边界，引导业务代码开发，避免偏离需求。
2.  全面覆盖原则：测试用例需覆盖正常流程、异常流程、边界场景、并发场景，确保代码健壮性。
3.  闭环原则：每个原子功能对应一套完整测试用例，测试通过后，再提交业务代码，形成闭环。
4.  可维护原则：测试用例与业务代码同步更新，确保代码迭代后，测试仍能有效验证功能。

## 三、使用样例（实操指令+输出效果）
### 样例1：触发技能指令
为 TTL+LRU 缓存的 Set、Get、Invalidate 方法，编写 TDD 测试用例
plaintext

### 样例2：技能输出效果（先写测试，后写代码）
```go
package cache_test

import (
	"testing"
	"time"
	"你的项目路径/cache"
)

// 测试正常Set+Get缓存
func TestTTLLRUCache_SetAndGet(t *testing.T) {
	// 初始化缓存（最大容量10，TTL30秒）
	c := cache.NewTTLLRUCache(10, 30*time.Second)
	key := "192.168.1.1/interfaces"
	value := "test config"

	// 执行Set操作
	c.Set(key, value)

	// 执行Get操作，验证结果
	val, ok := c.Get(key)
	if !ok {
		t.Errorf("Get缓存失败，期望存在key: %s", key)
	}
	if val != value {
		t.Errorf("缓存值不匹配，期望: %s，实际: %v", value, val)
	}
}

// 测试缓存过期
func TestTTLLRUCache_Expire(t *testing.T) {
	// 初始化缓存（TTL1秒）
	c := cache.NewTTLLRUCache(10, 1*time.Second)
	key := "192.168.1.1/interfaces"
	value := "test config"

	c.Set(key, value)
	// 等待2秒，确保缓存过期
	time.Sleep(2 * time.Second)

	// 验证缓存已过期
	val, ok := c.Get(key)
	if ok {
		t.Errorf("缓存未过期，期望过期key: %s，实际值: %v", key, val)
	}
}

// 测试主动失效缓存
func TestTTLLRUCache_Invalidate(t *testing.T) {
	c := cache.NewTTLLRUCache(10, 30*time.Second)
	key := "192.168.1.1/interfaces"
	value := "test config"

	c.Set(key, value)
	// 主动失效
	c.Invalidate(key)

	// 验证缓存已失效
	val, ok := c.Get(key)
	if ok {
		t.Errorf("缓存未失效，期望失效key: %s，实际值: %v", key, val)
	}
}

// 测试并发Set+Get（协程安全）
func TestTTLLRUCache_Concurrent(t *testing.T) {
	c := cache.NewTTLLRUCache(100, 30*time.Second)
	key := "192.168.1.1/interfaces"
	value := "test config"

	// 并发执行1000次Set+Get
	for i := 0; i < 1000; i++ {
		go func() {
			c.Set(key, value)
			val, ok := c.Get(key)
			if !ok || val != value {
				t.Errorf("并发缓存操作失败，val: %v, ok: %v", val, ok)
			}
		}()
	}

	// 等待所有协程执行完成
	time.Sleep(1 * time.Second)
}
```

### 样例 3：联动其他技能
为Controller的Reconcile功能，编写TDD测试用例，覆盖正常配置对齐、配置下发失败重试异常场景