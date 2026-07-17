## MODIFIED Requirements

### Requirement: BR-11 归属警告（软归属）

`POST /api/v1/config/:ip/*path` 与 `DELETE` 行删除命中业务意图认领路径时，缺省 SHALL 拒绝写入：信封码 409、message 指引「先删除/修改对应意图，或携带 force=true 强制下发」、data SHALL 携带认领意图 CR 名称列表（`intents`）；被拒请求 SHALL NOT 触达设备编解码与下发，SHALL NOT 产生审计记录。携带 query `force=true` 时 SHALL 放行下发，响应 SHALL 附加归属警告字段（认领意图名称、提示意图收敛会覆盖手改），且审计记录 SHALL 携带 force 标记与被覆盖的认领意图（见 operation-audit OA-01）。归属查询 SHALL 经 API 暴露供前端徽标渲染（一期行为不变）。

#### Scenario: 手改命中认领路径缺省拒绝
- **WHEN** 对被意图 X 认领的 vlan 路径 POST 配置（无 force）
- **THEN** SHALL 返回信封码 409、`success=false`，data.intents 含意图 X，SHALL NOT 下发、SHALL NOT 记审计

#### Scenario: force 放行并附警告
- **WHEN** 对被意图 X 认领的路径 POST 配置且 `force=true`
- **THEN** 下发 SHALL 照常接受，响应 SHALL 含 `ownershipWarning`（意图 X 名称与提示文案）

#### Scenario: 行删除同受硬锁
- **WHEN** 对被认领条目 DELETE（无 force）
- **THEN** SHALL 返回信封码 409 并携带认领意图，SHALL NOT 向设备下发删除

#### Scenario: 未认领路径无警告不拦截
- **WHEN** 对无任何意图认领的路径 POST 配置
- **THEN** 下发 SHALL 照常接受，响应 SHALL 不含归属警告字段

#### Scenario: 兄弟路径不受锁（负路径）
- **WHEN** 意图仅认领 vlan 路径，对同设备 ifm 路径 POST 配置（无 force）
- **THEN** SHALL 照常接受，SHALL NOT 返回 409
