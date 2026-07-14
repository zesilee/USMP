## Why

BGP 二期-2b 的 peer/AF **import/export route-policy** 属性（`common:*` leafref → `/rtp:routing-policy/policy-definitions/policy-definition/name`，见 huawei-bgp-common.yang:165/551/575 等多处）门控于其目标模型 `huawei-routing-policy` 的集成——`require-instance` 默认为真，目标 policy-definition 实例不存在时下发即设备侧非法（越序禁令，见归档 change huawei-bgp-public-config 的 design.md）。

`huawei-routing-policy`（rtp）在 2b 依赖 DAG 中依赖 `tunnel-management`（波次①，PR #155 已合入 main）。继波次①②（tunnel-management #155 / xpl #156）之后，本 change 交付波次③：让 rtp `policy-definitions/policy-definition` 成为 USMP 可配模型，route-policy 实例按 name 存在，后续（波次⑤）才能合法接入 BGP import/export route-policy 属性。

## What Changes

- **接入 rtp route-policy 配置面**：根容器 `/rtp:routing-policy`（`huawei-routing-policy` 模块顶层独立容器根，namespace `urn:huawei:yang:huawei-routing-policy`，容器根非 list 根，与 `/bgp:bgp`、`/tnlm:tunnel-management`、`/xpl:xpl` 同构，走通用引擎 plain-container XC-05 + per-node ns XC-06）。本波次接入 BGP import/export route-policy 的目标子树 `policy-definitions/policy-definition`（key=`name`）的**标量边界**全部 config-true leaf：
  - `name`（key）
  - `address-family-mismatch-deny`（boolean）
  - **此边界已完整满足 BGP import/export route-policy leafref**：目标 list 实例按 `name` 存在即可解析。
- **零 codegen**：`HuaweiRoutingPolicy_RoutingPolicy` 及子树（282 类型）已随一期 huawei-bgp 全闭包生成。本 change **不改 `gen.conf`、不 regen、不 touch generated/**（R04）。
- **驱动注册**：`backend/internal/drivers/huawei.go` 新增一条 `driver.Descriptor{Vendor:"huawei", Module:"routing-policy"}`——谓词精确锚定 `/rtp:routing-policy` + 显式 `Namespace` 常量 + SchemaTree 入口闭包（`HuaweiRoutingPolicy_RoutingPolicy`）。编解码全走通用引擎，零 XML 代码。
- **容器根 reconciler**：`internal/controller/routingpolicy` 镜像已合入的 bgp/tunnelmgmt/xpl 容器根 reconciler（单条整根 MODIFY 收敛防漂移）。
- **完备测试矩阵**（`yang-config-test-design` / T02b）：全（标量边界）属性可配 / 端到端到设备（B2）/ 并发-race / 边界 / 幂等 / 负路径 / 删除语义。
- **明确排除（分期，注册为 follow-up，非简化）**：
  - `policy-definition` 下**深层 `nodes/node` 子树**（~1490 行：node 的 `conditions`（match-tag/match-community-filters/match-as-path-filters/match-protocols/match-cost 等）+ `actions/apply` 子句、嵌套 list、choice）——这是 route-policy 的实际匹配/动作逻辑，门控于通用引擎对深层嵌套/choice 的支持，拆到本波次-follow-up。BGP 仅按 name 引用，标量边界已解除 leafref 阻塞。
  - rtp 的**其他 filter/list**（community-filters、ext-community-*、ipv4/ipv6-prefix-filters、as-path-filters、rd-filters、large-community-* 等）——各由其自身 BGP 消费属性（filter-policy/prefix 过滤/as-path 过滤等）门控，随对应属性接入时再补，本波次不接。config-false 只读态永不接入。

## Capabilities

### New Capabilities
- `huawei-routing-policy-config`: 华为 rtp `policy-definitions/policy-definition`（name + address-family-mismatch-deny 标量层）的模型驱动配置管理——覆盖字段清单、命名空间登记、容器根 SchemaTree 入口、路由/编码/解码谓词语义、B2 端到端、分期边界（深层 nodes + 其他 filter 门控 follow-up）、以及完备测试矩阵要求。

### Modified Capabilities
<!-- 预期无：容器根走既有 XC-05/XC-06（波次①② 已实证无新缺口）。若 apply 期暴露缺口再按 TDD 补 delta（yang-xml-codec）。 -->

## Impact

- **代码**：`backend/internal/drivers/huawei.go`（+1 描述符 + rtp namespace 常量）、`internal/controller/routingpolicy`（新增容器根 reconciler）、新增 `*_integration_test.go`（B2）+ 编解码单测。**不动 `generated/`、不动 `gen.conf`**。
- **依赖**：`tunnel-management`（✅ #155 已合入 main，rtp 的 2b DAG 前置）；`huawei-routing-policy` 结构体（✅ 一期已生成）。
- **序列化**：本 change 改 `internal/drivers/huawei.go`，与波次①②（#155/#156 已合入）、后续波次④同文件——按 TM03 串行，基于已合入 main。
- **生成物边界登记**：本 change 是 `huawei-routing-policy` 首个功能集成，仅 `policy-definitions/policy-definition` 标量边界有功能通道；深层 nodes 与其他 rtp filter 仍 generated-but-not-integrated。
- **前端**：rtp 配置经通用「模块控制台」YANG 自动渲染（R05），不新增硬编码表单。
- **下游解锁**：本 change 合入后，BGP AF import/export route-policy 属性（波次⑤）解除对 rtp 的阻塞。
- **合规**：R01/R02/R03/R04/R17、T02b、B2 集成、worktree 隔离、≤500 行/commit、PR ≤1000 行。
