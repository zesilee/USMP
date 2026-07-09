# nce-fidelity-polish — NCE 保真微调（P1'）

## Why

iMaster NCE 调研（`docs/research/imaster-nce-ux-insights.md` 洞察 G+F，§2.1/§2.2 有截图佐证）给出了 R05 自动渲染的终态视觉参照。P0/P3'/P4 已交付能力面，剩余为控件级保真差距（2026-07-09 探索实证，代码级核对）：

1. **enum 恒 el-select**（`FieldRenderer.vue:38`）——NCE 对 ≤3 选项的枚举用分段控件（segmented），一眼可见全部选项，少一次弹层交互；Element Plus 已锁定 2.13.7（`el-segmented` 2.7+ 可用，`main.ts` 全量注册无需按需引入）。
2. **表格无高密度形态**——NCE 是「高密度运维台账风，列多、留白克制」，USMP 的 el-table 全部沿用 Element Plus 默认密度（`theme.scss` 无 padding/size 覆盖，`DynamicTable.vue`/`ModuleListTab.vue` 未设 size）。
3. **FieldRenderer 内联硬编码色值绕过 token 体系**——`FieldRenderer.vue` styles 直写 `#909399/#f9fafb/#e5e7eb/#606266`，与已合入 redesign 的 `variables.scss` 单一事实源相悖，主题一致性有漂移风险。

**范围外（探索阶段裁定，防止范围蔓延）：**
- **leafref→el-transfer**：后端 `mapLeafType` 把 leafref 落成 string（`entry.go:437`），契约无 leafref 类型；要做需后端新 LeafType + 前端动态拉引用候选集，是独立 feature 非微调，且消费场景未证实。
- **主色换 `#307FE2`**：该色为调研文档自认的「肉眼估读」；已批准合入的 redesign 刻意选定 `#0C5EA6` 深钢蓝并成体系，不以估读色推翻既有设计决策。
- **description→控件下行内说明**：FieldDef 无 Description 字段，属后端契约变更，如需另立项（呈现元数据收割二期）。

## What Changes

- **S1 enum 分段控件**：`FieldRenderer.vue` enum 分支——**必填且选项 ≤3** 渲染 `el-segmented`；可选枚举保持 `el-select`（segmented 无 clearable，可选叶需保留「清空=不下发该键」能力）；>3 选项保持 `el-select`。阈值启发式与既有 `choiceUsesRadio` 同类（前端「注解优先、算法兜底」的算法兜底层；widget 注解已在 P3' 决策推迟）。
- **S2 高密度表格**：`theme.scss` 增表格密度覆盖（单元格 padding/行高/字号对齐 NCE 台账密度），`DynamicTable.vue` 与 `ModuleListTab.vue` 的 el-table 收敛到统一密度形态。
- **S3 色值令牌化**：`FieldRenderer.vue` 内联 hex 全部替换为 `variables.scss`/CSS 变量既有 token，不新增 token、不改视觉语义（灰阶近似映射）。

## Capabilities

### New Capabilities

（无）

### Modified Capabilities

- `frontend`：FE-01「schema 驱动渲染」的类型→控件映射细化——enum 增加必填 ≤3 → segmented 分支（MODIFIED）。S2/S3 为纯呈现样式收敛，无契约/行为变更，不动 spec。

## Impact

- **前端**：`src/components/config/FieldRenderer.vue`（S1 分支 + S3 样式）、`src/styles/theme.scss`（S2 密度）、`src/components/config/DynamicTable.vue` / `ModuleListTab.vue`（S2 接入）。
- **后端**：零改动（FieldDef/契约不变，`make gen-contract` 应零漂移）。
- **测试层**（§5.6，前端组件逻辑 → F2）：F2 覆盖 segmented 渲染分支（必填≤3/可选≤3/＞3/禁用态/emit）+ 存量 enum→select 用例适配；密度与令牌化由存量 F2/F4 冒烟守护（无行为断言新增）。el-segmented 为行内组件（无 teleport 弹层），不触发 F3 强制条款；若 happy-dom 渲染受限则按 §5.6 升 F3。
- **门禁**：覆盖率棘轮 前端 74/71/67/74（补测后按需上调）；PR 体积远低于 1000 行。
- 不涉及：数据库（R03）、YANG 模型（无 yang-config-test-design 触发）、后端下发链路。
