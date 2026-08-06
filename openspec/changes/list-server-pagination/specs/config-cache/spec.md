# Delta: config-cache

## ADDED Requirements

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
