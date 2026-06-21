# USMP 合规拦截体系 — 生效指引

## 快速生效（新成员必读）

```bash
# 1. 一键安装所有拦截层
make setup

# 2. 验证 hooks 激活
make hook-verify

# 3. 开始开发
#    - 新功能: EnterWorktree → /opsx:explore → /opsx:propose → /opsx:apply
#    - Bug修复: 创建分支 → TDD → PR
```

## 5 层拦截体系

| 层 | 机制 | 何时拦截 | 失败行为 |
|----|------|----------|----------|
| L1 | CLAUDE.md 强制规则 | AI Agent 生成代码时 | 软约束，AI 自动遵守 |
| L2 | Claude Code 命令拦截 | AI 执行 Bash 命令前 | 命令被阻止 |
| L3 | Git Hooks | git commit / git push 时 | exit 1，操作失败 |
| L4 | GitHub Actions CI | PR 提交时 | CI check 失败，无法合入 |
| L5 | GitHub Branch Protection | 合入 main 时 | GitHub 直接拒绝 |

## L3 Git Hooks 详解

### pre-commit（git commit 前触发）

| 检查项 | 规则编号 | 拦截条件 |
|--------|----------|----------|
| 敏感文件扫描 | R16 | `.env` / `.pem` / `.key` / `.p12` 等 |
| Generated 目录保护 | R04 | 修改 `internal/generated/` |
| Go vet | — | 有静态分析警告 |
| Go fmt | — | 有未格式化文件 |
| 变更包测试 | R15 | 变更的 Go 包测试失败 |

### commit-msg（commit message 写入前触发）

| 检查项 | 规则编号 | 拦截条件 |
|--------|----------|----------|
| Type 前缀 | 提交规范 | 不以 `feat/fix/docs/test/refactor/style/chore/perf:` 开头 |
| What/Why/How | 提交规范 | 缺少 `What:` / `Why:` / `How:` 任一段 |
| 代码量 | 提交规范 | 总变更 > 500 行 |

### pre-push（git push 前触发）

| 检查项 | 规则编号 | 拦截条件 |
|--------|----------|----------|
| main 分支保护 | R13 | 当前分支为 main/master |
| Force push 保护 | W07 | 使用 `--force` / `--force-with-lease` / `-f` |
| 全量测试 | R15 | `go test ./...` 失败 |

## L4 CI Workflows 详解

### compliance.yml（全量合规）

- `go test ./... -race -coverprofile`：全量测试 + 竞态检测 + 覆盖率
- 覆盖率守门：对比 `.coverage-baseline`，不下降
- `go vet ./...`：静态分析
- `gofmt -l .`：格式检查
- Generated 代码保护：PR 中检测 `internal/generated/` 变更

### commit-lint.yml（提交消息规范）

- PR 中所有 commit 必须符合 What/Why/How 三段式
- Type 前缀必须为 `feat|fix|docs|test|refactor|style|chore|perf`

### pr-size.yml（PR 体积限制）

- PR 总变更 ≤800 行（TM04），超出拆分

### branch-name.yml（分支命名规范）

- 格式：`<dev>/<change-name>` 或 `hotfix/<description>`

### sensitive-files.yml（敏感文件扫描）

- R16：`.env` / `.pem` / `.key` 等敏感文件
- 硬编码密码/密钥模式检测（warning 级别）

### openspec-check.yml（OpenSpec 制品检查）

- D01：关联 change 的 `proposal.md` + `design.md` + `tasks.md` 必须存在
- D02：`tasks.md` 至少有 1 个 `[x]` 标记

## L5 仓库保护配置（Maintainer 操作）

### Branch Protection Rules

GitHub → Settings → Branches → Branch protection rules → `main`

1. ☑ Require a pull request before merging
   - ☑ Require approvals: **1**
   - ☑ Dismiss stale reviews on push
2. ☑ Require status checks to pass
   - 选择: `compliance`, `commit-lint`, `pr-size`, `branch-name`, `sensitive-files`
   - ☑ Require branches to be up to date
3. ☑ Do not allow force pushes
4. ☑ Do not allow deletions

### CODEOWNERS

已在 `.github/CODEOWNERS` 配置分区审批规则：
- 默认：所有变更需 `@usmp-maintainers` 审批
- `internal/generated/`：必须 Maintainer 审批（R04）
- `pkg/yang-runtime/`：必须 Maintainer 审批（R01）
- 合规配置文件：必须 Maintainer 审批

> **注意**：需在 GitHub Organization 中创建 `usmp-maintainers` 团队并添加成员。

## 测试拦截是否生效

```bash
# 测试 pre-commit: 尝试提交敏感文件
echo "SECRET=xxx" > test.env
git add test.env
git commit -m "test"  # → pre-commit 拦截: [R16] 拒绝: 检测到敏感文件
rm test.env

# 测试 commit-msg: 尝试无格式提交
git commit -m "fix bug"  # → commit-msg 拦截: 缺少 What/Why/How

# 测试 pre-push: 尝试 push main（不要真的 push）
# 在 main 分支时: git push origin main → pre-push 拦截: [R13] 禁止直接 push main
```

## 绕过方式（仅紧急情况）

```bash
# 跳过 pre-commit 和 commit-msg
git commit --no-verify

# 跳过 pre-push
git push --no-verify
```

> ⚠️ `--no-verify` 仅用于 hotfix 等紧急场景。
> CI (L4) 和 Branch Protection (L5) 仍会拦截，`--no-verify` 无法绕过远端检查。
