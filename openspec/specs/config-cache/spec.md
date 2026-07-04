# config-cache — 行为契约（反向还原）

> 反向还原自 `backend/internal/cache/ttl_lru.go` + `manager.InMemoryConfigStore`，忠实 as-built。详见 `design.md`。权威（R03）。

## 能力概述

无数据库约束下的期望配置内存存储：TTL 过期 + LRU 淘汰 + 协程安全，作为 `reconcile.ConfigStore` 后端。

## 行为契约

### CC-01 键方案
- **Given** 某设备某 YANG 路径的期望配置
- **When** `ConfigStore.Set/Get/Delete`
- **Then** key = `"<deviceID>:<path>"`（`manager.go:32`）

### CC-02 TTL 硬过期
- **Given** 条目写入 `ttl` 之前
- **When** `Get` 读取
- **Then** 若 `time.Since(createdAt) > ttl` 则删除并返回未命中；TTL 自**写入时刻**计（非末次访问）

### CC-03 LRU 容量淘汰
- **Given** 条目数达 `capacity`
- **When** 新 `Set`
- **Then** 淘汰 `lastUsed` 最旧条目（O(n) 线性扫描）

### CC-04 协程安全
- **Given** 并发读写
- **When** 多 goroutine 访问
- **Then** RWMutex 保证一致；`Get` 命中后升级 Lock 更新 `lastUsed`

### CC-05 后台清理
- **Given** 缓存运行
- **When** cleanup ticker 到期
- **Then** `ClearExpired` 批量清除过期条目；`Stop` 经 `stopChan` 终止

## 关键契约边界（详见 design.md §4）

- **不是设备配置读缓存**：REST `GetConfig` 每次直连设备，不经本缓存；`force_refresh`/30s 读缓存为 TODO。缓存实为 **desired-state 后端**，TTL=5min。

## 关联
- `design.md`、`yang-controller-runtime/spec.md`（ConfigStore 消费）、`config-api`（确认未接读缓存）、`go-ttl-lru-memory-cache` 技能。
