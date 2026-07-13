## ADDED Requirements

### Requirement: BGP-01 公网 BGP 进程配置面接入

系统 SHALL 通过 Stack B 驱动注册表接入华为 `huawei-bgp` 模块顶层独立根容器 `/bgp:bgp` 的**公网 BGP 进程基础配置**（`global` + `base-process` 标量层，含 `confederation`/`graceful-restart`/`timer` 小容器），提供读改下发闭环，SchemaTree 入口为公网根容器（预期 `HuaweiBgp_Bgp`，确切名以 `make gen-yang` 产物为准）。BGP 配置 SHALL 由通用 XML 编解码引擎（`pkg/yang-runtime/xmlcodec`，见 yang-xml-codec XC-01~04）按 ygot 生成数据驱动编解码，SHALL NOT 手写 per-model XML 解析/序列化。ygot 结构体 SHALL 由 `make gen-yang` 自动生成，SHALL NOT 手写、SHALL NOT 手改 `generated/`（R04）。

本期范围 SHALL 覆盖 `/bgp:bgp` 下**全部 config-true（rw）标量 leaf，无遗漏**——以 SchemaTree 的 config 继承为权威判定，共 **29 个**：`global`（yang-enable、memory-overload-exception-discard-route）；`base-process` 直属 13 个（enable、as、keep-all-routes、check-first-as、router-id-auto-select、shutdown、local-ifnet-mtu、private-4byte-as、local-cross-no-med、as-path-limit、dynamic-session-limit、peer-up-route-lowest-priority、delay-time）；config-true 子容器 confederation（as、id、nonstanded）、graceful-restart（enable、peer-reset、restart-time、time-wait-for-rib）、reference-period（clear-interval、hold-interval、suppress-interval）、timer（connect-retry-time、hold-time、keep-alive-time、min-hold-time）。系统 SHALL NOT 做字段挑选式覆盖（商用交付：覆盖部分字段即遗漏），完备性 SHALL 由 schema 驱动的用例（枚举 config-true leaf 对照 fixture）保证。

本期 SHALL NOT 覆盖（各有依据，非简化遗漏）：(a) **config-false 只读态**——`default-parameter`（经 schema 核实为 config false，非早前误判的 rw）、`error-discard-info`、`graceful-restart-status`、`vpn-brief-infos`、`remote-prefix-sid-states`，SHALL NOT 作为下发目标、SHALL NOT 出现在 edit-config；(b) **config-true 列表**——`paf-controls/paf-control`（`/bgp:bgp` 下 `global` 之同级，list）与 `instance-processs/instance-process`（多进程 list），列表编解码本期不接，后续单列。

#### Scenario: 下发公网 BGP 进程配置并回读收敛
- **WHEN** 向模拟网元下发 `/bgp:bgp/base-process`（`enable=true`、`as=<有效AS号>`）
- **THEN** 系统 SHALL 编码为 `huawei-bgp` namespace 的 NETCONF edit-config 报文下发，回读 running config 后 desired↔actual 收敛（无持续漂移）

#### Scenario: 全属性可配（本期字段）
- **WHEN** 依次配置本期覆盖的每一个标量 leaf 与小容器字段
- **THEN** 每个字段 SHALL 能成功编码下发并原值回读，无字段丢失或被静默丢弃

#### Scenario: ygot 生成物零漂移（R04 门禁）
- **WHEN** 在 `gen.conf` 追加 `huawei-bgp` 后运行 `make gen-yang`
- **THEN** 生成物 SHALL 通过 regen-and-diff 门禁（重生成零漂移），且无任何手改 `generated/` 内容

### Requirement: BGP-02 命名空间显式登记

BGP 驱动描述符 SHALL 显式携带模块 XML namespace 常量 `urn:huawei:yang:huawei-bgp`，SHALL NOT 依赖内嵌 gzip schema 的 `Entry.Namespace()` 派生（实测返回空）。编码产出的 BGP 配置报文根元素 SHALL 归属该 namespace。

#### Scenario: 编码报文携带正确 namespace
- **WHEN** 编码一份 `/bgp:bgp/base-process` 配置
- **THEN** 输出 XML 根容器 SHALL 声明 `urn:huawei:yang:huawei-bgp` namespace（前缀不敏感）

### Requirement: BGP-03 路由/编解码谓词精确锚定，不误匹配 feature 模块

BGP 驱动描述符的 `MatchRoute`/`MatchDecode`/`MatchEncode` 谓词 SHALL 精确锚定公网 BGP 根路径（`bgp:bgp`），SHALL NOT 因裸子串 `bgp:` 误命中 feature 模块前缀（`bgp-flow:`、`bgp-evpn:`、`bgp-l2vpnad:` 等）。查找未命中的路径 SHALL 返回 `ok=false` 供调用方降级（R08），SHALL NOT panic。

#### Scenario: 公网 BGP 路径命中
- **WHEN** 以 `/bgp:bgp/base-process/...` 路径触发路由/编解码分发
- **THEN** SHALL 命中 BGP 描述符（`ControllerToken="bgp"`）

#### Scenario: feature 模块前缀负路径（不误命中）
- **WHEN** 以 `bgp-flow:`/`bgp-evpn:` 前缀路径触发分发
- **THEN** SHALL NOT 命中本期 BGP 描述符（本期不接 feature 模块），SHALL 走既有未匹配降级

#### Scenario: 注册可达性
- **WHEN** BGP 集成测试所在二进制/独立测试包运行
- **THEN** 该二进制 SHALL 空白导入 `internal/drivers` 触发注册，`Lookup("huawei", "/bgp:bgp/...")` SHALL 返回 `ok=true`（否则编码落 `xml.Marshal` 兜底对 map 报错）

### Requirement: BGP-04 模拟网元 BGP 方言与端到端集成

`simulator/netconfsim` SHALL 支持 BGP edit-config（整树替换语义，对齐既有 RFC edit-config 通道）与 get-config 回读，支撑 Reconciler↔设备端到端集成测试（B2，`*_integration_test.go`，`testing.Short()` 跳过）。集成测试 SHALL 覆盖下发→回读→收敛全链路。

#### Scenario: 模拟网元接受并回读 BGP 配置
- **WHEN** 集成测试向 netconfsim 下发公网 BGP 配置并随后 get-config
- **THEN** netconfsim SHALL 返回与下发等价的 running config，Reconciler 判定收敛

#### Scenario: 重复下发幂等
- **WHEN** 对同一份 BGP 配置连续下发两次
- **THEN** 第二次 SHALL 判定为 no-op（无 diff、无重复 edit-config 副作用）

### Requirement: BGP-05 完备测试矩阵（yang-config-test-design / T02b）

接入 BGP 到设备配置 SHALL 触发 `yang-config-test-design` 并通过其完备测试矩阵，覆盖：全属性可配、端到端到设备、并发-race、边界、嵌套、幂等、负路径。缺任一层视为未完成，SHALL NOT 合并（T02b/T06）。

#### Scenario: 并发下发无数据竞态
- **WHEN** 多协程并发对 BGP 路径发起 reconcile/编解码
- **THEN** SHALL `-race` 通过，无数据竞态（R09）

#### Scenario: 边界与负路径
- **WHEN** 下发越界 AS 号 / 违反 `must`/`when` 约束的组合（如 `enable=false` 却带 `as`）
- **THEN** 系统 SHALL 拒绝或按 YANG 约束校验失败，SHALL NOT 崩溃（R08），前端 SHALL 展示 YANG 约束提示（R05/§9）

#### Scenario: 下发失败缓存不更新
- **WHEN** netconfsim 模拟 BGP edit-config 失败
- **THEN** 系统 SHALL 保留原配置、缓存不更新，返回明确错误码（§9）

### Requirement: BGP-06 分期边界与前置依赖排序（依赖先交付，禁止越序简化）

本期配置面 SHALL 严格限定为公网 `/bgp:bgp/global` + `base-process` 全 rw 字段，SHALL NOT 包含 `instance-processs/instance-process`（peers、地址族 af）、per-VPN BGP（`augment /ni:network-instance/instances/instance/bgp`）、以及全部 feature 模块（evpn/flow/l2vpnad/link-state/lsp/mdt/mvpn/srpolicy/srv6/vpntarget 等）。这些能力 SHALL 由后续独立 change 分期接入。

后续分期依赖关系经全 augment 树核验（见 design.md），SHALL 遵循以下 DAG、先交付依赖再启动，SHALL NOT 在 leafref 目标模型未集成时以简化方式先接入对应属性（否则 `require-instance` 致设备侧非法）：
- **结构约束**：peers/afs/peer-groups 均在 `augment /ni:.../instance/bgp` 下（公网即 `instance[_public_]`），故任何 peering SHALL 以 `huawei-network-instance` 集成为唯一硬前置。
- **二期-2a（基础邻居：address/remote-as/af-type 及非策略字段）** 依赖 SHALL 仅为 `network-instance`；跨模型策略引用均为可选 leaf，2a SHALL 完整覆盖强制+非策略字段而不阻塞于策略模型。
- **二期-2b（可选策略属性）** 各属性 SHALL 以其 leafref 目标模型（`tunnel-management`/`xpl`/`routing-policy`/`acl`）集成为前置。
- **`routing`/`bfd`/`ethernet`** SHALL NOT 列为硬依赖：BGP 仅一处软 `must` 引用 routing，仅需 codegen-present。

本期（公网 base-process）功能性配置依赖为零，SHALL 可独立完整交付。

#### Scenario: 分期范围显式
- **WHEN** 审阅本 change 交付的配置面
- **THEN** peers/地址族/per-VPN/feature 模块 SHALL NOT 出现在本期驱动描述符的路由/编解码覆盖与测试矩阵中；分期路线与前置依赖 DAG SHALL 在文档中登记

#### Scenario: peering 硬前置门禁（负路径）
- **WHEN** 计划启动任何 peer/AF 接入（二期-2a）而 `huawei-network-instance` 尚未集成
- **THEN** SHALL NOT 启动；SHALL 先集成 network-instance（peers/AF 的 augment 根）

#### Scenario: 2b 策略属性越序门禁（负路径，防越序简化）
- **WHEN** 计划为 peer/AF 接入某策略属性（route-policy/route-filter/acl/tunnel）而其 leafref 目标模型未作为可配模型集成
- **THEN** 该属性 SHALL NOT 接入；SHALL 先交付目标模型（tunnel-management/xpl/routing-policy/acl 各自独立 change）

#### Scenario: config-false 只读态不被误下发
- **WHEN** 编码本期 BGP 配置
- **THEN** `base-process` 下 config-false 态（vpn-brief-infos 等）SHALL NOT 出现在 edit-config 下发报文中
