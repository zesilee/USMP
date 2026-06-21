# 团队开发协作手册设计

> 日期: 2026-06-22
> 状态: Approved

## 目标

生成项目多人团队开发协作手册，融合 OpenSpec 设计规范与 Superpowers worktree 开发流程，定义迭代完成统一标准，制定多人并行开发、代码评审、安全合并主干完整流程。

## 方案

**方案 A：场景驱动流程手册** — 以"团队成员一天的工作场景"为主线组织，每个场景有流程图+门禁清单+冲突处理规则。

## 关键设计决策

1. **团队规模**：4-8 人中型团队，跨时区/远程协作
2. **手册定位**：独立文档 TEAM_HANDBOOK.md + CLAUDE.md 新增 §11 引用指针
3. **合入策略**：PR + ≥1 人 approve + CI 全绿
4. **组织方式**：场景驱动（领任务→开发→评审→合入→收尾）

## 手册结构

```
1.  总览           — 与 CLAUDE.md 分工、适用范围、核心原则
2.  角色与职责      — Maintainer / Developer / Reviewer
3.  迭代完成标准(DoD) — D01-D09 九条可验证标准
4.  场景1: 领任务&开工 — change 认领、并行冲突检查、worktree 创建
5.  场景2: 开发&自审  — TDD 循环、存量改造规则、自审清单
6.  场景3: 代码评审   — PR 提交规范、Reviewer 清单、行为规范、Maintainer final approve
7.  场景4: 安全合入主干 — 合入门禁 M01-M05、冲突处理、hotfix 通道
8.  场景5: 收尾      — 收尾 checklist、通知模板
附录 A: 分支命名规范
附录 B: Worktree 并行矩阵
附录 C: 常见问题 FAQ
```

## 编号系统

| 前缀 | 域 | 条数 |
|------|------|------|
| D01-D09 | 迭代完成标准 | 9 |
| M01-M05 | 合入门禁 | 5 |
| TM01-TM07 | CLAUDE.md 团队协作指针 | 7 |

## CLAUDE.md 更新

在 §10 后新增 §11 团队协作，包含 TM01-TM07 七条精简指针，引用 TEAM_HANDBOOK.md。

## 与现有体系的关系

| 文档 | 定位 | 受众 |
|------|------|------|
| CLAUDE.md | AI 执行规范，红线+技能映射 | AI Agent |
| TEAM_HANDBOOK.md | 人类协作指南，场景+流程+门禁 | 团队成员 |
| OpenSpec skills | 规范化设计流程 | AI Agent + 人类 |
| Superpowers skills | 工作树+评审+完成分支 | AI Agent |
