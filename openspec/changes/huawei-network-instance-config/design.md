## Context

BGP 一期交付了公网 `/bgp:bgp` 进程标量层，并顺带把「容器根编解码」（yang-xml-codec XC-05）与「多-augment 闭包 schema 确定性」（yang-codegen-pipeline CG-02）两处通用引擎缺口补齐。BGP 二期要落地 peer/AF，但**华为把 peers/afs/peer-groups 全部 `augment` 到 `/ni:network-instance/instances/instance/bgp/base-process` 下**（bgp.yang:3207/797/2059），`when name='_public_'` 区分公网 vs VPN。因此 `huawei-network-instance` 是任何 peering 的**唯一硬前置**，本 change 先把它接成 Stack B 可配模块。

现状（已核实 generated/all.gen.go）：`huawei-network-instance` 已作为 BGP augment 闭包的惰性 struct 存在——`HuaweiNetworkInstance_NetworkInstance`（容器根）已生成，含 `Global` 与 `Instances` 两个子容器；`Instances_Instance` list 已生成。**关键发现**：生成的 `instance` struct 是**多模块 augment 的共享合并点**——除原生 `Name`/`Description`/`SysRouterId`/`VrfId`（module `huawei-network-instance`）外，还带 `Bgp`（module `huawei-bgp`）、`Afs`/`Parameter`/`TrafficStatisticEnable`（module `huawei-l3vpn`）等 augment 字段。故 `/ni:network-instance` 是**单一 ygot 根**，未来 BGP-2a 的 peers 是往这同一根里填 `Instance.Bgp` 子树，而非另立描述符。

约束：R01（分层不变）、R04（禁手写/禁改 generated/）、R17（spec-first）、T02b（yang-config-test-design 完备矩阵）、B2 集成必过、worktree 隔离、≤500 行/commit。

## Goals / Non-Goals

**Goals:**
- 接入 `/ni:network-instance` 下**原生 config-true 字段**的读改下发闭环：`global`{cfg-router-id, as-notation-plain, route-distinguisher-auto-ip} + `instances/instance`{name(key), description}。
- 注册单条 `driver.Descriptor`（route `/ni:network-instance`、entry `HuaweiNetworkInstance_NetworkInstance`、namespace `urn:huawei:yang:huawei-network-instance`），编解码零 XML 代码，复用容器根 XC-05。
- 新增 `internal/controller/networkinstance` reconciler（copy BGP 容器根收敛）+ main.go 注册。
- `yang-config-test-design` 完备矩阵，**schema 驱动枚举按 module tag 过滤到原生字段** + 计数断言。

**Non-Goals:**
- config-false 回读态（`sys-router-id`/`vrf-id`）——不纳入配置面。
- augment 子树（`Instance.Bgp` peers/AF = 下一个 change 2a；`Afs`/`Parameter` l3vpn = 更后期）——本期结构上存在但**不驱动、不暴露、不测**。
- 容器根 node-delete（`_public_` 不可删；一般 instance 删除走 MVP modify 收敛，node-delete 沿用 BGP 推迟债）。
- 前端硬编码表单——通用模块控制台按 YANG 自动渲染（R05）。

## Decisions

**D1：单一描述符覆盖整棵 `/ni:network-instance`，本期只驱动原生字段。**
`/ni:network-instance` 是单一 ygot 根 `HuaweiNetworkInstance_NetworkInstance`；因 augment 合并，peers（huawei-bgp）与 l3vpn 字段结构上同属此根。若按路径前缀拆成多描述符会重叠冲突（peers 路径 `/ni:.../instance/bgp/...` 也 `HasPrefix("/ni:network-instance")`）。故**注册一条描述符覆盖整棵子树**，route/decode/encode 谓词均 `HasPrefix("/ni:network-instance")`。本期 reconciler 只 set 原生字段（Global + instance Name/Description），augment 字段保持 nil → 不被编码。未来 2a **扩展同一描述符的驱动面**（填 `Instance.Bgp`），非新增描述符。
- 备选：拆两条描述符（ni-native / bgp-peers）——否决，路径前缀重叠、ygot 根不可分割。

**D2：完备性枚举按 `module` tag 过滤到 `huawei-network-instance`。**
BGP 完备测试枚举 entry 下每个 config-true 标量并计数断言。network-instance 若裸枚举 `HuaweiNetworkInstance_NetworkInstance` 会递归进 `Bgp`（huawei-bgp）与 `Afs`/`Parameter`（huawei-l3vpn）augment 子树（数百 leaf），越出本期范围。故完备枚举**过滤 `module:"huawei-network-instance"` 标签**，得原生 config-true 标量恰 **5** 个：`global`{as-notation-plain, cfg-router-id, route-distinguisher-auto-ip} + `instance`{name, description}（`sys-router-id`/`vrf-id` 为 config-false 排除）。计数断言=5，模型加原生字段即触发复审。
- 备选：枚举全子树——否决，把未集成的 augment 面纳入，违背分期与依赖 DAG。

**D3：reconciler copy BGP 容器根收敛模式。**
`internal/controller/networkinstance` 对齐 `internal/controller/bgp`：`diffEngineAdapter.Diff` 检出任一漂移即收敛为**单条整根 change**（下发整个 desired `/ni:network-instance`，经描述符容器根 xmlcodec 编码为单条 `<network-instance>…`，edit-config merge 收敛）。规避 `diff.walkStruct` 对容器根每个顶层子容器各发一条 change、落 `xml.Marshal` 兜底发 Go 类型名的漂移 bug（BGP 组 5 已验证根因）。最小侵入放本 adapter，不动共享 diff/client（VLAN/IFM list 路径零回归）。main.go 注册控制器。

**D4：namespace 单值 `urn:huawei:yang:huawei-network-instance`（原生权威）。**
沿用 BGP 决策：取 8.20.10 YANG 声明的 module namespace，谓词 `HasPrefix("/ni:network-instance")` 精确锚定。本期只发原生字段，单 namespace 足够。

**D5：gen.conf 加 `huawei-network-instance`，regen 预期近零漂移。**
type 已在 BGP 闭包内生成；加入 `modules` 主要产出 fakeroot/SchemaTree 根入口的稳定化与 R04 纳管。`make gen-yang` 后经 CG-02 确定性规范化，`regen-and-diff` 须零漂移方合规。

**D5b：「只做 config-true」收窄的是驱动/测试面，非生成/解码面——不影响任何跨模块依赖。**
澄清三层：**① codegen** 生成**完整** struct（`SysRouterId`/`VrfId` config-false 字段仍在生成物内，schema/类型对其它模块完整可解析）；**② xmlcodec 引擎不按 config 属性过滤**（`encode.go:159`「populated means pushed」，华为刻意把在发字段标 config-false，过滤反破坏行为等价 → config-false 照样可编可解、回读态解得出）；**③ 驱动面/完备测试**才收窄到 config-true（reconciler 只 set、完备枚举按 module tag 计数=5）。故其它模块对本模块的依赖不受影响，且有两条硬保证：(1) YANG 结构禁止 config-true `leafref`（require-instance 默认真）指向 config-false 目标——跨模型 leafref 只能引用我们的 config-true（如 BGP peers 引用 `instance/name` key）；(2) 即便是 state→state 回读依赖（如 BGP 一期 `vpn-brief-infos`，config-false→config-false），因①②全保真，回读照样解得出。将来若需前端展示 VRF 运行态（`sys-router-id`/`vrf-id`）= 纯增量只读特性、字段已就绪，不被本 change 阻挡。
- 备选：codegen 侧裁掉 config-false——否决，破坏 schema 完整性、破坏在发 config-false 字段行为等价、且引擎本就不过滤。

**D6：instance 删除 = 声明式 subset 语义（天然不删）+ DELETE 命令通道（推迟债）。**
apply 期实测确认（zz_probe）：从 desired 移除某 instance 后对账 `Changes==0`、设备保留该条目、无永久漂移——`DefaultDiffEngine` 的 `walkMap` 是 **subset 语义**，actual 中多出的条目不算漂移。这是**平台既有契约**（`config-delete-semantics` memory：声明式通道刻意删不了，删除走独立 DELETE 命令通道），VLAN/IFM/BGP 一致，非 network-instance 缺陷。故本 MVP：增/改经声明式对账收敛；instance 删除（含 `_public_` `ext:generated-by system` 不可删守卫）须走 DELETE 命令通道，**本期不接、登记推迟债**（沿用 BGP 容器根 node-delete 债）。`_public_` 在声明式通道下天然永不被删，与设备侧一致。用集成测试锁死此契约（remove-from-desired 是 no-op），防未来误判为 bug。

## Risks / Trade-offs

- **[嵌套 list 编解码缺口]** `instances`（容器根子容器）内含 `instance` list——BGP 容器根其子容器均为纯标量集，XC-05 从未走过「子容器内含 list」路径。→ **Mitigation**：B2 往返真值测试（RFC7951→XML 带 namespace→回读 DeepEqual，断真值非仅非空）拦截；若 `encodeContainer/encodeFields` 对嵌套 map 字段编码失败，补 yang-xml-codec ADDED/MODIFIED delta（第 6 处 list-中心缺口）。低概率（encodeFields 对 list 字段应复用 list-center 机制），但必须实测证实、不臆断。
- **[augment 字段误入配置面]** 单描述符覆盖整棵子树，若 API/前端或 diff 误将 `Bgp`/`Afs`/`Parameter` 纳入本期驱动，会越序触达未集成模型。→ **Mitigation**：reconciler 只 set 原生字段；完备测试按 module tag 过滤并计数断言=5；负路径用例断言未 set 的 augment 字段不出现在编码报文。
- **[per-node namespace 债]** 未来 2a 往 `Instance.Bgp` 填 peers 时，augment 节点须携带各自 module 的 namespace（huawei-bgp），而非本期单一 ni namespace。→ 本期不触发（只发原生字段）；登记为 2a 前置：xmlcodec 需按 `module` tag 派生 per-node namespace（若尚不支持）。
- **[生成物漂移放大]** 加 `huawei-network-instance` 到 modules 若触发 CG-02 未覆盖的无序集合重排。→ 低概率（network-instance 已在 BGP 闭包内、CG-02 已处理该闭包）；regen-and-diff 门禁兜底。

## Migration Plan

worktree 内按 tasks.md：gen.conf → make gen-yang（验证近零漂移）→ 描述符 + 单测 → reconciler + main.go 注册 → B2 集成（拦嵌套 list）→ 完备矩阵 → review → commit。回滚=还原 gen.conf 一行 + 摘描述符 + 删 controller 目录；network-instance type 保留为惰性 struct（BGP 闭包仍需），无破坏。

## Open Questions

- 嵌套 list 是否触发 XC-05 缺口——由 B2 实测判定（不臆断，spike 优先）。
- API/模块控制台是否需为「容器根 + 单实例 global + list」形态额外适配——由通用渲染现状决定，超出本期则登记债、不在本 change 扩面。
