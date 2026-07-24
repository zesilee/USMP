---
name: ext-ui-annotations
description: 呈现元数据收割 change（readonly/dynamic-default/units/task-name）关键实证与决策；扩展注解词汇/改 FieldDef/加呈现元数据前必读
metadata: 
  node_type: memory
  type: project
  originSessionId: 5becb587-f21e-4cb0-83ae-7eed4c3594be
---

Change `ext-ui-annotations`（P3'，PR #126 合入 main + #127 sync/归档，2026-07-08）：收割存量呈现元数据 → FieldDef，零自造 YANG 词汇。叠在 [[generic-module-console]] 管线（Entry.Exts→entry.go→FieldDef→gen-contract）之上。

**关键实证（探针实测内嵌 gzip schema，改这块前必读）：**
- **模块级 YANG 扩展不存活于运行期 schema**：`ext:task-name` 全 SchemaTree 扫描=0（ygot 生成丢弃模块级语句）→ 任务域分组走**构建期 codegen**（`backend/tools/tasknamegen` → `internal/yangschema/taskname.gen.go`，键=根容器名）。数据节点级 Exts（support-filter/operation-exclude/dynamic-default）才存活。
- `config false` 存活：Config=TSFalse 只在子树根、后代 TSUnset 继承；unzip 后 Parent 指针完好、`e.ReadOnly()` 可用，但实现选构树期下推（entry.go 各 builder 传 inheritedRO）。ifm 121 个 readonly 节点。
- 华为词汇盘点：全模型 30+ 关键字（node-ref/value-meaning/can-be-deleted…），但**编译的 3 模块只有 4 种**（operation-exclude/support-filter/dynamic-default×10/task-name×1）。`ext:can-be-deleted` 语义直指删除，归 P4。
- **决策：自造 `ext:ui-widget/ui-order/ui-view` 推迟**——无消费场景（segmented 可由 enum 基数派生、Tab 已由 container 派生、顺序=schema 序）+ patch 厂商模型=fork 维护债；将来用 augment/deviation 模块，不改厂商源文件（design.md 在 archive/2026-07-08-ext-ui-annotations）。

**踩坑：**
- **R04 pre-commit 门禁禁止提交 `internal/generated/` 任何改动**——生成物放属主包（taskname.gen.go 落 yangschema 而非 generated/huawei）。
- **readonly 叶 must 门禁死锁**：state 叶 must 违例（设备值用户不可改）会把 useConfigForm.blocked 永久置 true；已修（mustViolations 过滤 f.readonly）。同类新门禁要想「违例用户可修吗」。
- 真实 IFM `class` 也带 dynamic-default（测试负例要用 description 这类零扩展叶）。
- `frontend/coverage/` 误跟踪债又咬一口：`npm run test:coverage` 后 `git add -A` 会吞报告，须 restore（[[test-governance-military-rules]] 待办#2 仍未做）。

**交付面**：FieldDef+readonly（原恒空契约字段激活）/dynamicDefault/units、YangModuleInfo+category；前端 deriveTabs 只读 Tab、useConfigForm editableFlat（readonly 不入 rules/diff/payload/门禁）、dynamicDefault 空值豁免必填+不入 payload、units 后缀、Sidebar 按 category 分组（el-menu-item-group，全无 category 退化平铺）。

相关：[[generic-module-console]]、[[yang-constraint-engine]]、[[imaster-nce-ux-insights]]
