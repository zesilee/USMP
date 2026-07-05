# converge-crd-tree-bizv1 — design（D1 收敛到 api/biz/v1）

> change：`converge-crd-tree-bizv1` | 依赖：`proposal.md`

## Context

`api/v1`（旧）与 `api/biz/v1`（有生成 CRD YAML=部署 schema）Spec 不兼容。translator/crdsource 绑 api/v1，与部署的 biz/v1 解码错位（D1）。权威 = biz/v1（用户实际创建的 schema）。

## Goals / Non-Goals

**Goals:** translator + crdsource 消费 biz/v1；字段映射对齐 biz/v1↔huawei ygot；删 api/v1，解 D1。
**Non-Goals:** 物理删 actor（pr-size 债）；gNMI/plugin（D5/D3）；不改 biz/v1 CRD 定义。

## Decisions

### D-1 biz/v1 为权威
- 部署的 CRD YAML 出自 biz/v1，用户能设的即 biz/v1 字段。translator 读 api/v1 = 读用户设不了的字段（错位）。故 translator 迁 biz/v1。

### D-2 字段映射（biz/v1 → huawei ygot）
- VLAN：`VlanID→Id`（+key）、`Name`、`Description`、`AdminStatus`(up/down→1/2)、`MacLearning`(enabled/disabled→1/2)、`BroadcastDiscard`(true→1)、`UnknownMulticastDiscard`(true→1)。huawei ygot 均有同名字段。
- Interface：`IfName→Name`、`Description`、`AdminStatus`、`MTU`；biz/v1 模式仅 access/trunk/hybrid → 均 L2（`ServiceType=2`、`IsL2Switch=true`）。移除 L3/IpAddress 分支（biz/v1 无）。
- Route：map 用 `Destination/NextHop/Preference/Description/BfdEnabled`（stub，后续 ygot 化留 D8+）。

### D-3 原子迁移
- translator 包 4 文件共享 `bizv1` 导入 → 必须同 PR 全改（否则同名类型指向不同包，编译裂）。crdsource 随之。删 api/v1 为后续 PR（迁移后零引用）。

## Risks / Trade-offs

- **能力对齐部署 schema**：移除 api/v1-only 的 L3 接口/VLAN Type/端口成员/Speed/Duplex 映射。这些在部署的 biz/v1 CRD 里本不可设 → 无实际用户可用能力损失；反而修复 biz/v1 字段（广播丢弃/MAC 学习）此前被忽略的 bug。
- **无既有 translator 测试**：先补 TDD 单测锁定新映射，再改，避免回归。
- **Route/System 仍 stub**：本次只迁移字段引用不改 stub 本质（Route 返回 map，非 ygot）；ygot 化留后续。
- **reconcile 集成不受影响**：vlan/ifm reconciler 集成测试经 ConfigStore 直接喂 huawei ygot，不经 translator；本次不动它们。
