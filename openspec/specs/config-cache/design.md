# config-cache — 配置缓存架构设计（反向还原）

> **权威性**：✅ 权威（R03：无数据库，仅 TTL+LRU 内存 + JSON）。
> **还原基准**：`main@b1cfbae`，代码 `backend/internal/cache/ttl_lru.go` + `backend/pkg/yang-runtime/manager/manager.go`。
> **上层导航**：`openspec/specs/system-architecture/design.md`。

## 1. 职责

在**无数据库**约束下（R03），为 Stack B 提供**期望配置(desired state)**的内存存储：TTL 自动过期 + LRU 容量淘汰 + 协程安全。作为 `reconcile.ConfigStore` 的后端被 Manager 使用。

## 2. 组件

### 2.1 `TTLLRUCache` — `internal/cache/ttl_lru.go`
- 结构 `ttl_lru.go:16`：`map[string]*entry` + `sync.RWMutex` + capacity + ttl + `time.Ticker` 清理。
- **并发安全**：`Set`/`Delete`/`ClearExpired`/`evictLRU` 取 `Lock`；`Get` 先 `RLock` 查找再升级 `Lock` 更新 `lastUsed`（`ttl_lru.go:85`）；`Size` 用 `RLock`。
- **TTL 过期**：读时惰性——`Get` 判 `time.Since(createdAt) > ttl` 则删（`ttl_lru.go:94`）。**TTL 自写入时刻(createdAt)计，非末次访问**，故条目硬过期。后台 `cleanupLoop` 定时 `ClearExpired`（`ttl_lru.go:161`），`Stop` 经 `stopChan` 停（`ttl_lru.go:133`）。
- **LRU 淘汰**：`Set` 时若 `len >= capacity` 触发 `evictLRU`（`ttl_lru.go:70,144`）——O(n) 线性扫描最旧 `lastUsed`（无链表/堆）。

### 2.2 `InMemoryConfigStore` — `manager/manager.go:19`
- 实现 `reconcile.ConfigStore`（Get/Set/Delete/List/ListDevices），底层持 `TTLLRUCache`。
- **key 方案**：`key = fmt.Sprintf("%s:%s", deviceID, path)`（`manager.go:32,42,49`）→ `"<设备IP>:<YANG路径>"`。符合 §8「Key=设备IP+YANG路径」。
- 构造：`manager.New` 建 `cache.NewTTLLRUCache(1000, 1min, 5min)` → `NewInMemoryConfigStore`（`manager.go:123`），经 `GetConfigStore()` 暴露。

## 3. 数据流（缓存在系统中的真实角色）

```
REST POST /api/v1/config → SetConfig
   → mgr.GetConfigStore().Set(deviceID:path, desired ygot)   # 写入缓存=期望态
   → mgr.TriggerReconcile(deviceID, path)                    # 触发对齐
                                                             │
GenericReconciler.Reconcile ── ConfigStore.Get(desired) ◀────┘
```

## 4. ⚠️ 关键事实：缓存不是「设备配置读缓存」

反向探查澄清一个易误解点：

- **REST config handler 不使用本缓存做读缓存**。`internal/api/config_handler.go` 零 `internal/cache` 导入；`GetConfig` 每次直连设备 NETCONF get-config（`config_handler.go:57`），`force_refresh` 是**已解析未实现**的 TODO（`config_handler.go:32`）。故 §8「运行配置缓存 TTL 30s」在 REST 读路径上**尚未落地**。
- 缓存**真实用途**是 **desired-state 后端**（ConfigStore），TTL=5min（非 §8 所述 30s），非 device-config 响应缓存。
- `InitGlobalCache`/`GetGlobalCache`（`ttl_lru.go:24`）为**死代码**，包外无调用者。
- 其余引用（`internal/controller/*/reconciler_integration_test.go`，`NewTTLLRUCache(100,30s,1m)`）**均为测试**。

## 5. as-built 缺口

| 缺口 | 位置 | 说明 |
|------|------|------|
| 读缓存未落地 | `config_handler.go:32` | `force_refresh`/30s TTL 读缓存为 TODO；GET 每次直连设备 |
| TTL 与 §8 不符 | `manager.go:125` | 实际 5min，§8 描述 30s |
| GlobalCache 死代码 | `ttl_lru.go:24` | 无调用者 |
| LRU O(n) 扫描 | `ttl_lru.go:144` | 容量大时淘汰成本线性；当前 cap=1000 可接受 |

## 6. 红线对照

- **R03 无数据库**：✅ 仅内存 map + 元信息 JSON，无 MySQL/Redis/SQLite。（注意 Stack A 的 etcd 依赖是另一条与 R03 张力的路径，见 `system-architecture/design.md` §6。）

## 7. 关联
- `go-ttl-lru-memory-cache` 技能；`yang-controller-runtime/design.md`（ConfigStore 消费方）；`config-api`（REST 读写接口，确认未接读缓存）。
