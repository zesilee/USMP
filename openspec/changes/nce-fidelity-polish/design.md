# design — nce-fidelity-polish（NCE 保真微调）

## Context

渲染器现状（2026-07-09 探索实证）：`FieldRenderer.vue` 已映射 string/number/boolean/enum/group/presence/leaf-list/choice/list，enum 分支恒 `el-select clearable`（`:38-53`）；leaf-list 的枚举元素同样走 select（`:96`）。控件类型由后端 FieldDef 驱动（`field_gen.go` fieldType），FieldDef 无 Widget 字段——控件变体是前端启发式（同 `choiceUsesRadio`），widget 注解已在 P3'（ext-ui-annotations）决策推迟（零消费场景不自造词汇）。设计 token 单一事实源为 `variables.scss`（redesign 交付，主色 `#0C5EA6`），`theme.scss` 已有 el-table 配色覆盖但无密度覆盖。Element Plus 锁定 2.13.7，`main.ts` 全量注册。

## Goals / Non-Goals

**Goals:**
- 必填 ≤3 选项的枚举叶用分段控件（NCE §2.2 映射表），交互零弹层。
- 全站 el-table 收敛到 NCE 台账密度（高密度、留白克制）。
- FieldRenderer 消灭内联 hex，全部走 token。

**Non-Goals:**
- 不加 FieldDef/契约字段（widget 注解属 P3' 已推迟决策，不在本期重开）。
- 不做 leafref→transfer（契约无 leafref 类型，独立 feature）。
- 不换主色/不动 redesign 既定 token 值；S3 只做消费侧替换。
- 不动 leaf-list 的枚举元素控件（行内多值场景 segmented 占宽不适，保持 select）。

## Decisions

### D1 segmented 仅用于「必填 且 options ≤3」的标量 enum
`el-segmented` 无 clearable、必须有值；可选枚举叶需要「清空 = 键不入 payload」的能力（与 presence/undefined 语义一致），若强上 segmented 会让用户失去撤销选择的路径。故判定条件为 `field.required && field.options.length <= 3 && options.length > 0`；可选或 >3 保持 `el-select clearable`。阈值 3 来自 NCE 截图证据（Local/Third-party/Remote 三段）。此为算法兜底层启发式，将来 widget 注解落地时自然被「注解优先」覆盖。

### D2 密度走 theme.scss 全局覆盖，不引组件级 size 分叉
Element Plus 表格密度由 `--el-table-*-padding` 类变量与字号决定。在 `theme.scss` 的 el-table 覆盖块内统一收紧（th/td padding、`--el-font-size-base` 沿用），而非在每个组件上散点设置 `size="small"`——保证 DynamicTable/ModuleListTab/后续新表格自动继承，符合「新模块零前端代码」的通用控制台原则。若个别表格需回宽松密度，再局部放开（当前无此场景）。

### D3 S3 令牌映射表（近似灰阶，零视觉语义变更）
`#909399`→`var(--text-tertiary)`、`#606266`→`var(--text-secondary)`、`#f9fafb`→`var(--bg-elevated)`（组内嵌套底）、`#e5e7eb`→`var(--border-color)`。取既有 token 不新增；映射后由存量 F2 快照/断言与 F4 冒烟守护回归。

### D4 测试策略：F2 为主，兼容存量
新增 F2 用例断言 segmented 分支四态（必填≤3 渲染 segmented 并 emit、可选≤3 仍 select、>3 仍 select、readonly/disabled 透传）；存量以「enum→select」为前提的用例逐个核对，仅必填≤3 组合受影响者改写。el-segmented 是行内渲染（无 teleport），happy-dom 可测；若实测受限则该分支升 F3 真浏览器（§5.6 条款）。

## Risks / Trade-offs

- [启发式误伤：某必填 3 选枚举语义上更适合下拉] → 无契约可表达 per-field 意图（widget 注解已推迟）；NCE 证据支持默认 segmented，接受统一启发式，个例等注解机制。
- [全局密度收紧影响非配置页表格（如台账页）] → 台账风本就是 NCE 密度取向（调研 §2.1），全局一致是目标而非副作用；F4 staging 冒烟核对可读性。
- [存量 F2/F3 用例以 select 为前提挂红] → 视为预期红（映射规则变更），按新规则改写而非放宽断言。
- [el-segmented 在 happy-dom 下渲染异常] → 降级方案：该分支断言组件存在性+props（不断言内部 DOM），或升 F3；apply 首个任务先探测。

## Migration Plan

纯前端渐进增强，单 PR；回滚 = revert（enum 回落 select、密度回默认）。无数据迁移、无契约漂移（`make gen-contract` 零 diff 验证）。

## Open Questions

（无）
