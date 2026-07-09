# tasks — nce-fidelity-polish

> TDD 红绿循环（T01/T05）：每组先测试（红）再实现（绿）。测试层按 §5.6：前端组件逻辑 → F2。
> 单 commit ≤500 行、原子功能；What/Why/How 三段式。

## 1. S1 enum 分段控件（F2，FE-01 MODIFIED）

- [x] 1.1 探测：happy-dom 下 `el-segmented` 可渲染性最小用例（决定 D4 主/降级路径）
- [x] 1.2 F2 红：FieldRenderer enum 四态——必填≤3→segmented（渲染选项+emit update）、可选≤3→select（保留 clearable）、>3→select、readonly/disabled 透传 segmented
- [x] 1.3 绿：`FieldRenderer.vue` enum 分支加 segmented 判定（`required && options.length<=3`）
- [x] 1.4 存量适配：核对以 enum→select 为前提的既有 F2/F3 用例，受影响者按新规则改写

## 2. S2 高密度表格（theme.scss 全局，D2）

- [x] 2.1 F2 红（轻量）：DynamicTable/ModuleListTab 渲染冒烟不回归（存量用例即防线，补缺口才新增）
- [x] 2.2 绿：`theme.scss` el-table 覆盖块收紧 th/td padding 与行高至 NCE 台账密度

## 3. S3 FieldRenderer 色值令牌化（D3）

- [x] 3.1 绿：内联 hex → 既有 token（映射见 design D3），无行为变更，存量 F2 守护

## 4. 收口

- [x] 4.1 全量验证：前端单测 + vue-tsc + `make gen-contract` 零漂移 + `make e2e-local`（frontend/ 改动强制，§6.2）
- [x] 4.2 覆盖率对齐棘轮（前端 74/71/67/74），补测后按需上调
- [ ] 4.3 delta spec sync 前自检 + code review + What/Why/How 提交整理
