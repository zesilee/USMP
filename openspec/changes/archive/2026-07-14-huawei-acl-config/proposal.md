## Why

BGP 二期-2b 的 peer/AF **ACL group** 属性（`common:*` leafref → `/acl:acl/groups/group/identity` 与 `/acl:acl/group6s/group6/identity`，见 huawei-bgp-common.yang:1089/1124/1160/1195/1572… 多处）门控于其目标模型 `huawei-acl` 的集成——`require-instance` 默认为真，目标 ACL group 实例不存在时下发即设备侧非法（越序禁令，见归档 change huawei-bgp-public-config 的 design.md）。

`huawei-acl` 是 2b 依赖 DAG 的一支。继波次①②③（tunnel-management #155 / xpl #156 / routing-policy #157 已合入 main）之后，本 change 交付波次④：让 acl `groups/group`（IPv4）与 `group6s/group6`（IPv6）成为 USMP 可配模型，ACL group 实例按 identity 存在，后续（波次⑤）才能合法接入 BGP ACL group 属性。

**范围核实纠正了早前的 codegen 顾虑**：早前登记「acl 需先补 time-range/l3vpn codegen」。实测——(a) `time-range` 类型一期全闭包已生成（`HuaweiTimeRange_*` 14 类型）；(b) `l3vpn` 仅在 acl 深层 `rule-*` 子树的 **`must` 约束** 中被引用（非 require-instance leafref），且 `must` 在 l3vpn 未配时 xpath 求空自满足；acl group **标量边界**（identity/type/match-order/step/description/number）**不引用 l3vpn/time-range**。故本波次 = **零 codegen**（acl 62 类型一期已生成），与波次①②③同构；l3vpn 仅在接入深层 rule-* 时才需，属 follow-up。

## What Changes

- **接入 acl group 配置面**：根容器 `/acl:acl`（`huawei-acl` 模块顶层独立容器根，namespace `urn:huawei:yang:huawei-acl`，容器根非 list 根，与 `/bgp:bgp` 等同构，走通用引擎 plain-container XC-05 + per-node ns XC-06）。本波次接入 BGP ACL group 的目标子树 `groups/group`（IPv4，key=`identity`）与 `group6s/group6`（IPv6，key=`identity`）的**标量/枚举边界**全部 config-true leaf，无遗漏：
  - `groups/group`：`identity`（key）、`type`（**mandatory** enum group4-type）、`match-order`（enum）、`step`（uint32）、`description`（string）、`number`（uint32）
  - `group6s/group6`：`identity`（key）、`type`（**mandatory** enum group6-type）、`match-order`（enum）、`step`、`description`、`number`
  - **此边界已完整满足 BGP ACL group leafref**：目标 list 实例按 `identity` 存在即可解析。首次覆盖**枚举 leaf**（type/match-order），验证通用引擎枚举编解码。
- **零 codegen**：`HuaweiAcl_Acl` 及子树（62 类型）已随一期 huawei-bgp 全闭包生成。本 change **不改 `gen.conf`、不 regen、不 touch generated/**（R04）。
- **驱动注册**：`backend/internal/drivers/huawei.go` 新增一条 `driver.Descriptor{Vendor:"huawei", Module:"acl"}`——谓词精确锚定 `/acl:acl` + 显式 `Namespace` 常量 + SchemaTree 入口闭包（`HuaweiAcl_Acl`）。编解码全走通用引擎，零 XML 代码。
- **容器根 reconciler**：`internal/controller/acl` 镜像已合入的 bgp/tunnelmgmt/xpl/routingpolicy 容器根 reconciler（单条整根 MODIFY 收敛防漂移）。
- **完备测试矩阵**（`yang-config-test-design` / T02b）：全（标量/枚举边界）属性可配（含枚举往返）/ 端到端到设备（B2）/ 并发-race / 边界 / 幂等 / 负路径 / 删除语义。
- **明确排除（分期，注册为 follow-up，非简化）**：
  - group/group6 下**深层 `rule-*` 子树**（rule-basics/rule-advances/rule-ethernets/rule-interfaces/rule-mplss——实际 ACL 规则条目，含 `must` 引用 l3vpn/network-instance、time-range 引用、深层嵌套 list）——门控于通用引擎对深层嵌套的支持 **及** l3vpn 模型集成（其 must 目标），拆到 follow-up。BGP 仅按 identity 引用，标量边界已解除阻塞。
  - acl 的**其他子树**（`ip-pools`/`ip-pool6s`/`port-pools`）——非 BGP ACL group leafref 目标，follow-up。config-false 只读态永不接入。

## Capabilities

### New Capabilities
- `huawei-acl-config`: 华为 acl `groups/group`（IPv4）+ `group6s/group6`（IPv6）标量/枚举层的模型驱动配置管理——覆盖字段清单（含枚举 type/match-order）、命名空间登记、容器根 SchemaTree 入口、路由/编码/解码谓词语义、B2 端到端、分期边界（深层 rule-* 门控 l3vpn+follow-up）、以及完备测试矩阵要求。

### Modified Capabilities
<!-- 预期无：容器根 + 枚举 leaf 均由既有引擎覆盖（BGP 2a af-type 枚举已实证）。若 apply 期暴露缺口再按 TDD 补 delta。 -->

## Impact

- **代码**：`backend/internal/drivers/huawei.go`（+1 描述符 + acl namespace 常量）、`internal/controller/acl`（新增容器根 reconciler）、新增 `*_integration_test.go`（B2）+ 编解码单测。**不动 `generated/`、不动 `gen.conf`**。
- **依赖**：`huawei-acl` 结构体（✅ 一期已生成）；功能依赖 = 零未满足项（标量边界不引用 l3vpn/time-range；l3vpn 仅深层 rule-* 的 must 目标，本波次不接）。
- **序列化**：本 change 改 `internal/drivers/huawei.go`，与波次①②③（#155/#156/#157 已合入）同文件——按 TM03 串行，基于已合入 main。
- **生成物边界登记**：本 change 是 `huawei-acl` 首个功能集成，仅 `groups/group` + `group6s/group6` 标量边界有功能通道；深层 rule-* 与 ip-pools/port-pools 仍 generated-but-not-integrated；l3vpn 仍完全未集成。
- **前端**：acl 配置经通用「模块控制台」YANG 自动渲染（R05），不新增硬编码表单。
- **下游解锁**：本 change 合入后，BGP AF ACL group 属性（波次⑤）解除对 acl 的阻塞。至此波次⑤ 的四类策略属性目标（tunnel-policy/route-filter/route-policy/ACL group）前置全部就位。
- **合规**：R01/R02/R03/R04/R17、T02b、B2 集成、worktree 隔离、≤500 行/commit、PR ≤1000 行。
