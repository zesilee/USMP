## MODIFIED Requirements

### Requirement: CC-02 TTL 硬过期

条目 SHALL 自**写入时刻**计 TTL（非末次访问续期）；`Get` 时若 `time.Since(createdAt) > ttl` SHALL 删除该条目并返回未命中。期望配置存储实例 TTL 为 1min，运行读缓存实例 TTL 为 30s（§8）。**例外**：经 `SetPending` 写入且尚未 `MarkExpiring` 的**待同步**条目 SHALL NOT 计 TTL（见 CC-08），`Get/GetWithAge/Keys/ClearExpired` SHALL 一律视其为未过期。

#### Scenario: 过期返回未命中
- **WHEN** 条目写入后经过时间超过其 TTL 再 `Get`
- **THEN** SHALL 删除该条目并返回未命中（`ok=false`）

#### Scenario: TTL 自写入时刻计
- **WHEN** 条目在 TTL 窗口内被多次 `Get` 命中
- **THEN** 命中 SHALL NOT 续期 TTL，过期仍以写入时刻为基准

#### Scenario: 待同步条目不过期
- **WHEN** 条目经 `SetPending` 写入，经过远超 TTL 的时间后 `Get`
- **THEN** SHALL 命中返回原值，`ClearExpired` SHALL NOT 清除它

## ADDED Requirements

### Requirement: CC-08 待同步/已同步两态与写代

`TTLLRUCache` SHALL 提供两态条目原语：`SetPending(key, value)` 写入**待同步**条目并递增该 key 的写代 `gen`；`MarkExpiring(key, gen)` 仅当 `gen` 与当前写代一致时 SHALL 把条目转为**已同步**并以当下为起点计 TTL，否则 SHALL 返回 false 且条目保持待同步；`Track(key)` SHALL 返回 `(gen, pendingSince, pending, ok)`。对已同步条目再次 `SetPending` SHALL 退回待同步并递增 `gen`。既有 `Set` 语义 SHALL 保持不变（写入即计 TTL）。以上操作 SHALL 与 CC-04 同一把锁，`-race` 全绿。

#### Scenario: 同步确认后起算 TTL
- **WHEN** `SetPending` 后经过 2×TTL，再 `MarkExpiring(key, gen)` 成功，再经过 TTL+ε
- **THEN** `MarkExpiring` 前 `Get` SHALL 命中；`MarkExpiring` 后 TTL 内 SHALL 命中，超过 TTL SHALL 未命中

#### Scenario: 写代守卫
- **WHEN** `Track` 得 `gen=1`，随后 `SetPending` 同 key（`gen` 变 2），再 `MarkExpiring(key, 1)`
- **THEN** SHALL 返回 false，条目 SHALL 仍为待同步（`Track.pending=true`），SHALL 不过期

#### Scenario: 再写入退回待同步
- **WHEN** 条目已 `MarkExpiring`，再次 `SetPending` 同 key
- **THEN** `Track.pending` SHALL 为 true，`pendingSince` SHALL 为本次写入时刻，`gen` SHALL 递增

#### Scenario: 并发两态操作
- **WHEN** 多协程并发 `SetPending/MarkExpiring/Track/Get/ClearExpired` 同一批 key
- **THEN** SHALL 无数据竞态（`-race`），SHALL NOT panic
