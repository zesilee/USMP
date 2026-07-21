# netconf-simulator delta — netconf-get-state-read

## ADDED Requirements

### Requirement: NS-08 `<get>` RPC 与状态数据 overlay

模拟器 SHALL 实现 NETCONF `<get>` RPC（RFC6241 §7.7）：返回 running 配置树与状态数据 overlay 树的**合并结果**（状态叶挂接到配置树对应节点下，同名节点以状态树为准），并 SHALL 支持与 get-config 相同的 subtree filter 语义。状态数据 overlay SHALL 经 `SetStateDataXML` 注入、独立于 running/candidate 数据存——edit-config/commit/discard-changes SHALL NOT 影响状态树。`<get-config>` 行为 SHALL 保持不变（仅返回配置树，不含状态数据）。模拟器 SHALL 内置 `DemoStateSeed` 演示状态种子（IFM 接口 `dynamic` 容器，设备侧数字枚举形态；demo 配置无 VLAN 故 VLAN 状态场景由集成测试覆盖），standalone 二进制启动时 SHALL 注入。

#### Scenario: get 返回配置+状态合并树
- **WHEN** 经 `SetStateDataXML` 注入接口 `dynamic` 状态子树后，客户端发送携 subtree filter 的 `<get>`
- **THEN** 模拟器 SHALL 返回配置叶与状态叶合并的子树 XML，状态叶（如 `oper-status`）挂接在对应 `interface` 条目下

#### Scenario: 未注入状态时 get 等值 get-config
- **WHEN** 未调用 `SetStateDataXML`，客户端发送 `<get>` 与 `<get-config source=running>` 同一 filter
- **THEN** 两者 SHALL 返回等值数据

#### Scenario: get-config 不含状态数据
- **WHEN** 已注入状态 overlay，客户端发送 `<get-config source=running>`
- **THEN** 返回 SHALL 仅含配置树，SHALL NOT 出现状态叶

#### Scenario: edit-config 不触碰状态树
- **WHEN** 已注入状态 overlay，客户端 edit-config 修改配置并 commit
- **THEN** 后续 `<get>` SHALL 返回更新后配置与**原状态数据**的合并结果，状态树不受写操作影响

#### Scenario: 并发 get 与写操作安全
- **WHEN** 多客户端并发发送 `<get>` 与 edit-config/commit
- **THEN** 模拟器 SHALL NOT 出现数据竞态（`-race` 干净，R09），各响应均为一致快照
