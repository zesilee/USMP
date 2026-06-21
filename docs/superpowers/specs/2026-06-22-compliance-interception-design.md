# 多层合规拦截体系设计

> 日期: 2026-06-22
> 状态: Approved

## 目标

全自动搭建项目多层合规拦截体系，实现开发不按规范直接强制拦截，杜绝违规提交与野蛮合入。

## 方案

**方案 A：5 层拦截金字塔** — CLAUDE.md 规则层 → Claude Code 命令拦截层 → Git Hooks 本地拦截层 → GitHub Actions CI 拦截层 → 仓库保护层。

## 关键设计决策

1. **Git Hooks 方案**：原生 Git Hooks（.githooks/ + core.hooksPath），零依赖
2. **CI 平台**：GitHub Actions
3. **拦截强度**：强制拦截，不合规直接 exit 1

## 5 层拦截体系

| 层 | 机制 | 反馈速度 | 拦截力度 |
|----|------|----------|----------|
| L1 | CLAUDE.md 强制规则 | 纳秒（AI 读取即遵守） | 软约束 |
| L2 | Claude Code hooks（命令拦截） | 毫秒 | 硬约束 |
| L3 | Git Hooks（pre-commit/commit-msg/pre-push） | 秒 | 硬约束 |
| L4 | GitHub Actions CI | 分钟 | 硬约束 |
| L5 | GitHub Branch Protection + CODEOWNERS | 永久 | 硬封锁 |

## 交付文件清单

| 文件 | 层 | 说明 |
|------|-----|------|
| `.githooks/pre-commit` | L3 | 敏感文件扫描 + generated 保护 + go vet + go fmt + 测试 |
| `.githooks/commit-msg` | L3 | 三段式格式校验 + type 前缀 + 代码量 ≤500 行 |
| `.githooks/pre-push` | L3 | main 分支保护 + force push 保护 + 全量测试 |
| `.claude/settings.json` (更新) | L2 | 危险命令拦截 hooks |
| `.github/workflows/compliance.yml` | L4 | 全量测试 + vet + fmt + 覆盖率 |
| `.github/workflows/commit-lint.yml` | L4 | commit message 三段式 |
| `.github/workflows/pr-size.yml` | L4 | PR 体积 ≤800 行 |
| `.github/workflows/branch-name.yml` | L4 | 分支命名规范 |
| `.github/workflows/sensitive-files.yml` | L4 | 敏感文件扫描 |
| `.github/workflows/openspec-check.yml` | L4 | OpenSpec 三制品检查 |
| `.github/CODEOWNERS` | L5 | 分区审批规则 |
| `Makefile` | 辅助 | setup/test/lint/compliance/hook-install/hook-verify |
| `CLAUDE.md` (更新) | L1 | R13-R16 + W07 + TM08-TM09 |
| `docs/compliance/SETUP_GUIDE.md` | 辅助 | 一键生效指引 |

## 新增规则编号

| 前缀 | 新增 | 说明 |
|------|------|------|
| R13-R16 | 4 条 | 禁止 push main / 绕过 PR / 无测试提交 / 敏感文件 |
| W07 | 1 条 | 禁止 force push |
| TM08-TM09 | 2 条 | make setup 必执行 / CI required checks 必通过 |

## 自审检查

- Placeholder scan: 无 TBD/TODO ✅
- Internal consistency: L2-L5 每层检查项不重复，互补兜底 ✅
- Scope check: 聚焦于拦截体系，单一目标 ✅
- Ambiguity check: 每个检查项有具体匹配模式和 exit 条件 ✅
