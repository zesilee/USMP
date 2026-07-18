# USMP 开发协作手册

## 1. 总览

本文档是 USMP 项目的人类协作指南，与 [CLAUDE.md](CLAUDE.md)（AI 执行规范）互补。

| 文档 | 定位 | 受众 |
|------|------|------|
| CLAUDE.md | AI 执行规范：红线 + 技能映射 + 工作流约束 | AI Agent |
| 本手册 | 开发指南：场景 + 流程 + 门禁 + 自审 | 开发者（当前为个人项目） |

### 适用范围

- 个人项目，可扩展至小团队
- 所有功能开发、Bug 修复、存量改造

### 核心原则

- **隔离开发**：功能在独立 worktree 开发，main 永远可部署
- **门禁驱动**：每步有明确准入条件，不满足不推进
- **自审替代他审**：个人项目无他人审批，用结构化自审 + CI 自动化替代人工评审

## 2. 角色与职责

| 角色 | 职责 | 当前状态 |
|------|------|----------|
| Owner | main 分支守门人，架构决策，紧急 hotfix，代码自审 | 唯一开发者 |
| CI | 自动化评审：测试 + lint + 格式 + 合规检查 | 替代人工 Reviewer |

> 个人项目无他人审批，依赖 **L3 Git Hooks + L4 CI** 自动拦截替代人工门禁。
> 代码评审转为 **结构化自审**（§6.3 清单），CI 全绿视为自动 approve。

## 3. 迭代完成统一标准（Definition of Done）

> 一个 change / feature 满足以下全部条件才算"完成"。

| 编号 | 标准 | 验证方式 |
|------|------|----------|
| D01 | OpenSpec 三制品齐全 | `proposal.md` + `design.md` + `tasks.md` 存在 |
| D02 | tasks.md 全部标记 `[x]` | 检查文件内容 |
| D03 | 全部测试通过 | `go test ./...` 零失败 |
| D04 | 代码自审通过 | 完成自审清单（§6.3） |
| D05 | CI 全绿 | 所有 GitHub Actions check 通过 |
| D06 | 增量覆盖率未下降 | 后端 `backend/.coverage-baseline`、前端 `vitest` thresholds 双棘轮（低于基线 CI fail，T08） |
| D07 | delta spec 已 sync 到主 spec | `/opsx:sync` 完成 |
| D08 | CLAUDE.md 红线无违反 | §2 R01-R16 全部合规 |
| D09 | change 已归档 | `/opsx:archive` 完成 |

## 4. 场景 1：领任务 & 开工

### 流程图

```
领取 change ──→ explore ──→ propose ──→ 建 worktree
    │              │           │            │
    │              │           │            └─ 命名: <change-name>
    │              │           └─ 三制品: proposal + design + tasks
    │              └─ 审计存量，标记 legacy 边界
    └─ 确认无并行冲突（检查活跃 change）
```

### 4.1 领取 change

| 步骤 | 操作 | 门禁 |
|------|------|------|
| 1 | 在 openspec/ 下创建 change | — |
| 2 | 执行 `/opsx:explore` | 禁止写代码 |
| 3 | 执行 `/opsx:propose` | 三制品齐全 |

### 4.2 并行冲突检查

| 检查项 | 方法 |
|--------|------|
| 同一 YANG 模块 | 查看 openspec/specs/ 下活跃 change，是否涉及相同 YANG path |
| 同一 Go package | 检查 tasks.md 中的文件路径列表 |
| 同一前端组件 | 检查 design.md 中的组件依赖图 |

> **规则**：多个 worktree 不可同时修改同一 Go package 或 YANG 模块。
> 冲突时：一个 change 先入，另一个 rebase 适配。

### 4.3 创建 worktree

| 步骤 | 操作 |
|------|------|
| 1 | 命名规范：`<change-name>`（如 `vlan-api`、`fix-cache-race`） |
| 2 | 从最新 main 创建：`EnterWorktree --name <name>` |
| 3 | 执行 `go test ./...` 确认基线 |

### 4.4 开工 Checklist

- [ ] change 目录已创建
- [ ] 并行冲突检查通过
- [ ] explore 完成，存量边界已标记
- [ ] propose 完成，三制品齐全
- [ ] worktree 已创建，基线测试通过

## 5. 场景 2：开发 & 自审

### 流程图

```
apply tasks.md ──→ 写测试(红) ──→ 写代码(绿) ──→ 重构 ──→ 自审 ──→ commit
      │                │              │             │         │
      │                │              │             │         └─ go-code-review-check
      │                │              │             └─ 不破坏测试
      │                │              └─ 最小代码通过测试
      │                └─ 先失败再实现
      └─ 逐项 task，标记 [x]
```

### 5.1 TDD 循环

| 阶段 | 操作 | 门禁 |
|------|------|------|
| 红 | 写测试，确认失败 | 测试编译通过但断言失败 |
| 绿 | 写最小实现，确认通过 | `go test ./...` 零失败 |
| 重构 | 消除重复，不改行为 | 重构后测试仍全绿 |
| commit | What/Why/How 三段式 | 单次 ≤500 行 |

### 5.2 存量改造专用规则

| 步骤 | 操作 | 验证 |
|------|------|------|
| 1 | 旧代码保留，新增新架构代码 | 旧路径仍可走通 |
| 2 | 新旧并行运行，双路径验证 | 对比结果一致 |
| 3 | 切换入口到新架构 | 功能回归测试通过 |
| 4 | 删除旧代码 | 编译 + 测试通过 |

### 5.3 自审 Checklist

- [ ] `go test ./...` 全绿
- [ ] `go vet ./...` 零警告
- [ ] 无新增 TODO / FIXME（必须创建 issue 替代）
- [ ] 无硬编码配置（IP / 端口 / 密码）
- [ ] 新增代码有对应测试
- [ ] CLAUDE.md 红线 R01-R16 无违反

## 6. 场景 3：代码自审 & PR

### 流程图

```
自审通过 ──→ push branch ──→ 创建 PR ──→ CI 自动检查 ──→ 全绿 → Owner 自审合入
                                                     │
                                                     └─ CI 失败 → 修复 → 重新提交
```

### 6.1 提交 PR

| 项 | 规则 |
|----|------|
| 标题 | `<type>: <subject>` 与 commit 格式一致 |
| 描述 | 关联 change 目录、OpenSpec design.md 链接 |
| 体积 | 单 PR ≤1000 行变更（超出拆分） |
| 基线 | 从最新 main rebase，无冲突 |
| CI | 全绿才能合入 |

### 6.2 代码自审清单

> 个人项目无他人评审，用以下结构化清单 + CI 自动化替代。

| 维度 | 检查项 | 自动化 |
|------|--------|--------|
| 正确性 | 逻辑正确、边界条件、错误处理 | 人工 |
| 架构合规 | §4 分层正确、C3 Reconciler 无状态 | 人工 + go-code-review-check |
| 类型安全 | 无滥用 interface{}、ygot 结构体合规（R04） | CI: generated 目录保护 |
| 并发安全 | 无数据竞态、锁使用正确（R09） | CI: go test -race |
| 测试覆盖 | 按 [CLAUDE.md §5.6](CLAUDE.md) 改动类型选层补齐（缺层=未完成，T06）、覆盖率未下降 | CI: 后端 baseline + 前端 vitest thresholds 守门；本地 pre-commit（前后端单测）+ pre-push（-race + e2e） |
| 规范合规 | commit 格式、代码风格 | CI: commit-lint + gofmt |

### 6.3 合入条件

| 条件 | 说明 |
|------|------|
| 自审清单通过 | §6.2 全部 ✓ |
| CI 全绿 | `go test ./...` + `go vet ./...` + 所有 workflow |
| 红线无违反 | CLAUDE.md §2 R01-R16 |
| 合入方式 | squash merge 或 merge commit |

## 7. 场景 4：安全合入主干

### 流程图

```
CI 全绿 ──→ Owner 自审 ──→ 合入 main ──→ main 验证 ──→ 清理 worktree
    │            │              │              │
    │            │              │              └─ 仅 Superpowers 创建的
    │            │              └─ squash merge 或 merge commit
    │            └─ 对照自审清单
    └─ 所有 workflow 通过
```

### 7.1 合入前门禁

| 编号 | 门禁 | 验证 |
|------|------|------|
| M01 | CI 全绿 | 所有 GitHub Actions check 通过 |
| M02 | 与 main 无冲突 | rebase 后 CI 验证 |
| M03 | 红线无违反（R01-R16） | 自审 + go-code-review-check |
| M04 | 合入后 main 全量测试通过 | `go test ./...` |

### 7.2 合入操作

| 步骤 | 操作 |
|------|------|
| 1 | 在 GitHub 合入 PR |
| 2 | 本地 `git pull origin main` |
| 3 | 执行 `go test ./...` 验证 main |
| 4 | 执行 `/opsx:sync` 将 delta spec 合并到主 spec |
| 5 | 执行 `/opsx:archive` 归档 change |
| 6 | 清理 worktree（仅 Superpowers 创建的，参考 CLAUDE.md §6.4） |

### 7.3 合入冲突处理

| 冲突类型 | 处理方式 |
|----------|----------|
| 同一文件不同区域 | rebase main 解决，CI 验证后重新提交 |
| 同一文件同一区域 | 手动解决冲突，保留正确逻辑 |
| OpenSpec spec 冲突 | `/opsx:sync` 手动合并 delta |
| Go package 结构冲突 | 先重构公共接口，再合入 |

### 7.4 紧急 Hotfix 通道

| 规则 | 说明 |
|------|------|
| 允许在 main 直接修复 | 但必须即时提交，不走 PR 流程 |
| 后补 PR | hotfix 提交后补开 PR 供 CI 验证 |
| 回归验证 | hotfix 后立即 `go test ./...` |

## 8. 场景 5：收尾

### 收尾 Checklist

- [ ] D01-D09 全部满足（§3 迭代完成标准）
- [ ] main 全量测试通过
- [ ] delta spec 已 sync，主 spec 已更新
- [ ] change 已归档
- [ ] worktree 已清理（Superpowers 创建的）
- [ ] 相关 issue 已关闭

## 附录 A：分支命名规范

| 类型 | 格式 | 示例 |
|------|------|------|
| 功能开发 | `<change-name>` | `vlan-api` |
| Bug 修复 | `fix-<issue>` | `fix-cache-race` |
| 存量改造 | `migrate-<module>` | `migrate-ifm-actor` |
| Hotfix | `hotfix/<description>` | `hotfix/netconf-reconnect` |

## 附录 B：Worktree 跟踪表

> 记录当前活跃的 worktree，防止并行冲突。

| Worktree | Change | YANG 模块 | Go Package | 状态 |
|----------|--------|-----------|------------|------|
| `retire-openconfig-models` | retire-openconfig-models | （删 openconfig 载体） | yangschema / client / api | 开发中 |

> 新建 worktree 后更新此表，防止多个 worktree 修改同一包。

## 附录 C：常见问题

| 问题 | 解答 |
|------|------|
| 多个 worktree 改同一包怎么办？ | 一个先入，另一个 rebase 适配 |
| PR 超过 1000 行怎么办？ | 拆分为多个 PR，每个对应一个原子功能 |
| 合入后 main 测试失败怎么办？ | 立即 revert PR，本地修复后重新提交 |
| worktree 过期怎么更新？ | `git rebase main` 或 `git merge main`，重新运行测试 |
| 存量改造双路径验证怎么做？ | 在 Reconciler 入口加 switch（旧 / 新），对比两次执行结果 |
| 个人项目怎么保证代码质量？ | 结构化自审清单（§6.2）+ CI 全自动检查 + L3 Git Hooks 拦截 |
