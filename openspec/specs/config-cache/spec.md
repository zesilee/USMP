# config-cache — TTL+LRU 内存缓存原语

## Purpose

config-cache 提供无数据库约束（R03）下的通用内存缓存原语 `TTLLRUCache`：TTL 过期 + LRU 容量淘汰 + 协程安全 + 后台清理。该原语被**两处复用**：(a) 期望配置存储（`InMemoryConfigStore`，作为 `reconcile.ConfigStore` 后端，供 Manager 对账使用）；(b) 设备运行配置读缓存（§8，`GET /config` 命中优先，下发后主动失效）。参见 [[config-api]] 读写路径与 [[yang-controller-runtime]] ConfigStore 消费。

## Requirements

### Requirement: CC-01 键方案

`ConfigStore.Set/Get/Delete` SHALL 以 `key = "<deviceID>:<path>"` 唯一标识某设备某 YANG 路径的期望配置条目。运行配置读缓存 SHALL 使用 `key = "<ip>|<path>"`（`config_handler.go:runKey`，path 去尾 `/`）以区分不同复用场景。

#### Scenario: 期望配置键
- **WHEN** 对设备 `dev1` 的路径 `p1` 调用 `ConfigStore.Set/Get/Delete`
- **THEN** SHALL 以 `key="dev1:p1"` 定位条目，不与其他设备/路径冲突

#### Scenario: 运行缓存键
- **WHEN** 对 IP `10.0.0.1` 的路径 `/ifm` 读缓存
- **THEN** SHALL 以 `key="10.0.0.1|/ifm"` 定位，与期望配置键（`:` 分隔）命名空间隔离

### Requirement: CC-02 TTL 硬过期

条目 SHALL 自**写入时刻**计 TTL（非末次访问续期）；`Get` 时若 `time.Since(createdAt) > ttl` SHALL 删除该条目并返回未命中。期望配置存储实例 TTL 为 1min，运行读缓存实例 TTL 为 30s（§8）。

#### Scenario: 过期返回未命中
- **WHEN** 条目写入后经过时间超过其 TTL 再 `Get`
- **THEN** SHALL 删除该条目并返回未命中（`ok=false`）

#### Scenario: TTL 自写入时刻计
- **WHEN** 条目在 TTL 窗口内被多次 `Get` 命中
- **THEN** 命中 SHALL NOT 续期 TTL，过期仍以写入时刻为基准

### Requirement: CC-03 LRU 容量淘汰

条目数达到 `capacity` 时，新 `Set` SHALL 淘汰 `lastUsed` 最旧的条目以腾出空间。期望配置存储容量 1000，运行读缓存容量 4096。

#### Scenario: 满容淘汰最旧
- **WHEN** 缓存已满且写入一个新 key
- **THEN** SHALL 淘汰最久未使用（`lastUsed` 最旧）的条目，新条目写入成功

### Requirement: CC-04 协程安全

并发读写 SHALL 由 RWMutex 保证数据一致、无竞态（R09）；`Get` 命中后 SHALL 升级为写锁更新条目 `lastUsed`。

#### Scenario: 并发访问一致
- **WHEN** 多 goroutine 并发 `Set/Get/Delete` 同一或不同 key
- **THEN** SHALL NOT 发生数据竞态（`-race` 通过），读写结果一致

### Requirement: CC-05 后台清理与停止

后台 cleanup ticker 到期时 SHALL 调用 `ClearExpired` 批量清除已过期条目；`Stop` SHALL 经 `stopChan` 终止清理 goroutine，避免泄漏。

#### Scenario: 定期批量清理
- **WHEN** cleanup ticker 到期
- **THEN** SHALL 移除所有已超过 TTL 的条目

#### Scenario: 停止终止后台
- **WHEN** 调用 `Stop`
- **THEN** SHALL 关闭 `stopChan` 并终止清理 goroutine，无残留协程

### Requirement: CC-06 运行配置读缓存复用

同一 `TTLLRUCache` 原语 SHALL 作为设备**运行配置读缓存**被复用（Manager 持有独立的 `runningCache` 实例，TTL 30s）。`GET /config` SHALL 优先读运行缓存，命中返回 `source="cache"` 且携带 `cache_age_seconds`；`force_refresh=true` SHALL 绕过缓存强制回读设备并回填。下发（`POST /config`）成功后 SHALL 以 `InvalidatePrefix("<ip>|")` 主动失效该设备全部运行缓存条目（§8 下发后失效）。

> **契约注记**：本原语并非单一用途。它同时是 (a) desired-state 存储后端（`InMemoryConfigStore`）与 (b) running-config 读缓存两处的实现。早期"REST GetConfig 每次直连设备、force_refresh/30s 读缓存为 TODO"的描述已过时——运行读缓存与 `force_refresh` 均已实现（PR-B2），详见 [[config-api]]。

#### Scenario: 运行缓存命中
- **WHEN** `GET /config` 且运行缓存中该 `"<ip>|<path>"` 条目未过期、未带 `force_refresh`
- **THEN** SHALL 返回缓存值，`source="cache"`、`cached=true`，不访问设备

#### Scenario: force_refresh 绕缓存
- **WHEN** `GET /config` 带 `force_refresh=true`
- **THEN** SHALL 跳过运行缓存直接回读设备，`source="device"`，并回填缓存

#### Scenario: 下发后主动失效
- **WHEN** `POST /config` 对某 IP 下发成功
- **THEN** SHALL 调用 `InvalidatePrefix("<ip>|")` 清除该设备运行缓存，后续 `GET` 重新回读设备

### Requirement: CC-07 状态快照缓存实例

系统 SHALL 以既有 TTL+LRU 实现（CC-01..CC-05 同一机制与协程安全约束）另起独立实例承载 `include_state` 状态快照：键方案与运行配置缓存一致（`IP|路径`），TTL 独立（默认 30 秒，SHALL 可经环境变量 `USMP_STATE_SNAPSHOT_TTL` 配置），容量受自身 LRU 上限约束、超限按最近最少使用淘汰（万级行大条目被淘汰后 SHALL 走重拉，无崩溃路径，R08）。运行配置缓存的写后失效（CC-06）SHALL NOT 作用于本实例；本实例 SHALL NOT 持久化（R03）。

#### Scenario: 独立 TTL 生效

- **WHEN** 设置 `USMP_STATE_SNAPSHOT_TTL=10s` 并写入快照
- **THEN** 10 秒后读取 SHALL 未命中；运行配置缓存 TTL SHALL 不受影响仍为 30 秒

#### Scenario: 大条目 LRU 淘汰后降级重拉（边界）

- **WHEN** 快照实例达容量上限后写入新条目导致最旧快照被淘汰
- **THEN** 对被淘汰路径的下次读取 SHALL 未命中并触发设备重拉，SHALL NOT panic

#### Scenario: 并发读写安全（race）

- **WHEN** 多协程并发读快照、写快照与 TTL 过期清理
- **THEN** SHALL 无数据竞态（`-race` 通过），读到的值 SHALL 为完整快照或未命中，无撕裂
