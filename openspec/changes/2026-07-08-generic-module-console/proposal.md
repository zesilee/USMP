# Change: generic-module-console

## Why

现有 Stack B 配置页（`DeviceConfigPage` + `useDeviceConfig`）虽是模型驱动，但只呈现**单个目标 list**（路由 props 硬编码 `itemListSuffix`/`columns`），模块根下的其余顶层节点（如 `ifm` 的 `global` 统计/冲突开关、`damp` 阻尼、`auto-recovery-times` 自愈时间）**全部不可见不可配**；表格列写死在路由 props、无高级搜索、无分页、操作列不受模型控制。同时 YANG 里三类关键呈现元数据仍被丢弃：

- `ext:support-filter`（哪些字段可作查询条件）——`Entry.Exts` 中存活但前后端均未透出；
- `ext:operation-exclude`（create-only/不可删字段与节点）——同上，导致编辑态可改不可改字段无从区分；
- presence 容器（如 `ipv4-conflict-enable`）与容器级 `when/must` ——被当普通 group，丢失「存在即启用」语义与条件门禁。

后果：接入一个新 YANG 模块仍要人工设计路由 props 与列，且大半模块能力（模块级全局属性）没有入口，违背 R05「YANG 自动渲染」与「新增模块零前端代码」的目标。

本变更交付**通用模块控制台**：左侧导航（模块列表驱动）→ 右侧「面包屑 + 一级 Tab」布局，Tab 由模块根的顶层子节点自动派生（container→表单页、list→列表页）；列表页含模型驱动列派生、`ext:support-filter` 驱动的高级搜索、分页、`ext:operation-exclude` 驱动的操作门禁、行级 `when` 动态单元格；表单页复用约束引擎并新增 presence→开关渲染。**引擎零厂商/零模块硬编码**，以 `huawei-ifm`（interfaces/global/damp/auto-recovery-times，含 support-filter、operation-exclude、presence+must）作为高复杂度验证模型。

## What Changes

- **后端 `yang-api`**
  - `schema.LeafNode` 新增 `SupportFilter()`/`OperationExcludes()`：从 goyang `Entry.Exts` 按扩展关键字本名（`support-filter`/`operation-exclude`，前缀无关）采集；`ContainerNode` 新增 `WhenExpr()`/`MustExprs()`（复用既有 Extra 采集），presence 经既有 `IsPresence()` 透出。
  - `FieldDef` 扩展：`supportFilter`(bool)、`operationExclude`([]string)、`presence`(bool)；group 类型 FieldDef 携带 `when`/`must`。
- **前端 `frontend`**
  - 新增通用页 `ModuleConsolePage`（路由 `/module/:module`，零 per-module props）：面包屑 + 设备选择 + 一级 Tab（模块根顶层子节点派生）。
  - 新增 `ModuleListTab`：模型驱动列派生（key→identity(operation-exclude∋update)→when 条件列→enum→其余，封顶 9 列）、enum 列 Tag 化、up/down 值状态点、行级 `when` 单元格（不满足显示 `-`）、工具栏（新增 + 高级搜索折叠面板：`supportFilter` 字段→enum 下拉/文本输入 + 查询/重置）、客户端分页、操作列（list 级 `operationExclude` 隐藏编辑/删除；编辑态叶级 `operationExclude∋update` 字段禁用）；编辑/新增复用既有 drawer 表单 + 约束引擎 + 对账流。
  - 新增 `ModuleFormTab`：container 类 Tab 的表单渲染（FieldRenderer + 约束引擎），presence 容器→开关（关=节点不存在，不入 payload），presence 容器 `must` 不满足→隐藏/禁用（对齐 when 门禁语义）。
  - `Sidebar` 业务配置菜单改为 `/yang/modules` 驱动（复用 menu store），`/config/interface`、`/config/vlan` 路由重定向到 `/module/:module`。
  - 模拟网元种子数据补 5 条接口（3 条 main-interface/200GE/up + 2 条 sub-interface/Vlanif/down，parent 指向前者），供 staging 冒烟与演示。
- **测试**：后端 B1（entry ext/presence 采集）+ B3（FieldDef 透传）；前端 F1（列派生/过滤纯逻辑）、F2（Tab 派生、搜索、分页、动态单元格、操作门禁、presence 开关、statistic-interval must）、F4（staging-smoke 增补模块控制台冒烟）。

## Impact

- **Affected specs**: `yang-api`（ADDED BR-07/BR-08）、`frontend`（ADDED FE-10/FE-11/FE-12/FE-13）。
- **Affected code（后端）**: `pkg/yang-runtime/schema/{types.go,schema.go,entry.go}`、`internal/api/{yang_handler.go,field_gen.go}`。
- **Affected code（前端）**: 新增 `views/ModuleConsolePage.vue`、`components/config/{ModuleListTab.vue,ModuleFormTab.vue}`、`utils/moduleConsole.ts`（列派生/过滤纯逻辑）；改 `router/index.ts`、`components/layout/Sidebar.vue`、`utils/crdSchemaParser.ts`(Field 扩宽)、`components/config/FieldRenderer.vue`(presence)。
- **写入链路**: 不变——列表页下发仍走 `setConfig(configPath)` 既有链路；表单 Tab 对后端暂不支持写入的路径（如 `ifm` global/damp）提交时如实透出后端错误（§9 降级，不伪装成功）。
- **兼容**: `FieldDef`/`Field` 契约**扩宽不破坏**；`DeviceConfigPage` 旧路由重定向保留书签可达。
- **R 合规**: R05（全部呈现元数据模型驱动）、R08（扩展缺失/求值失败降级）、R10（零新依赖）、R11/R12（沿用既有设计系统与图标）、R17（本 change 即 spec-first）；TM04 分 PR 交付见 tasks.md。
