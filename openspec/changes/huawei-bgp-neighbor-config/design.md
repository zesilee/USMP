## Context

network-instance 接入时确立（ni design D1）：`/ni:network-instance` 因 augment 合并是**单一 ygot 根** `HuaweiNetworkInstance_NetworkInstance`，其生成物 `Instance` 结构已带 `Bgp`（module huawei-bgp）augment 子树；未来 peering「扩展同一描述符驱动面，非另立描述符」。XC-06 交付后，根下 huawei-bgp 子树编码带正确 namespace。故 2a 的链路（描述符路由/编解码/reconciler 收敛/namespace）**全部已就位**，本 change 是把 peer/af 字段纳入 desired 并证明端到端。

peer 结构（8.20.10）：`peers/peer`（key=`address`，mandatory `remote-as`）含 27 config-true 标量 + 9 子容器（timer/graceful-restart/bfd-parameter 属 2a；fake-as-parameter/egress-engineer-parameter/`*-state` 状态容器不属）+ `afs/af`（key=`af-type` 枚举）。深度：`instance/bgp/base-process/peers/peer/afs/af`——多层容器 + list-under-list + 枚举 key。

约束：R04、零回归（ni/vlan/ifm/bgp golden）、TDD、B2 必过、per-node namespace 真值断言（sim 测不出）。

## Goals / Non-Goals

**Goals:**
- 公网 peer config-true 标量（27）+ timer/graceful-restart/bfd-parameter + afs/af af-type 的读改下发闭环，复用 ni 描述符 + XC-06。
- 完备测试矩阵：schema 驱动枚举 huawei-bgp peer config-true 标量 + 计数断言、B2 端到端（含 namespace 真值）、并发、边界、嵌套、幂等、负路径。

**Non-Goals:**
- af 内策略属性（route-policy/filter/acl/tunnel）——2b，门控未集成模型。
- peer-groups/dynamic-peer-prefixes/egress-engineer/fake-as-parameter、config-false 状态、per-VPN（非 _public_）peers。
- 新描述符/新控制器/新 namespace 引擎（全已就位）。

## Decisions

**D1：零新描述符/控制器——复用 network-instance 链路。**
peer 路径 `HasPrefix("/ni:network-instance")` 已命中 ni 描述符；生成物 `Instance.Bgp.BaseProcess.Peers.Peer` 已存在；ni reconciler 整根收敛整棵 `/ni:network-instance`；XC-06 让 `<bgp>` 带 huawei-bgp namespace。故 2a desired 填 peer 字段即经既有链路下发。**本 change 不动 drivers/controller 生产代码**（除非缺口）。
- 备选：为 peers 单立描述符——否决（ni design D1 已定：单一 ygot 根不可按前缀拆）。

**D2：完备枚举锚定 peer 子树、按 huawei-bgp module 计数。**
schema 驱动枚举 `Peer` 结构（经 SchemaTree 定位 `…/peers/peer`）的 config-true 标量，赋值→编码（经 ni Spec，带 XC-06 namespace）→解码→DeepEqual + 计数断言。基础子容器（timer/graceful-restart/bfd-parameter）递归纳入；策略子容器/状态容器（config-false）/其他 2b 子容器排除（按 schema config 继承 + 明确跳过清单，非人工挑字段）。计数在 apply 期由 schema 实测锁定（防"全属性可配"漏字段）。

**D3：正确性防线含 per-node namespace 真值。**
B2 集成 + 编码断言须验证下发 XML 中 `<bgp xmlns="urn:huawei:yang:huawei-bgp">`、`<peer>`/`<afs>`/`<af>` 继承之、`<address>`/`<remote-as>` 真值正确——sim/decode namespace-宽容，故断言 encode 输出 namespace（XC-06 方法论延续）。

**D4：af-type 枚举 key 编解码验证。**
`afs/af` key 是 `E_HuaweiBgp_AfType` 枚举——list-under-list + 枚举 key 是 ni（string key）未走过的路径。用往返真值证明枚举 key 正确编解码（若暴露缺口补 xmlcodec delta）。

**D5：删除语义沿用平台契约。**
peer/af 删除经声明式 subset 天然不删（同 NI-06），设备侧删除走 DELETE 命令通道推迟债。

## Risks / Trade-offs

- **[深层嵌套/枚举 key 缺口]** peer→afs→af 多层 + 枚举 key 是新路径，可能暴露 xmlcodec 缺口。→ **Mitigation**：往返真值 + B2 拦截；TDD 红灯先行；若成立补 yang-xml-codec delta（如 ni 一样实测判定，不臆断）。
- **[per-node namespace 深层]** `<bgp>` 之下若再有跨模块 augment（af 内 feature 模块）→ 多层边界。→ 本期 af 只到 af-type（huawei-bgp），无更深跨模块；XC-06 递归天然支持，2b 接 feature 时验证。
- **[完备枚举误纳 2b/config-false 字段]** → 按 schema config 继承 + 明确排除清单，计数断言 + 负路径（策略/状态字段不出现在下发）双重锁定。
- **[mandatory remote-as]** peer 无 remote-as 时 ΛValidate 应拒绝 → 边界用例覆盖。

## Migration Plan

worktree TDD：红灯（peer 完备枚举计数 + namespace 真值 + af-type 枚举 key 往返）→ 若缺口补 xmlcodec → B2 集成（ni+_public_+peers 收敛、namespace 真值）→ 并发/边界/负路径 → review → commit（预期测试为主，≤500 行/commit 拆分）。回滚=删测试（若无生产改动）或还原 xmlcodec 增量。

## Open Questions

- peer 完备枚举精确计数——apply 期 schema 实测锁定。
- 是否暴露枚举 key/深层嵌套 xmlcodec 缺口——B2/往返实测判定，不臆断。
