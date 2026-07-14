## Why

BGP 二期-2b 的 peer/AF **route-filter** 属性（`common:*` leafref → `/xpl:xpl/route-filters/route-filter/name`，见 huawei-bgp-common.yang:528/1424/1701/2233/2269 等多处，type `xpl:filter-parameter-type`）门控于其目标模型 `huawei-xpl` 的集成——`require-instance` 默认为真，目标 route-filter 实例不存在时下发即设备侧非法（越序禁令，见归档 change huawei-bgp-public-config 的 design.md）。

`huawei-xpl` 是 2b 依赖 DAG 的**叶子**（唯一功能依赖 `ifm` 已集成✅）。继波次①（tunnel-management，PR #155 已合入）之后，本 change 交付波次②：让 xpl `route-filters/route-filter` 成为 USMP 可配模型，route-filter 实例按 name 存在，后续（波次⑤）才能合法接入 BGP `route-filter` 属性。

## What Changes

- **接入 xpl route-filter 配置面**：根容器 `/xpl:xpl`（`huawei-xpl` 模块顶层独立容器根，namespace `urn:huawei:yang:huawei-xpl`，容器根非 list 根，与 `/bgp:bgp`、`/tnlm:tunnel-management` 同构，走通用引擎 plain-container XC-05 + per-node ns XC-06）。本波次接入 BGP 引用的目标子树 `route-filters/route-filter`（key=`name`）的**全部 config-true leaf，无遗漏**：
  - `name`（`xpl-filter-name` 类型，key）
  - `content`（string 1..16380，**mandatory**，XPL 策略正文文本）
  - **此边界已完整满足 BGP `route-filter` leafref**：目标 list 实例按 `name` 存在即可解析。
- **零 codegen**：`HuaweiXpl_Xpl` 及子树（23 类型）已随一期 huawei-bgp 全闭包生成。本 change **不改 `gen.conf`、不 regen、不 touch generated/**（R04）。
- **驱动注册**：`backend/internal/drivers/huawei.go` 新增一条 `driver.Descriptor{Vendor:"huawei", Module:"xpl"}`——谓词精确锚定 `/xpl:xpl` + 显式 `Namespace` 常量 + SchemaTree 入口闭包（`HuaweiXpl_Xpl`）。编解码全走通用引擎 `pkg/yang-runtime/xmlcodec`，零 XML 代码。
- **容器根 reconciler**：`internal/controller/xpl` 镜像已合入的 bgp/tunnelmgmt 容器根 reconciler（单条整根 MODIFY 收敛防漂移）。
- **完备测试矩阵**（`yang-config-test-design` / T02b）：全属性可配 / 端到端到设备（B2）/ 并发-race / 边界 / 幂等 / 负路径 / 删除语义。
- **明确排除（分期，注册为 follow-up，非简化）**：xpl 的**其他策略 list**——`global`、`as-path-lists`、`community-lists`、`ext-community-rt-lists`、`ext-community-soo-lists`、`ipv4-prefix-lists`、`ipv6-prefix-lists`、`rd-lists`、`large-community-lists`、`route-flow-group-lists`、`interface-lists` 等。这些不是 BGP `route-filter` leafref 的目标（BGP 仅引用 `route-filters/route-filter`），各由其自身消费者门控，门控于对应 change；本波次只接 BGP route-filter 所需的 `route-filters/route-filter`。config-false 只读态永不接入。

## Capabilities

### New Capabilities
- `huawei-xpl-config`: 华为 xpl `route-filters/route-filter`（name + content）的模型驱动配置管理——覆盖字段清单、命名空间登记、容器根 SchemaTree 入口、路由/编码/解码谓词语义、B2 端到端、分期边界（其他 xpl 策略 list 门控各自消费者）、以及完备测试矩阵要求。

### Modified Capabilities
<!-- 预期无：容器根走既有 XC-05/XC-06，编解码机制复用（波次① tunnel-management 已实证容器根无新缺口）。若 apply 期往返实测暴露缺口再按 TDD 补 delta（yang-xml-codec）。 -->

## Impact

- **代码**：`backend/internal/drivers/huawei.go`（+1 描述符 + xpl namespace 常量）、`internal/controller/xpl`（新增容器根 reconciler）、新增 `*_integration_test.go`（B2）+ `internal/drivers` 编解码单测 + `pkg/yang-runtime/xmlcodec` 用例。**不动 `generated/`、不动 `gen.conf`**。
- **依赖**：`huawei-xpl` 结构体（✅ 一期已生成）；功能依赖 = 零未满足项（ifm✅，且 route-filter 不引用 ifm）。
- **序列化**：本 change 改 `internal/drivers/huawei.go`，与波次①（已合入 #155）、后续波次③④同文件——按 TM03 串行，基于 #155 已合入的 main。
- **生成物边界登记**：本 change 是 `huawei-xpl` 首个功能集成，仅 `route-filters/route-filter` 有功能通道；其他 xpl list 仍 generated-but-not-integrated。
- **前端**：xpl 配置经通用「模块控制台」YANG 自动渲染（R05），不新增硬编码表单。
- **下游解锁**：本 change 合入后，BGP AF `route-filter` 属性（波次⑤）解除对 xpl 的阻塞。
- **合规**：R01/R02/R03/R04/R17、T02b、B2 集成、worktree 隔离、≤500 行/commit、PR ≤1000 行。
