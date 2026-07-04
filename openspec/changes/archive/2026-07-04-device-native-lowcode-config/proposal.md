## Why

场景①「交换机全量配置」的核心承诺——**前端只定义通用控件，后端基于交换机 YANG 模型动态生成可配置属性**——目前基本是桩：`yang-api` 的 schema 全部硬编码在 handler（仅 IFM/VLAN，未知模块回退 2 字段），`config-api` POST 只认硬编码的三条 path（system/ifm/vlan），设备端 YANG 驱动的前端动态表单是未接路由的死代码。根因是迁移债 **D4**（manager 从不加载 YANG 模型、运行时 schema 树为空）与 **D7**（ConfigStore 无法枚举设备）。本变更在 R01 权威的 Stack B 上把这条链路真正打通，作为架构优化 P1（场景① 设备原生面真低码）。

## What Changes

- **修 D4：加载 YANG 模型进 manager schema 树**，采用**混合源**——运行时优先读设备 NETCONF hello capabilities / `<get-schema>`，缺失回退到 `internal/generated` 的 ygot YANG 模型树。
- **`yang-api` 动态化**：`/yang/modules`、`/yang/schema/:module` 改为从 schema 树动态生成，替换 handler 硬编码；未知模块不再回退 2 字段桩。
- **`config-api` 通用编解码**：POST 改为通用 **path ↔ ygot** 编解码，替换硬编码 `convertToTypedStruct`（system/ifm/vlan 三条），支持任意已加载 YANG 路径。
- **前端接通 YANG 驱动动态表单**：复用当前活跃的 `DynamicForm`/`FieldRenderer` 引擎，把设备原生模块的 schema 源改为 `yang-api` 动态 schema；清理/迁移设备侧死代码 `components/yang/*`（D9 设备侧）。
- **修 D7（前置 P0）**：补 `ConfigStore.List`/`ListDevices`，使 `PeriodicSource` 可枚举设备。
- **不改动**场景②意图面（CRD/translator/Actor）——留待 P2。
- 遵 §5.3 渐进迁移：旧 hardcoded 路径**保留并行 → 双路径验证 → 切换 → 删除**；非破坏性对外 REST 契约（路径/响应形态保持，语义由「硬编码 3 模块」扩为「任意已加载 YANG 模块」）。

## Capabilities

### New Capabilities
<!-- 无新增能力：本变更是让既有能力真正兑现契约（去桩），不新造能力域。 -->

### Modified Capabilities
- `yang-controller-runtime`: 新增 YANG 模型加载进 schema 树（混合源，修 D4）；`ConfigStore` 补齐 `List`/`ListDevices`（修 D7）——从「schema 树运行时为空、设备不可枚举」变为「schema 树可用、设备可枚举」。
- `yang-api`: `/yang/modules`、`/yang/schema/:module` 的 schema **来源契约**从「handler 硬编码」变为「从 schema 树动态生成」；未知已加载模块返回真实 schema 而非 2 字段桩。
- `config-api`: POST 的类型转换契约从「硬编码 path→struct（3 条）」变为「通用 path↔ygot 编解码（任意已加载 YANG 路径）」。
- `frontend`: 设备原生配置页的 schema 源契约从「CRD/硬编码」变为「yang-api 动态 YANG schema」，经通用低码引擎渲染（R05）；退役设备侧静态 YANG 死代码。

## Impact

- **后端**：`pkg/yang-runtime/manager`（schema 加载、ConfigStore.List/ListDevices）、`pkg/yang-runtime/schema`（schema 树构建，去空转 D4）、`pkg/yang-runtime/client`（NETCONF capabilities/`<get-schema>` 读取）、`internal/api`（yang-api / config-api handler）、`internal/generated`（作为回退模型源，只读）。
- **前端**：`composables/useConfigPage.ts`（设备原生模块 schema 源）、`components/config/*`（复用）、`components/yang/*` + `types/yang-schema.ts`（设备侧死代码清理）。
- **测试**：每模块 netconfsim 端到端集成测试（T02）；动态 schema/编解码单测（正常/异常/未知路径/并发）。
- **红线**：R01（落 Stack B）、R03（仍仅内存 ConfigStore）、R04（ygot 生成、编解码不滥用 interface{}）、R05（通用控件 + 动态渲染）、R06（TDD 测试先行）。
- **对外契约**：REST 路径与响应结构不变；行为语义扩大（更多模块可配）；旧硬编码路径迁移期并存，切换后删除。
- **不在范围**：场景②意图面、Actor 退役、gNMI、多厂商翻译（P2/P3）。
