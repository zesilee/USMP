# device-protocol — delta（snd-xml-codec）

## MODIFIED Requirements

### Requirement: DP-07 NETCONF 删除变更编码（operation=delete 键式条目）

NETCONF 客户端对 `DeleteChange` SHALL 经驱动描述符注册表按 `OldValue` 的 GoStruct 类型解析编解码数据，由通用删除编码器（yang-xml-codec XC-03，主键经 ygot `ΛListKeyMap()` 获取）构造删除 edit-config：外层包裹模型容器，条目元素携带 NETCONF base 命名空间的 `operation="delete"` 属性，且 SHALL 仅序列化 key 叶（vlan→`id`、ifm→`name`，且 key 为首个子元素以对齐 RFC 键匹配惯例）；SHALL NOT 以 per-model 类型 switch 硬编码 key 名。编码 SHALL 经既有 `Set`（candidate→commit 两阶段，DP-04）下发；`<rpc-error>`（如 data-missing）SHALL 反映为该变更失败。未注册/未知模型类型 SHALL 返回明确编码错误（R08 不 panic）。

#### Scenario: VLAN 条目删除编码

- **WHEN** DeleteChange 的 OldValue 为仅含键 10 的 HuaweiVlan 条目集
- **THEN** 编码 SHALL 形如 `<vlans><vlan nc:operation="delete" …><id>10</id></vlan></vlans>`（operation 属性带 NETCONF base 命名空间前缀）
- **AND** SHALL NOT 序列化 key 以外的叶

#### Scenario: IFM 接口删除编码

- **WHEN** DeleteChange 的 OldValue 为仅含 name 的 HuaweiIfm 接口集
- **THEN** 编码 SHALL 在 `<interface>` 元素上携带 operation="delete" 且仅含 `<name>`

#### Scenario: 设备拒绝透出（负路径）

- **WHEN** 设备对删除返回 `<rpc-error>`（data-missing）
- **THEN** 该变更 SHALL 标记失败并携带错误，SHALL NOT 吞错

#### Scenario: 未知模型类型（负路径）

- **WHEN** DeleteChange 的 OldValue 非任何已注册模型
- **THEN** SHALL 返回编码错误，SHALL NOT 发送裸 delete 元素
