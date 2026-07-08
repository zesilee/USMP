# ext-ui-annotations — 收割存量呈现元数据（P3'）

## Why

R05 的终局是「渲染意图成为受版本管理的契约」。审计（2026-07-08，代码级核对）发现：呈现扩展管线（`Entry.Exts` → `entry.go` → FieldDef → `gen-contract` 漂移门禁）已成熟，但**已存活于内嵌 schema 的呈现元数据仍有四类未收割**，导致真实呈现缺陷：

1. `config false` 未透出 → `FieldDef.Readonly` 契约存在但恒空，前端把 remote-interfaces 等 **state 子树渲染成可编辑 Tab**（generic-module-console 遗留债 #3）；
2. `ext:dynamic-default`（华为 IFM 10 处：admin-status/link-protocol 等）未消费 → 前端无法区分「系统动态缺省」与「用户未配置」，空值语义误导；
3. `units` 已提取到 LeafNode 但 FieldDef 不透出 → 数值输入框丢失单位后缀；
4. `ext:task-name`（模块根级任务域）未消费 → 左导航无法按任务域分组，模块数增长后将退化为平铺长列表。

同期决策：**本期零自造 YANG 词汇**。设想中的 `ext:ui-widget/ui-order/ui-view` 当下无消费场景（segmented 可由 enum 基数派生、Tab 已由 container 派生、顺序=schema 序），且自造词汇需 patch 厂商模型源文件，构成 fork 维护债——推迟到出现真实消费场景，届时以 augment/deviation 模块承载（见 design.md）。

## What Changes

- **S1 只读透出**：后端从 goyang `Entry` 读 `config false`，填充既有 `FieldDef.Readonly`；前端通用控制台把 readonly 子树/叶降级为只读呈现（不可编辑、不入下发 payload、Tab 标记只读）。
- **S2 动态缺省**：后端消费 `ext:dynamic-default` 扩展 → 新增 `FieldDef.DynamicDefault bool`；前端对该类字段呈现「系统自动分配」占位语义（空值不视为缺配置、不强制必填）。
- **S3 单位后缀**：后端把 LeafNode 已有的 `units` 透出为 `FieldDef.Units string`；前端数值/文本输入框追加单位后缀。
- **S4 任务域分组**：模块级 `ext:task-name` 经实证**不存活**于内嵌运行期 schema（全树扫描=0，模块级语句在 ygot 生成时被丢弃）→ 改走**构建期 codegen**：仿 ygot go:generate，从 yang-models 提取模块级 task-name 生成 `taskname.gen.go` 映射表（提交入库，运行期零 submodule 依赖）；`/yang/modules` 响应新增 `category` 字段；前端左导航按 category 分组展示（无 category 归入默认组）。
- FieldDef 契约变更后同步 `make gen-contract`（api.gen.ts 漂移门禁）。
- 不改 Reconciler/写链路/NETCONF 编解码；纯 schema 透出 + 前端呈现层。

## Capabilities

### New Capabilities

（无 — 全部落在既有能力的需求扩展上）

### Modified Capabilities

- `yang-api`：新增 BR-09「原生呈现元数据透出（config-false→readonly / units）」、BR-10「dynamic-default 扩展透出」；BR-01 模块列表扩展 `category`（源自 `ext:task-name`）。
- `frontend`：新增 FE-14「state 子树只读降级」、FE-15「动态缺省占位与单位后缀」；FE-13 模型驱动导航扩展「按任务域分组」。

## Impact

- **后端**：`backend/pkg/yang-runtime/schema/entry.go`（+config/units/dynamic-default/task-name 提取）、`schema/schema.go`（LeafNode/Node 存取器）、`backend/internal/api/field_gen.go` + `yang_handler.go`（FieldDef/ModuleInfo 透出）。
- **前端**：`frontend/src/utils/moduleConsole.ts`（readonly 派生已半接线）、`components/config/FieldRenderer.vue`（units 后缀/动态缺省占位/只读态）、左导航组件（category 分组）、`src/types/api.gen.ts`（gen-contract 再生成）。
- **测试层**（§5.6）：B1（entry.go 提取，表格驱动+race）、B3（/yang/schema、/yang/modules 契约）、F1（moduleConsole 派生纯函数）、F2（FieldRenderer 只读/占位/后缀 + 导航分组）。
- **门禁**：`make gen-contract` 漂移门禁；覆盖率棘轮后端 57 / 前端 73/70/65/73，补测后按需上调。
- 不涉及：数据库（R03 不变）、协议层、下发链路、DeviceConfigPage（已退役）。
