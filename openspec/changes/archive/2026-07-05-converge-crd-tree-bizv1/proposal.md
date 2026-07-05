## Why

迁移债 **D1**：`api/v1` 与 `api/biz/v1` 都注册 `biz.usmp.io/v1`，但 Spec 结构不兼容——而**部署的 CRD YAML 出自 `api/biz/v1`**（用户实际创建的就是这套），translator/crdsource 却绑 `api/v1` 解码。结果：用户能设的 biz/v1 字段（BroadcastDiscard/MacLearning/UnknownMulticastDiscard、L2 接口模式）被当前解码**静默忽略**，而 translator 读的 api/v1 字段（Type/TaggedPorts/L3/IpAddress）在部署 schema 里根本不存在——即 CRD 意图源解码错位。收敛到 `api/biz/v1`（权威 = 部署的 schema）修复此错位，消除 D1。

## What Changes

- **translator 迁到 `api/biz/v1`**：`huawei_vlan.go`/`huawei_interface.go`/`huawei.go` 的 `bizv1` 导入从 `api/v1` 改为 `api/biz/v1`，重写字段映射：
  - VLAN：`VlanID/Name/Description/AdminStatus` 不变；新增 `MacLearning`(enum)/`BroadcastDiscard`(bool)/`UnknownMulticastDiscard`(bool) → huawei ygot 同名字段（1:1）；**移除** `Type`/`MacLearningEnabled`/`StatisticEnabled`/`BroadcastDiscardEnabled`/`TaggedPorts`（biz/v1 无，部署 schema 不可设）。
  - Interface：`InterfaceName→IfName`、`TrunkAllowedVlans→TrunkVlans`([]uint16)；**移除** L3/IpAddress/Netmask/NativeVlan/Speed/Duplex 逻辑（biz/v1 为 L2-only：access/trunk/hybrid，无 L2/L3 模式）。
  - Route：map 用 biz/v1 字段（`Destination/NextHop/Preference/Description/BfdEnabled`）。
- **crdsource 迁到 `api/biz/v1`**：`VlanProjectFunc`/`InterfaceProjectFunc`/`register.go`（`AddToScheme`、`VlanObject`/`InterfaceObject`）+ 测试。
- **删除 `api/v1`**（6 文件；其 `NativeDeviceConfig` 类型死代码，真身在 `api/core/v1`）。
- **BREAKING（内部）**：翻译能力对齐部署 schema（L3 接口/VLAN Type/端口成员等 api/v1-only 能力移除——它们在部署的 biz/v1 CRD 里本就不可设，无实际损失）。CRD 用户接口（biz/v1）不变。

## Capabilities

### Modified Capabilities
- `translation-engine`: translator 消费 `api/biz/v1`（部署 schema），字段映射对齐；VLAN 广播/组播丢弃 + MAC 学习经 huawei ygot 生效；Interface 收敛为 L2 模型。
- `business-crd`: 统一 CRD 树到 `api/biz/v1`+`api/core/v1`，退役 `api/v1`（解 D1）。

## Impact

- **后端**：`pkg/translator/{huawei_vlan,huawei_interface,huawei}.go`、`internal/crdsource/*`、删 `api/v1/*`。
- **测试**：新增 translator 单测（VLAN/Interface biz/v1 字段→huawei ygot）；更新 crdsource 测试为 biz/v1；全量 `go test ./...` + `go build ./...` 绿。
- **红线**：R04（ygot desired，映射对齐生成模型）、R06（TDD）。
- **不在范围**：物理删 `actor` 包（pr-size 债）；gNMI/plugin 空转（D5/D3）。
- **迁移策略**：translator+crdsource 原子迁移（同包 `bizv1` 导入不可拆）+ 测试为一 PR；`api/v1` 删除为随后 PR（迁移后无引用）。
