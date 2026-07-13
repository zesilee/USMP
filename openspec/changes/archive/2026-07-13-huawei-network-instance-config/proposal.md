## Why

BGP 一期（公网 `/bgp:bgp` 进程标量层）已交付（#146/#147），但**所有 BGP peering（peers / address-family / peer-groups）在 `huawei-bgp` 里都是 `augment /ni:network-instance/instances/instance/bgp/base-process` 挂进去的**，`when name='_public_'` 区分公网 vs VPN——**公网邻居即 `network-instance/instance[_public_]/…/peers`，不独立存在**。因此 `huawei-network-instance`（103 行、零未集成依赖）是 BGP 二期一切 peering 的**唯一硬前置**，必须先把它接成 Stack B 可配模块，才能在下一个 change（2a 基础邻居）落地 peer 配置。

## What Changes

- **新增 network-instance 配置面**：接入 `/ni:network-instance`（`huawei-network-instance` 模块顶层独立根容器，`ext:task-name "l3vpn"`）下**全部 config-true（rw）字段，无遗漏**（以 SchemaTree config 继承为权威）——
  - `global` 子容器 3 标量：`cfg-router-id`（ipv4-address-no-zone）、`as-notation-plain`（boolean default false）、`route-distinguisher-auto-ip`（ipv4-address-no-zone）；
  - `instances/instance` list（key=`name`，string 1..31）：config-true 非键字段仅 `description`（string 1..242、pattern `([^?]*)`、`when not(../name='_public_')`）。
  - SchemaTree 入口 `HuaweiNetworkInstance`，结构为**容器根 + 嵌套 list**（root 无直属标量，两个子容器 `global`/`instances`，list 嵌于 `instances` 下）——复用 BGP 引入的容器根编解码（yang-xml-codec XC-05）。**不做字段挑选**，完备性由 schema 驱动用例保证。
- **ygot 生成**：`backend/internal/generated/huawei/gen.conf` 的 `modules` 增加 `huawei-network-instance`，`make gen-yang` 重生成（R04：禁手写、禁改 generated/）。注：`huawei-network-instance` 已作为 BGP augment 闭包的惰性 struct 存在于 generated/；本 change 将其**从"generated-but-not-integrated"提升为首类可配模块**（fakeroot / SchemaTree 根入口 / R04 regen-and-diff 纳管），预期生成差异极小。
- **驱动注册**：`backend/internal/drivers/huawei.go` 新增一条 `driver.Descriptor{Vendor:"huawei", Module:"network-instance"}`，含 `MatchRoute`/`MatchDecode`/`MatchEncode` 谓词（谓词用 `HasPrefix("/ni:network-instance")`）、**显式** `Namespace "urn:huawei:yang:huawei-network-instance"`、SchemaTree 入口闭包。编解码全部走通用引擎 `pkg/yang-runtime/xmlcodec`，**零 XML 代码**。
- **Reconciler**：新增 `internal/controller/networkinstance`，copy BGP 容器根收敛模式（diffEngineAdapter 检出漂移即收敛为单条整根 change，经容器根 xmlcodec 编码为单条 `<network-instance>…`）；`backend/main.go` 注册控制器。
- **模拟网元**：`simulator/netconfsim` 通用 tree datastore 按 local 名存取，预期**无需 network-instance 专用方言**（沿用 BGP 结论），B2 集成验证。
- **完备测试矩阵**：触发 `yang-config-test-design`（T02b），schema 驱动枚举 config-true 标量 + 计数断言 / B2 下发→回读→收敛 / 并发-race / 边界 / 嵌套 list 增删改 / 幂等 / 负路径。
- **明确排除（分期）**：config-false 回读态（`sys-router-id`/`vrf-id`）；BGP peers/afs/peer-groups（下一个 change 2a）；per-VPN feature 模块（evpn/l2vpn/mvpn…，augment network-instance 者，四期+）。
- **删除语义边界**：`_public_` 实例 `ext:generated-by system`（filter `name='_public_'`）设备侧不可删——USMP 侧对 `_public_` 的 instance 节点删除须降级/禁用（沿用 BGP 容器根 node-delete 推迟债，MVP 走 modify 收敛，非 node-delete）。

## Capabilities

### New Capabilities
- `huawei-network-instance-config`: 华为 network-instance（L3VPN 实例，`/ni:network-instance` global + instances/instance 配置面）的模型驱动配置管理——覆盖字段清单、命名空间登记、根 SchemaTree 入口、路由/编码/解码谓词语义、容器根+嵌套 list 编解码复用、`_public_` 不可删语义、分期边界，以及完备测试矩阵要求。

### Modified Capabilities
<!-- 预期不改任何既有 spec 契约：容器根编解码已由 BGP 交付 XC-05；genfix schema 确定性已由 CG-02 覆盖 network-instance 所在的 augment 闭包；network-instance 已在 BGP codegen 闭包内，无新的非确定性。若 apply 期 B2 集成实测"嵌套 list（list 挂于容器根的子容器下）"暴露 XC-05 未覆盖的编解码缺口（第 6 处 list-中心缺口），再补 yang-xml-codec ADDED/MODIFIED delta——沿用 BGP 用往返真值测试暴露缺口的方法。 -->

## Impact

- **代码**：`backend/internal/generated/huawei/gen.conf`（+1 模块名）、`backend/internal/generated/huawei/*`（regen，勿手改）、`backend/internal/drivers/huawei.go`（+1 描述符 + namespace 常量）、新增 `internal/controller/networkinstance/*`、`backend/main.go`（+1 控制器注册）、新增 `*_integration_test.go` + xmlcodec/driver 单测 + hwfix golden；`simulator/netconfsim` 预期不改。
- **依赖闭包**：`huawei-network-instance` import `huawei-extension`（✅ 已集成）+ `ietf-inet-types`（基础类型），**零未集成功能依赖**——DAG 中"唯一硬前置且极易"即此。
- **版本**：`8.20.10/ne40e-x8x16`（gen.conf 现行目标）。
- **前端**：network-instance 配置经通用「模块控制台」由 YANG 模型自动渲染（R05），本期不新增前端硬编码表单。
- **风险（低）**：嵌套 list（`instances/instance` 挂于容器根子容器下）可能暴露 XC-05 容器根编解码在"子容器内含 list"路径上的缺口——BGP 容器根其子容器均为标量集、未走过此路径。由 B2 往返真值测试拦截；若缺口成立则补 yang-xml-codec delta（低概率，因 encodeFields 对 list 字段应已复用 list-center 机制）。
- **合规**：R01/R02/R03/R04/R17、T02b、B2 集成、worktree 隔离、≤500 行/commit、PR ≤3000 行。
