## Why

BGP 二期-2a 交付了公网邻居基础配置，但 peer/AF 的**可选策略属性**（route-policy/route-filter/ACL group/tunnel-policy）被显式推迟到 2b，门控于其 leafref 目标模型的集成——`require-instance` 默认为真，目标实例不存在时下发即设备侧非法（2a proposal「明确排除」+ public-config design.md「越序禁令」）。

`tunnel-policy`（BGP AF 属性之一，`common:*` leafref → `/tunnel-management/.../tunnel-policy`）的目标模型 `huawei-tunnel-management` 是 2b 依赖 DAG 的**叶子**（唯一功能依赖 `ifm` 已集成✅，见 public-config design.md「依赖分析」矩阵）。先把它接成 USMP 可配模型，`tunnel-policy` 实例按 name 存在，才能在后续波次合法接入 BGP `tunnel-policy` 属性；同时它也是 `routing-policy`（2b 波次③）的前置。故本 change 是 2b 的第一块地基。

## What Changes

- **接入公网 tunnel-management 配置面**：根容器 `/tunnel-management`（`huawei-tunnel-management` 模块顶层独立容器根，namespace `urn:huawei:yang:huawei-tunnel-management`，结构与 `/bgp:bgp` 同构——容器根非 list 根，走通用引擎 XC-05 plain-container 路径）。本波次接入**标量边界**的全部 config-true leaf：
  - `tunnel-policys/tunnel-policy`（key=`name`，string 1..39）的 `name` + `description`（string 1..80）
  - `tunnel-down-switch/enable`（boolean）
  - **此边界已完整满足 BGP `tunnel-policy` leafref**：目标 list 实例按 `name` 存在即可解析，BGP 引用不触及 policy 内部 tunnel 选路细节。
- **零 codegen**：`HuaweiTunnelManagement_TunnelManagement` 及子树（60 类型）已在一期 huawei-bgp 全闭包生成时"免费"生成（public-config design.md「副产物洞察」）。本 change **不改 `gen.conf`、不 regen、不 touch generated/**（R04 regen-and-diff 不涉及）。
- **驱动注册**：`backend/internal/drivers/huawei.go` 新增一条 `driver.Descriptor{Vendor:"huawei", Module:"tunnel-management"}`——`MatchRoute`/`MatchDecode`/`MatchEncode` 谓词 + **显式** `Namespace` 常量 + SchemaTree 入口闭包（`HuaweiTunnelManagement_TunnelManagement`）。编解码全走通用引擎 `pkg/yang-runtime/xmlcodec`，**零 XML 代码**。
- **模拟网元方言**：`simulator/netconfsim` 增加 tunnel-management edit-config/get-config 方言，支撑 B2 端到端集成测试。
- **完备测试矩阵**：触发 `yang-config-test-design`（T02b），覆盖全（标量边界）属性可配 / 端到端到设备 / 并发-race / 边界 / 幂等 / 负路径 / 删除语义。
- **明确排除（分期，注册为 follow-up，非简化）**：`tunnel-policy` 下 `ipv4-set`/`ipv6-set` 深层子树——`choice policy-type`（select-sequences/binding）、presence 容器 `select-sequence`、mandatory `loadbalance`/`bind-type`、`ordered-by user` 的 `select-tunnel-type`、`nexthops/nexthop`(key=address) 嵌套 binding、`tunnel-names/tunnel-name`(leafref→ifm interface, must type=Tunnel)、`auto-names`。这些门控于通用引擎对 choice 拍平 / presence / ordered-by / 深层嵌套 list 往返的支持（可能暴露新 XC 缺口，按 TDD 逐个补 delta），拆到本波次-follow-up。config-false 子树 `tunnel-infos`/`subscribe-tunnel-policys` **永不接入**。

## Capabilities

### New Capabilities
- `huawei-tunnel-management-config`: 华为公网 tunnel-management（`/tunnel-management/tunnel-policys/tunnel-policy` 标量层 + `tunnel-down-switch`）的模型驱动配置管理——覆盖字段清单、命名空间登记、容器根 SchemaTree 入口、路由/编码/解码谓词语义、模拟网元 tunnel-management 方言、分期边界（深层 ipv4/ipv6-set 门控 follow-up）、以及完备测试矩阵要求。

### Modified Capabilities
<!-- 预期无：容器根走既有 XC-05（plain-container）、namespace 走既有 XC-06（per-node ns），编解码机制复用。若 apply 期往返实测暴露缺口（如某标量类型/容器根边界），再按 TDD 补对应 delta（yang-xml-codec）。届时回填本节。 -->

## Impact

- **代码**：`backend/internal/drivers/huawei.go`（+1 描述符 + tunnel-management namespace 常量）、`simulator/netconfsim`（+tunnel-management 方言）、新增 `*_integration_test.go`（B2）+ `internal/drivers` 编解码单测 + `pkg/yang-runtime/xmlcodec` 用例（若需）。**不动 `generated/`、不动 `gen.conf`**。
- **依赖**：`ifm`（✅ 已集成，`tunnel-name` leafref 目标，但本波次标量边界不触及）；`huawei-tunnel-management` 结构体（✅ 一期已生成）。本 change 功能依赖 = 零未满足项。
- **生成物边界登记（防误判）**：本 change 是 `huawei-tunnel-management` 首个**功能集成**（此前仅有惰性生成类型、无描述符/无配置通道，属 public-config 登记的 generated-but-not-integrated）。本波次功能集成的配置面仅 `/tunnel-management` 标量边界；深层 ipv4/ipv6-set 仍为 generated-but-not-integrated，勿因类型存在误判已集成。
- **前端**：tunnel-management 配置经通用「模块控制台」由 YANG 模型自动渲染（R05），本期不新增前端硬编码表单。
- **下游解锁**：本 change 合入后，BGP AF `tunnel-policy` 属性（波次⑤）与 `routing-policy` 集成（波次③）解除对 tunnel-management 的阻塞。
- **合规**：R01/R02/R03/R04/R17、T02b、B2 集成、worktree 隔离、≤500 行/commit、PR ≤1000 行。
