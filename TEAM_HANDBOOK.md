# USMP 团队开发协作手册

## 1. 总览

本文档是 USMP 团队的人类协作指南，与 [CLAUDE.md](CLAUDE.md)（AI 执行规范）互补。

| 文档 | 定位 | 受众 |
|------|------|------|
| CLAUDE.md | AI 执行规范：红线 + 技能映射 + 工作流约束 | AI Agent |
| 本手册 | 人类协作指南：场景 + 流程 + 门禁 + 冲突处理 | 团队成员 |

### 适用范围

- 4-8 人中型团队，跨时区 / 远程协作
- 所有功能开发、Bug 修复、存量改造

### 核心原则

- **隔离开发**：每人在独立 worktree 工作，main 永远可部署
- **门禁驱动**：每步有明确准入条件，不满足不推进
- **异步优先**：评审 / 合入无需同步等待，PR 即协作单元

## 2. 角色与职责

| 角色 | 职责 | 人数 |
|------|------|------|
| Maintainer | main 分支守门人，PR 最终审批，架构决策，紧急 hotfix | 1-2 |
| Developer | 功能开发，提交 PR，参与评审 | 4-6 |
| Reviewer | PR 代码评审（轮值，非固定角色，由活跃 Developer 担任） | 当期活跃开发者 |

> Maintainer 不是审批瓶颈——任何 Developer 都可以担任 Reviewer。
> Maintainer 职责是 final approve（最终合入权）和架构红线把关（CLAUDE.md §2 R01-R12）。

## 3. 迭代完成统一标准（Definition of Done）

> 一个 change / feature 满足以下全部条件才算"完成"。

| 编号 | 标准 | 验证方式 |
|------|------|----------|
| D01 | OpenSpec 三制品齐全 | `proposal.md` + `design.md` + `tasks.md` 存在 |
| D02 | tasks.md 全部标记 `[x]` | 检查文件内容 |
| D03 | 全部测试通过 | `go test ./...` 零失败 |
| D04 | 代码评审通过 | ≥1 人 approve（非作者本人） |
| D05 | 主分支 rebase / merge 无冲突 | CI 验证 |
| D06 | 增量覆盖率未下降 | `go test -cover` 对比基线 |
| D07 | delta spec 已 sync 到主 spec | `/opsx:sync` 完成 |
| D08 | CLAUDE.md 红线无违反 | §2 R01-R12 全部合规 |
| D09 | change 已归档 | `/opsx:archive` 完成 |

## 4. 场景 1：领任务 & 开工

### 流程图

```
领取 change ──→ explore ──→ propose ──→ 建 worktree
    │              │           │            │
    │              │           │            └─ 命名: <developer>/<change-name>
    │              │           └─ 三制品: proposal + design + tasks
    │              └─ 审计存量，标记 legacy 边界
    └─ 确认无并行冲突（检查活跃 change）
```

### 4.1 领取 change

| 步骤 | 操作 | 门禁 |
|------|------|------|
| 1 | 在 openspec/ 下创建或认领 change | change 目录未被他人占用 |
| 2 | 执行 `/opsx:explore` | 禁止写代码 |
| 3 | 执行 `/opsx:propose` | 三制品齐全 |

### 4.2 并行冲突检查

| 检查项 | 方法 |
|--------|------|
| 同一 YANG 模块 | 查看 openspec/specs/ 下活跃 change，是否涉及相同 YANG path |
| 同一 Go package | 检查 tasks.md 中的文件路径列表 |
| 同一前端组件 | 检查 design.md 中的组件依赖图 |

> **规则**：两人不可并行修改同一 YANG 模块或同一 Go package。
> 冲突时：后到者等先到者归档，或拆分 change 使冲突区域归一人。

### 4.3 创建 worktree

| 步骤 | 操作 |
|------|------|
| 1 | 命名规范：`<developer>/<change-name>`（如 `alice/vlan-api`） |
| 2 | 从最新 main 创建：`EnterWorktree --name <name>` |
| 3 | 执行 `go test ./...` 确认基线 |
| 4 | 记录 worktree 与 change 对应关系（更新并行矩阵或频道通知） |

### 4.4 开工 Checklist

- [ ] change 目录已创建 / 认领
- [ ] 并行冲突检查通过
- [ ] explore 完成，存量边界已标记
- [ ] propose 完成，三制品齐全
- [ ] worktree 已创建，基线测试通过
- [ ] 团队已通知（频道 / 并行矩阵）

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
- [ ] CLAUDE.md 红线 R01-R12 无违反

## 6. 场景 3：代码评审

### 流程图

```
自审通过 ──→ push branch ──→ 创建 PR ──→ Reviewer 评审 ──→ approve/request-changes
                                                     │                │
                                                     │                └─ 作者修复 → 重新评审
                                                     └─ ≥1 人 approve → Maintainer final approve
```

### 6.1 提交 PR

| 项 | 规则 |
|----|------|
| 标题 | `<type>: <subject>` 与 commit 格式一致 |
| 描述 | 关联 change 目录、OpenSpec design.md 链接 |
| 体积 | 单 PR ≤800 行变更（超出拆分） |
| 基线 | 从最新 main rebase，无冲突 |
| CI | 全绿才能请求评审 |

### 6.2 Reviewer 评审清单

| 维度 | 检查项 |
|------|--------|
| 正确性 | 逻辑正确、边界条件、错误处理 |
| 架构合规 | §4 分层正确、C3 Reconciler 无状态 |
| 类型安全 | 无滥用 interface{}、ygot 结构体合规（R04） |
| 并发安全 | 无数据竞态、锁使用正确（R09） |
| 测试覆盖 | 新增代码有测试、覆盖率未下降 |
| 规范合规 | commit 格式、代码风格、无 AI 陈词滥调（R11） |

### 6.3 评审行为规范

| 规则 | 说明 |
|------|------|
| 24h 响应 | Reviewer 须在 24h 内完成首次评审 |
| 技术优先 | 意见必须附带技术理由，禁止纯主观评价 |
| 阻塞 vs 建议 | 阻塞问题标记 `BLOCK`，建议标记 `NIT`，作者必须处理所有 BLOCK |
| 推回权 | 作者可以技术理由推回不正确的评审意见 |
| 自审前置 | 提交 PR 前必须完成 §5.3 自审清单，Review 应验证非替代 |

### 6.4 Maintainer Final Approve

| 条件 | 说明 |
|------|------|
| ≥1 人 code approve | 非 PR 作者 |
| CI 全绿 | `go test ./...` + `go vet ./...` |
| 红线无违反 | CLAUDE.md §2 R01-R12 |
| 合入方式 | Maintainer squash merge 或 merge commit（团队约定） |

## 7. 场景 4：安全合入主干

### 流程图

```
PR approved ──→ Maintainer 合入 ──→ main 验证 ──→ 清理 worktree ──→ 通知团队
      │                │                │              │
      │                │                │              └─ 仅 Superpowers 创建的
      │                │                └─ CI 全绿 + 全量测试
      │                └─ squash merge 或 merge commit
      └─ ≥1 approve + CI 全绿
```

### 7.1 合入前门禁

| 编号 | 门禁 | 验证 |
|------|------|------|
| M01 | PR ≥1 人 approve（非作者） | GitHub / GitLab 状态 |
| M02 | CI 全绿 | 自动化检查 |
| M03 | 与 main 无冲突 | rebase 后 CI 验证 |
| M04 | 红线无违反（R01-R12） | Maintainer 目视 + go-code-review-check |
| M05 | 合入后 main 全量测试通过 | `go test ./...` |

### 7.2 合入操作

| 步骤 | 操作 |
|------|------|
| 1 | Maintainer 在 GitHub / GitLab 合入 PR |
| 2 | 本地 `git pull origin main` |
| 3 | 执行 `go test ./...` 验证 main |
| 4 | 执行 `/opsx:sync` 将 delta spec 合并到主 spec |
| 5 | 执行 `/opsx:archive` 归档 change |
| 6 | 清理 worktree（仅 Superpowers 创建的，参考 CLAUDE.md §6.4） |
| 7 | 通知团队：频道播报合入内容和影响范围 |

### 7.3 合入冲突处理

| 冲突类型 | 处理方式 |
|----------|----------|
| 同一文件不同区域 | 开发者 rebase main 解决，CI 验证后重新请求评审 |
| 同一文件同一区域 | 两开发者协商：一方先入，另一方 rebase 后适配 |
| OpenSpec spec 冲突 | `/opsx:sync` 按先入优先原则，后入者手动合并 delta |
| Go package 结构冲突 | Maintainer 裁决，可能需重构公共接口 |

### 7.4 紧急 Hotfix 通道

| 规则 | 说明 |
|------|------|
| 允许在 main 直接修复 | 但必须即时提交，不走 PR 流程 |
| 后补 PR | hotfix 提交后 24h 内补开 PR 供评审回顾 |
| 通知 | 频道立即通知所有成员 main 有 hotfix |
| rebase | 所有人 worktree 必须 rebase 最新 main |

## 8. 场景 5：收尾

### 收尾 Checklist

- [ ] D01-D09 全部满足（§3 迭代完成标准）
- [ ] main 全量测试通过
- [ ] delta spec 已 sync，主 spec 已更新
- [ ] change 已归档
- [ ] worktree 已清理（Superpowers 创建的）
- [ ] 团队已通知合入内容和影响范围
- [ ] 相关 issue 已关闭

### 收尾通知模板

```
✅ [change-name] 已合入 main
- 范围：[影响模块/包]
- PR：[#xxx](link)
- 注意：[其他人需 rebase 的变更/接口变更]
```

## 附录 A：分支命名规范

| 类型 | 格式 | 示例 |
|------|------|------|
| 功能开发 | `<dev>/<change-name>` | `alice/vlan-api` |
| Bug 修复 | `<dev>/fix-<issue>` | `bob/fix-cache-race` |
| 存量改造 | `<dev>/migrate-<module>` | `carol/migrate-ifm-actor` |
| Hotfix | `hotfix/<description>` | `hotfix/netconf-reconnect` |

## 附录 B：Worktree 并行矩阵

> 团队共享表，记录谁在哪个 worktree 做什么，防止并行冲突。

| Developer | Worktree | Change | YANG 模块 | Go Package | 状态 |
|-----------|----------|--------|-----------|------------|------|
| alice | `alice/vlan-api` | vlan-api | openconfig-vlan | controller/vlan | 开发中 |
| bob | `bob/device-status` | device-status | openconfig-interfaces | controller/interfaces | 评审中 |

> 新建 worktree 后必须更新此表（或频道通知），防止并行冲突。
> 表格维护方式由团队自选：共享文档 / 频道置顶 / Git 仓库内 markdown。

## 附录 C：常见问题

| 问题 | 解答 |
|------|------|
| 两人需改同一包怎么办？ | 拆分 change，冲突区域归一人；或一人先入，另一人 rebase 适配 |
| PR 超过 800 行怎么办？ | 拆分为多个 PR，每个对应一个原子功能 |
| Reviewer 超过 24h 未响应？ | 频道 @ 提醒，48h 无响应可指定备选 Reviewer |
| 合入后 main 测试失败怎么办？ | 立即 revert PR，开发者本地修复后重新提交 |
| worktree 过期怎么更新？ | `git rebase main` 或 `git merge main`，重新运行测试 |
| 存量改造双路径验证怎么做？ | 在 Reconciler 入口加 switch（旧 / 新），对比两次执行结果 |
| 并行矩阵怎么维护？ | 共享文档 / 频道置顶 / Git 仓库内 markdown，团队自选 |
| Maintainer 不在怎么办？ | Developer 完成评审后可临时合入，Maintainer 回来后回顾确认 |
