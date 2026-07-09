# config-api delta — config-delete-semantics

## ADDED Requirements

### Requirement: BR-09 行删除通道（DELETE，命令语义）

`DELETE /api/v1/config/:ip/*path?key=<条目主键>` SHALL 以命令语义删除列表单条目：按 path 路由模型分支解析 key → 从 ConfigStore desired 移除该键（与合并写同临界区，幂等）→ 构造仅含 key 叶、带 `operation="delete"` 的 edit-config **同步下发**（candidate→commit）→ 成功后失效该设备运行缓存、记录审计、触发对账验证收敛。下发失败（含设备 data-missing）SHALL 原样透出错误、SHALL NOT 失效缓存或写审计。POST 合并语义与 `walkMap` merge/subset 收敛语义 SHALL NOT 因删除通道改变。

#### Scenario: 删除成功端到端

- **WHEN** 设备存在 vlan 10，请求 `DELETE /config/<ip>/vlan:vlan/vlan:vlans?key=10`
- **THEN** SHALL 返回成功，回读 SHALL 不再含 vlan 10
- **AND** desired SHALL 已移除键 10，二轮对账 SHALL 0 change（不复活、不漂移）

#### Scenario: 删除不存在条目（负路径）

- **WHEN** 请求删除设备上不存在的条目键
- **THEN** SHALL 透出设备错误（如 data-missing），SHALL NOT 失效缓存

#### Scenario: 非法 key / 未知路径（负路径）

- **WHEN** key 缺失、类型不合模型（如 vlan key 非整数）、或 path 无模型分支
- **THEN** SHALL 返回 `code=400`，SHALL NOT 触达设备

#### Scenario: 并发删除（R09）

- **WHEN** 并发对同一 path 发起删除与合并下发
- **THEN** desired 读改写 SHALL 串行化，SHALL NOT 丢更新或竞态（-race 全绿）

### Requirement: BR-10 删除的模型驱动门禁

DELETE SHALL 在触达设备前按 schema 校验：目标 list 节点 `operation-exclude` 含 `delete`、或目标节点为 readonly（config false）时 SHALL 拒绝（400 + 明确错误），与前端按钮门禁互为防御。schema 查不到该路径时 SHALL 放行（降级，R08——设备/转换层为最终权威）。

#### Scenario: operation-exclude 拒绝

- **WHEN** 目标 list 在模型中标注 `ext:operation-exclude` 含 delete
- **THEN** SHALL 返回 400 且 SHALL NOT 触达设备

#### Scenario: readonly 路径拒绝

- **WHEN** 目标路径为 `config false` state 子树
- **THEN** SHALL 返回 400 且 SHALL NOT 触达设备

#### Scenario: schema 未覆盖路径降级放行（边界）

- **WHEN** 路径在 schema 中查不到对应节点
- **THEN** 门禁 SHALL 放行，由模型分支路由/设备决定成败
