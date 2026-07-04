# device-native-lowcode-config — design（P1 场景① 设备原生面真低码）

> change：`device-native-lowcode-config` | 依赖：`proposal.md`

## Context

USMP 有两个配置面：设备原生面（场景①，YANG 直配）与意图面（场景②，CRD→翻译）。当前意图面（CRD 驱动动态表单）是活跃路径，而设备原生面的「基于交换机 YANG 模型动态生成低码表单」基本是桩：
- `yang-api` schema 硬编码在 handler（`internal/api`），未从 YANG 模型生成；`manager` 运行时 schema 树为空（迁移债 D4，`manager.go` 从不设 `SchemeDir`）。
- `config-api` POST 用硬编码 `convertToTypedStruct`（system/ifm/vlan 三条 path），其余回退裸 map。
- 前端设备侧 YANG 驱动表单 `components/yang/*` 未接路由（死代码 D9）；`ConfigStore.List/ListDevices` 返回 nil（D7）。

本变更在 R01 权威的 Stack B（`backend/main.go` + `pkg/yang-runtime`）上打通这条链路。架构总览见 `openspec/specs/system-architecture/design.md`。

## Goals / Non-Goals

**Goals:**
- 交换机全量 YANG 可配置属性经**通用前端控件 + 后端动态 schema**低码渲染与下发（R05）。
- `yang-api` 出真·动态 schema；`config-api` 通用 path↔ygot 编解码；`manager` schema 树可用；设备可枚举。
- 全程落 Stack B，不引入 DB（R03），不手写 YANG 结构（R04），TDD（R06）。

**Non-Goals:**
- 场景②意图面（CRD/translator/Actor 退役）——P2。
- gNMI 补齐、多厂商翻译、plugin 钩子——P3。
- 不改对外 REST 路径/响应结构（仅扩大语义覆盖）。

## Decisions

### D-1 schema 源：混合（设备能力优先 → ygot 模型树回退）【已定】
- 运行时优先从设备 NETCONF **hello capabilities**（及可选 `<get-schema>`）确定该设备支持的 YANG 模块/版本；不可得时回退到 `internal/generated` 已生成的 ygot YANG 模型树。
- schema 树由 `pkg/yang-runtime/schema` 构建并挂到 `manager`（修 D4）。`yang-api` 从该树生成前端 FieldDef，不再硬编码。
- 理由：最贴近「基于交换机 YANG 对象动态生成」，又有离线兜底，设备不支持 `<get-schema>` 时仍可用。
- 权衡：设备能力 ≠ 完整 schema（capabilities 只给模块清单/版本，属性细节仍取自 ygot 模型树）；故实现上「能力定模块集合，模型树定属性 schema」的组合，`<get-schema>` 作为增强项可后置。

### D-2 config-api 通用编解码：path ↔ ygot（替硬编码三条）
- 建立**path→ygot 类型**的通用编解码：POST body(JSON) → 依 path 定位 ygot 目标类型 → `ygot` 反序列化/构造 → `ConfigStore.Set`；GET 反向由 ygot/树序列化。
- 不再逐模块手写 `convertMapToHuaweiXxx`；用 ygot 生成结构 + 反射/JSON 绑定（R04：不滥用 interface{}，以 ygot 强类型为准）。
- 迁移期：保留旧 `convertToTypedStruct` 作为回退分支，双路径对拍（同输入两条路径 desired 等价）后切换、删除。

### D-3 前端：复用活跃低码引擎，仅换 schema 源
- 复用 `components/config/DynamicForm/FieldRenderer/DynamicTable`（活跃路径），把设备原生模块的 schema 从 CRD/硬编码换为 `GET /api/v1/yang/schema/:module`（动态）。
- `useConfigPage.ts` 原生模块分支已调 yang-api，只需让其 schema 变为动态生成的真 schema；退役设备侧 `components/yang/*` + `types/yang-schema.ts`（先并存、切换后删）。
- 理由：不新造第二套渲染引擎（避免重蹈 D9 双代），R05 一套通用控件。

### D-4 ConfigStore.List/ListDevices（修 D7，前置 P0）
- 用内存 ConfigStore 现有条目枚举设备与已存 path，供 `PeriodicSource(deviceIDs)` 轮询与设备列表。仍纯内存（R03）。

### D-5 渐进迁移与验证（§5.3）
- 每项：旧硬编码保留 → 新动态并行 → 双路径验证（yang-api 动态 schema vs 旧硬编码 schema 对 IFM/VLAN 等价；config-api 新编解码 vs 旧 convert 对拍 desired 等价）→ 切换入口 → 删旧。
- 每模块 netconfsim 端到端集成测试（T02）：前端/REST 提交 → reconcile → NETCONF 落 netconfsim → 断言。

## Risks / Trade-offs

- **schema 完整度**：ygot 模型树能覆盖生成模块的属性，但设备真实支持集可能是子集；用 capabilities 收敛模块集合缓解，属性级差异标注为已知限制（可后续 `<get-schema>` 增强）。
- **通用编解码的边界**：ygot 对含 map 的结构（如 vlan map）序列化历史上需手搓 XML（见客户端 marshalChange）；编解码层需处理 map/list 键，先覆盖在用模块，未知路径**保留裸 map 回退**并 `log` 告警（不静默截断）。
- **双路径期成本**：迁移期两套 schema/编解码并存增加临时复杂度；以对拍测试锁定等价、尽快切换删除控制窗口。
- **前端死代码清理**：`components/yang/*` 退役需确认无隐藏路由/引用（grep + 构建验证）后再删。
- **范围克制**：严格不碰场景②，避免与 P2 的 Actor 退役耦合导致大爆炸变更。
