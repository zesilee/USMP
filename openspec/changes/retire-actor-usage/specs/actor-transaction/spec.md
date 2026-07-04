## MODIFIED Requirements

### Requirement: Actor 子系统生产使用清零（R01/D2）

系统 SHALL NOT 在任何生产或框架路径中使用 `pkg/yang-runtime/actor`（Actor/2PC 模型违反 R01）。设备连接/探活 SHALL 经 `ClientPool` 直连，而非借用 Actor。`pkg/yang-runtime/actor` 物理文件在其单文件超 pr-size 上限时 MAY 暂留（无生产使用即不再违规），并作机械清理债后续分批删除。

#### Scenario: 无生产代码引用 Actor
- **WHEN** 审计 `pkg/yang-runtime/actor` 的非测试导入方
- **THEN** SHALL 为空（BusinessSwitch 探活改 ClientPool、死码 `vlan/actor_reconciler.go` 已删）

#### Scenario: 设备探活经 ClientPool
- **WHEN** BusinessSwitch reconcile 探测设备
- **THEN** SHALL 用 `ClientPool.Get` 建连并以 `IsConnected()` 判定在线，连接失败降级为离线（R08），SHALL NOT 创建 Actor
