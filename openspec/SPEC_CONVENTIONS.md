# USMP Spec 规范

> 本文件是 USMP 所有 spec（主 spec + change delta spec）的**唯一格式权威**。
> 与 OpenSpec CL(`@fission-ai/openspec`) 对齐，保证 `openspec validate / sync / archive` 全链可用。

## §0 军规（对应 CLAUDE.md R17）

> **刷 spec 先于开发（spec-first）。**
>
> 代码变更若需要新增/修改行为契约，**必须先刷新 spec（或提交 change delta），再启动开发**。
> 禁止先写代码、后补 spec。违反即视为不合规，禁止提交。
>
> - 新能力/契约变更 → 走 `/opsx:propose` 产出 change delta（ADDED/MODIFIED/REMOVED）→ apply。
> - 已有 spec 与实现不一致 → 先修 spec，再改代码对齐。
> - 纯 bug 修复/重构、不改契约 → 无需刷 spec。

## §1 主 spec 标准格式

每个能力（capability）一个目录：`openspec/specs/<capability>/spec.md`。文件结构**必须**为：

```markdown
# <capability> — <一句话标题>

## Purpose

<该能力存在的目的，1–3 句。说明它对系统提供什么行为契约。>

## Requirements

### Requirement: <XX-NN> <简短陈述>

<该需求的 SHALL 陈述（RFC2119 关键词）。一段说明"必须满足什么"。>

#### Scenario: <场景名>

- **WHEN** <触发条件>
- **THEN** <系统 SHALL <可验证结果>>

#### Scenario: <另一个场景（异常/边界）>

- **WHEN** ...
- **THEN** ...
```

**硬性约束**（`openspec validate <cap> --type spec` 强制）：
- **必须**有 `## Purpose` 和 `## Requirements` 两个二级标题。
- 每个 `### Requirement:` **必须**至少含一个 `#### Scenario:`。
- 每个 Scenario **必须**有 `WHEN` / `THEN`。

## §2 Requirement 写法

- 标题：`### Requirement: <XX-NN> <陈述>`。**保留** XX-NN 编号作前缀（如 `AT-01`/`BR-01`/`SC-01`）用于跨文档追溯。
- 正文用 **RFC2119** 关键词，大写：
  - **SHALL / SHALL NOT** — 强制，测试/CI 必须覆盖。
  - **SHOULD / SHOULD NOT** — 推荐，偏离需理由。
  - **MAY** — 可选。
- 一个 Requirement 聚焦一条契约；组合行为拆成多个 Requirement 或多个 Scenario。
- 契约要**可验证**：能映射到一个测试或一段可观察行为。

## §3 Scenario 写法

- `#### Scenario: <名>`，名字点出场景意图（正常/离线/超时/并发/边界/幂等/负路径）。
- 用 `- **WHEN** …` / `- **THEN** …` 列项；可加 `- **AND** …`。
- THEN 内嵌 SHALL，指向**可观察的结果**（返回码、状态、落盘、降级表现）。
- 覆盖面对齐 §7.1 测试矩阵（T02b）：全属性/端到端/并发-race/边界/嵌套/幂等/负路径。

## §4 Change delta spec 格式

change 内的 delta：`openspec/changes/<change>/specs/<capability>/spec.md`。用**增量头**：

```markdown
## ADDED Requirements
### Requirement: <XX-NN> <新增契约>
#### Scenario: ...

## MODIFIED Requirements
### Requirement: <与主 spec 中完全一致的既有标题>
<修改后的完整内容 + Scenario>

## REMOVED Requirements
### Requirement: <被删除的既有标题>
```

- **新增需求用 `## ADDED Requirements`**。切勿对主 spec 里不存在的 requirement 用 `MODIFIED`——`openspec archive` 按标题精确匹配既有 requirement，找不到会失败。
- `MODIFIED`/`REMOVED` 的 `### Requirement:` 标题**必须与主 spec 中的完全一致**（含 XX-NN 前缀）。

## §5 校验与工作流

- 单个主 spec：`openspec validate <capability> --type spec`
- 单个 change：`openspec validate <change-name>`
- 全部：`openspec validate --all`（**迁移完成前不会全绿**，属预期）
- CI（openspec-check）**只校验本次 PR 改动的**主 spec / change，未迁移的旧 spec 过渡期不卡（避免全红）。
- 工作流：`propose`（产 delta）→ `apply` → `sync`（delta 合入主 spec）→ `archive`（change 归档 + 更新主 spec）。archive 依赖主 spec 已是本标准格式。

## §6 迁移状态（渐进）

12 个既有主 spec 由自定义「行为契约（反向还原）」格式**渐进迁移**到本标准，**逐能力一个 PR**（禁一次性重写）。迁移时**借机修正 spec 与代码不一致**处（非忠实照搬）。迁移进度见对应 change 的 tasks。未迁移的 spec 暂留旧格式，CI 不卡。

| 状态 | 含义 |
|------|------|
| ✅ 已迁移 | `openspec validate <cap> --type spec` 通过 |
| ⏳ 待迁移 | 仍为旧「行为契约」格式，过渡豁免 |
