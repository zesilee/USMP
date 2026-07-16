# USMP — 开发规范

## §1 项目身份

| 维度 | 值 |
|------|------|
| 定位 | 无数据库、高并发、模型驱动的交换机设备管理平台 |
| 架构 | yang-controller-runtime 声明式配置管理 **[R01: 禁止更换]** |
| 语言 | Go 1.21+ / Vue3 |
| 协议 | NETCONF (SSH 830) + gNMI (9339/9340) **[R02: 禁止旧协议]** |

## §2 架构红线

> 违反任一条即视为不合规，禁止提交。

| 编号 | 红线 | 说明 |
|------|------|------|
| R01 | 禁止更换架构 | Manager→Controller→Reconciler→Source，禁止回退 Actor 模型 |
| R02 | 禁止旧协议 | 仅 NETCONF/gNMI，禁止 Telnet/SNMP |
| R03 | 禁止数据库 | 仅 TTL+LRU 内存缓存；持久元信息经 K8s CRD（apiserver，平台 etcd 承载）；禁止 MySQL/Redis/SQLite 等自管数据库，多实例部署禁止本地持久文件（SC-02/SC-06） |
| R04 | 禁止手写 YANG 结构体 | ygot 自动生成，禁止手写，禁止滥用 interface{} |
| R05 | 禁止手写固定表单 | 前端由 YANG 模型自动渲染 |
| R06 | 禁止先代码后测试 | TDD 红绿循环，测试先行 |
| R07 | 禁止合并流程 | OpenSpec→测试→代码→Review→Commit，不可跳过或合并 |
| R08 | 禁止崩溃 | 所有异常必须有降级处理 |
| R09 | 禁止数据竞态 | 协程安全、无内存泄漏、panic 防护 |
| R10 | 禁止无关依赖 | 不引入与项目无关的第三方库 |
| R11 | 禁止 AI 陈词滥调 | 紫粉蓝渐变、左边框圆角卡片、滥用 Inter/Roboto |
| R12 | 禁止 emoji 替代图标 | 无真实图标时使用规范占位符 |
| R13 | 禁止直接 push main | 使用 PR 合入，L2-L5 四层拦截兜底 |
| R14 | 禁止绕过 PR 合入 | CI required checks + 分支保护 |
| R15 | 禁止无测试提交 | pre-commit + CI 双重拦截 |
| R16 | 禁止提交敏感文件 | `.env`/`.pem`/`.key`/`.p12` 等，pre-commit + CI 扫描 |
| R17 | 禁止先代码后刷 spec | 契约变更 **spec-first**：先刷 spec/change delta 再启动开发，禁止先写代码后补 spec。spec 格式对齐 OpenSpec CLI 标准，详见 [openspec/SPEC_CONVENTIONS.md](openspec/SPEC_CONVENTIONS.md) |

## §3 技术栈

| 层 | 选型 | 依赖 | 约束 |
|----|------|------|------|
| 后端 | Go 1.21+ / yang-controller-runtime / Gin | ygot, scrapligo | §4 分层架构 |
| 模型 | YANG + ygot | openconfig/ygot | R04: 自动生成 |
| 协议 | NETCONF (SSH 830) + gNMI | RFC6241, openconfig/gnmi | R02: 禁止旧协议 |
| 缓存 | TTL+LRU 内存 | 协程安全 | R03: 无数据库, Key=IP+YANG路径, TTL 30s, 下发后失效 |
| 前端 | Vue3 + Element Plus | Axios, Pinia | R05: YANG 自动渲染, 编辑→提交→下发联动后端, 展示设备/缓存/下发/异常状态 |

## §4 yang-controller-runtime 分层

| 组件 | 职责 | 用户接口 |
|------|------|----------|
| C1 Manager | 全局生命周期：schema 加载、client 连接池、controller 注册、插件管理 | 启动/停止 |
| C2 Controller | 每 YANG 模块一个，处理事件队列，调用 Reconciler | 注册 Reconciler |
| C3 Reconciler | 对齐 desired↔actual（diff + 推送），无状态 | **用户实现此接口** |
| C4 EventSource | 产生 reconcile 事件：周期轮询 / gNMI 订阅 / 文件变更 | 注册 Source |
| C5 ClientPool | 设备连接池：断线重连、超时重试、异常处理 | 获取连接 |

> 框架处理所有 boilerplate（schema 解析、连接管理、diff 计算、协议编码、限频重试、事件排队）。用户只需实现 C3 Reconciler。

## §5 开发工作流

> 所有功能开发必须遵循此工作流，禁止跳过阶段。hotfix 允许在 main 操作但必须即时提交。

### 阶段总览

```
explore → propose → apply → sync → archive
   │         │         │        │        │
   │         │         │        │        └─ 归档 change
   │         │         │        └─ delta spec → 主 spec
   │         │         └─ worktree 内: 实现+测试+review+commit
   │         └─ 创建 change: proposal + design + tasks
   └─ 探索需求，禁止写代码
```

### 5.1 explore — 探索

| 项 | 值 |
|----|------|
| 命令 | `/opsx:explore` |
| 产出 | 需求澄清、架构映射、风险清单 |
| 门禁 | **禁止写代码** |
| 存量改造 | 必须审计存量代码，标记 `legacy` / `新架构` 边界，输出改造策略（渐进替换 / 并行运行 / 隔离封装） |

### 5.2 propose — 提案

| 项 | 值 |
|----|------|
| 命令 | `/opsx:propose` |
| 产出 | `proposal.md` + `design.md` + `tasks.md` |
| 门禁 | 三件制品齐全才能进入 apply |
| 存量改造 | tasks.md 须标注 `legacy→新架构` 迁移步骤，禁止一次性重写 |

### 5.3 apply — 实现（worktree 内）

| 项 | 值 |
|----|------|
| 命令 | `/opsx:apply` |
| 前置 | **必须进入 worktree 隔离**（§6） |
| 循环 | 按 tasks.md 逐项：写测试 → 写代码 → review → commit |
| 门禁 | 全部测试通过 + code review 通过 → 才能进入 §6.3 完成分支 |
| 代码量 | 单次输出 ≤500 行，超出拆分到下一迭代 |
| 存量改造 | 每步迁移必须：旧代码保留 + 新代码并行 + 双路径验证 → 切换 → 删除旧代码 |

### 5.4 sync — 同步

| 项 | 值 |
|----|------|
| 命令 | `/opsx:sync` |
| 产出 | delta spec 合并到主 spec |

### 5.5 archive — 归档

| 项 | 值 |
|----|------|
| 命令 | `/opsx:archive` |
| 产出 | change 移入归档目录 |

### TDD 规则（适用于 apply 阶段）

| 编号 | 规则 |
|------|------|
| T01 | 先写测试（正常/异常/并发），再写实现，**禁止先代码后测试** |
| T02 | 新增 YANG 模块必须添加 NETCONF 模拟网元集成测试（`*_integration_test.go`） |
| T02b | 新增 YANG 模型接入设备配置时**一定触发 `yang-config-test-design`**，产出并通过其完备测试矩阵（全属性可配/端到端到设备/并发-race/边界/嵌套/幂等/负路径），未覆盖视为未完成、禁止合并 |
| T03 | 集成测试用 `testing.Short()` 跳过短测试 |
| T04 | 代码评审不通过，禁止提交 |
| T05 | **测试设计先行**：写实现前先产出该改动的完备测试用例清单（对照 §5.6 选层 + 正常/异常/边界/并发/负路径），红灯先行。T01 推广到**所有改动类型**（不止 YANG） |
| T06 | **按层补齐**：每个改动按 §5.6「改动类型→必补层」补齐对应测试，**缺层=未完成、禁止合并**（T02b 的强制性推广到全部改动） |
| T07 | **Bug 必先回归**：修 Bug 前必先写复现该 Bug 的回归测试（红），再修（绿） |
| T08 | **覆盖率不下降**：后端 `backend/.coverage-baseline`、前端 `vitest` thresholds 为棘轮，低于基线 CI 即 fail；补测后同步上调阈值 |
| T09 | **本地门禁对称**：前端与后端一样有本地拦截——pre-commit 前端变更跑 happy-dom 单测、pre-push 全量 `-race` + 前端 e2e smoke |

> **§2 R15 补充**：「无测试提交」对前后端对称生效——后端变更包测试 + 前端 happy-dom 单测均在 pre-commit 本地拦截，CI 兜底。

### 5.6 测试分层与职责（军规）

> 每个改动**必须**按下表选层补齐测试，缺层即未完成（T06）。测试设计先于编码（T05）。

**后端 Go 层**

| 层 | 命名/位置 | 职责 |
|----|-----------|------|
| B1 单元/表格驱动 | `*_test.go`（`pkg/yang-runtime/*`、`internal/cache`、`internal/api` 编解码） | 纯逻辑：diff/cache/schema/parser，正常·异常·边界·**并发(race)** |
| B2 集成-模拟网元 | `*_integration_test.go` + `simulator/netconfsim`，`testing.Short()` 跳过 | Reconciler↔设备端到端：下发→回读→收敛 |
| B3 API/契约 | `internal/api/*_test.go` | HTTP handler、配置编解码、域约束、错误码 |
| ~~B0 退役遗留~~ | `backend/test/{e2e,integration}`（build-tag，Stack A/K8s CRD） | Stack A 遗留，不进任何门禁、不再维护 |

**前端层**

| 层 | 配置/位置 | 职责 |
|----|-----------|------|
| F1 纯逻辑单测 | happy-dom，`test/{utils,composables,stores}` | 纯函数/composable/store |
| F2 组件单测 | happy-dom，`test/{components,views}` + `@vue/test-utils` | 组件渲染·props·emit·分支（list/group 的 add/**edit/remove**、校验错误态） |
| F3 真浏览器 | 真 Chromium，`vitest.browser.config.ts`，`test/browser/` | **仅** happy-dom 伪造不了的：el-select 弹层/teleport、嵌套 list 真实交互（add/edit/remove 全覆盖） |
| F4 E2E | Playwright，`frontend/tests/staging-smoke.spec.ts` | 部署冒烟：路由·挂载·种子数据·YANG 表单渲染·校验拦截 |

**改动类型 → 必须补的测试层**

| 改动类型 | 必补层 |
|----------|--------|
| 后端纯逻辑 | B1（表格驱动 + race） |
| 新增/改 Reconciler·协议编解码 | B1 + **B2 集成** |
| 新增 YANG 模型接入设备配置 | **`yang-config-test-design` 完备矩阵**（T02b） |
| API handler/编解码 | B3（+B2 若涉下发） |
| **Bug 修复** | **先写复现回归测试（红）再修**（T07） |
| 前端 util/composable/store | F1 |
| 前端组件/页面逻辑 | F2（含 add/**edit/remove**/校验态，非只 render） |
| 前端 el-select/teleport/嵌套 list 增删改 | **F3 真浏览器** |
| 新页面/路由/端到端流 | F4 staging-smoke |

> 详见 [frontend/TESTING.md](frontend/TESTING.md)（前端分层权威规范）。

### 提交规范

| 项 | 规则 |
|----|------|
| 时机 | 原子功能完成 + 测试通过 → 立即提交，禁止积累 |
| 格式 | What/Why/How 三段式（`git-what-why-how-commit` 技能） |
| What | 明确变更的具体功能点/BUG 修复内容，不模糊、不冗余 |
| Why | 业务背景、解决的痛点、架构必要性，禁止无理由提交 |
| How | 技术实现逻辑、改动范围、核心交互流程，贴合本次 ≤500 行变更 |
| 范围 | 单次 Commit 仅对应一个原子功能 |

## §6 Worktree 安全隔离

> 新功能开发 **必须** 在 worktree 中进行，禁止在 main 上直接开发。
> hotfix 允许在 main 操作但必须即时提交。

### 6.1 创建 Worktree

| 步骤 | 操作 |
|------|------|
| 1 | 调用 `EnterWorktree`，每个 change/feature 对应一个 worktree |
| 2 | 验证 worktree 目录已在 `.gitignore` 中（避免污染 git status） |
| 3 | 执行项目基线测试，确认环境可用 |
| 4 | 记录 worktree 名称与 change 对应关系 |

### 6.2 开发中门禁

| 门禁 | 条件 |
|------|------|
| 测试通过 | `go test ./...` 全绿才能 commit |
| 代码评审 | `go-code-review-check` 技能通过 |
| 提交规范 | What/Why/How 三段式完整 |
| 前端 e2e smoke | **推送含 `frontend/` 改动时，提交代码前必须本地跑 Playwright staging smoke 全绿**：`make e2e-local`（起 docker 全栈→浏览器冒烟）。pre-push 钩子自动拦截，`USMP_SKIP_E2E=1` 仅限无 docker 机器显式跳过 |

### 6.3 完成分支

> 开发完成、测试全绿后，**必须**执行完成分支流程（`superpowers:finishing-a-development-branch`），禁止直接 merge/push。

| 步骤 | 操作 |
|------|------|
| 1 | **验证测试**：`go test ./...` 全绿，否则禁止继续 |
| 2 | **检测环境**：判断 normal repo / named-branch worktree / detached HEAD |
| 3 | **选择合入方式** |

| 选项 | 操作 | 保留 worktree | 删除分支 |
|------|------|---------------|----------|
| A. 本地合并 | merge 到 main → 验证测试 → 删除 worktree → 删除分支 | ❌ | ✅ |
| B. 推送+PR | push -u origin → 创建 PR | ✅ | ❌ |
| C. 保持现状 | 保留分支和 worktree | ✅ | ❌ |
| D. 丢弃 | 确认后强制删除 | ❌ | ✅(force) |

| 4 | **清理 worktree**（仅 A/D 选项） |

### 6.4 Worktree 清理规则

| 条件 | 操作 |
|------|------|
| worktree 路径在 `.claude/worktrees/` 下 | Superpowers 创建 → 本工具负责清理 |
| worktree 路径在其他位置 | 外部环境创建 → 禁止删除，使用 ExitWorktree |
| 删除前 | 必须 `cd` 到主仓库根目录 |
| 删除后 | 执行 `git worktree prune` 清理过期注册 |

### 6.5 安全红线

| 编号 | 红线 |
|------|------|
| W01 | 禁止在 main 上开发新功能 |
| W02 | 禁止测试未通过就合入 |
| W03 | 禁止从 worktree 内部执行 `git worktree remove` |
| W04 | 禁止合并成功前删除 worktree |
| W05 | 禁止未经确认执行丢弃（需输入 'discard' 确认） |
| W06 | 禁止清理非自己创建的 worktree（路径溯源） |
| W07 | 禁止 force push 任何分支（hotfix 除外需 Maintainer 确认） |

## §7 技能映射

> 触发时 **必须** 调用对应技能，禁止跳过。

### 7.1 后端技能

| 触发场景 | 技能 | 说明 |
|----------|------|------|
| 新 YANG 控制器开发 | `yang-controller-runtime-dev` | 架构合规（§4） |
| YANG→Go 结构体 | `yang-ygot-generate` | 自动生成（R04） |
| 配置缓存开发 | `go-ttl-lru-memory-cache` | TTL+LRU 并发安全（R03） |
| NETCONF 对接 | `netconf-switch-protocol` | SSH 830（R02） |
| 集成测试 | `netconf-sim-integration-test` | 模拟网元端到端（T02） |
| **新增 YANG 模型接入设备配置** | **`yang-config-test-design`** | **一定触发：产出完备测试用例矩阵（全属性/端到端/并发/边界/嵌套/幂等/负路径），未覆盖禁止合并（T02b）** |
| TDD 开发 | `tdd-test-driven-dev` | 测试先行（T01） |
| 代码评审 | `go-code-review-check` | 提交前强制（T04） |
| 提交规范 | `git-what-why-how-commit` | 三段式 Commit |

### 7.2 前端技能

| 触发场景 | 技能 | 规则 |
|----------|------|------|
| 功能型（YANG 驱动表单/动态渲染） | `frontend-yang-dynamic-form` | YANG 类型自动映射：boolean→开关、enum→下拉、list→表格（R05） |
| 视觉型（美化/可视化/交互原型） | `web-design-engineer` | 先声明设计系统→v0 草案→≥2 变体 |
| 纯逻辑/工程化/纯功能 | **不触发设计技能** | 状态管理/API/构建/校验/路由/权限 |
| 前端测试 | 按 §5.6 选层：util/composable/store→F1、组件/页面→F2、el-select/嵌套 list 交互→F3 真浏览器、端到端流→F4 | 缺层禁止合并（T06）；详见 [frontend/TESTING.md](frontend/TESTING.md) |

### 7.3 Superpowers 技能

| 触发场景 | 技能 | 说明 |
|----------|------|------|
| 任何创造性工作前 | `superpowers:brainstorming` | 探索意图→设计→审批 |
| 功能开发开始 | `superpowers:using-git-worktrees` | §6 隔离环境 |
| 实施计划执行 | `superpowers:executing-plans` | 按计划逐步执行 |
| 多任务并行 | `superpowers:subagent-driven-development` | 独立子任务并行 |
| 开发完成 | `superpowers:finishing-a-development-branch` | §6.3 完成分支 |
| Bug/测试失败 | `superpowers:systematic-debugging` | 根因分析优先 |
| TDD 实现 | `superpowers:test-driven-development` | 红绿循环 |
| 声称完成前 | `superpowers:verification-before-completion` | 必须有新鲜验证证据 |
| 编写实施计划 | `superpowers:writing-plans` | 从规格到可执行计划 |

## §8 数据存储

| 数据类型 | 存储方式 | 生命周期 |
|----------|----------|----------|
| 运行配置 | 实时 NETCONF/gNMI 从交换机读取 | 缓存 TTL 30s，过期自动重拉 |
| 配置缓存 | TTL+LRU 内存 | Key=设备IP+YANG路径，下发后主动失效 |
| 元信息（业务意图/认领/收敛状态） | K8s CRD（apiserver；CRD 仅当持久化+watch 载体，禁止当架构通道） | 持久 |
| 操作审计 | K8s CRD（AuditRecord，每条一 CR；无集群降级内存） | 持久（上限 1000 条滚动） |

> **R03: 禁止数据库** — 不持久化运行配置，不使用 MySQL/Redis/SQLite 等自管数据库；持久元信息走 K8s CRD（2026-07-15 拍板：USMP 为 K8s 内多实例 PaaS 组件，禁止本地存储）。

## §9 异常处理

| 异常场景 | 处理策略 | 前端表现 |
|----------|----------|----------|
| 设备离线 | NETCONF 自动重连 | 展示离线状态，API 返回明确错误 |
| 缓存过期 | 自动重新拉取配置 | 静默更新 |
| Controller 故障 | Manager 自动重启，隔离其他模块 | 部分功能降级 |
| NETCONF 下发失败 | 前端提示错误，缓存不更新 | 保留原配置 |
| 前端表单校验失败 | 不提交，展示 YANG 约束提示 | 行内校验提示 |

> **R08: 禁止崩溃** — 所有异常必须有降级处理。

## §10 交付标准

| 维度 | 标准 |
|------|------|
| 后端 | 可运行 Go 项目：Controller 系统 + API 接口 + NETCONF 对接 |
| 前端 | 可运行 Vue3 项目：动态表单 + 树形菜单 + 配置下发 |
| 测试 | 单元 + 异常 + 并发 + NETCONF 模拟网元集成测试 |
| 合规 | 满足 §2 全部红线，无违规代码 |

## §11 开发协作

> 详见 [TEAM_HANDBOOK.md](TEAM_HANDBOOK.md) — 开发流程、自审清单、安全合入完整指南。
> 个人项目：无他人审批，用结构化自审 + CI 自动化替代人工评审。

| 编号 | 规则 |
|------|------|
| TM01 | 合入 main 须经 PR + CI 全绿 + 自审清单通过 |
| TM02 | 分支命名：`<change-name>` 或 `hotfix/<desc>` |
| TM03 | 多 worktree 不可修改同一 Go package 或 YANG 模块 |
| TM04 | PR 体积 ≤1000 行，超出拆分 |
| TM05 | 自审清单（§6.2）全部 ✓ 才可合入 |
| TM06 | hotfix 允许 main 直修但须补 PR 供 CI 验证 |
| TM07 | 迭代完成须满足 D01-D09 全部标准（见手册 §3） |
| TM08 | `make setup` 为 clone 后必执行步骤，一键激活全部拦截层 + 依赖 + 基线测试 |
| TM09 | CI required checks 全通过才可合入（compliance + commit-lint + pr-size + branch-name + sensitive-files + openspec-check） |
| TM10 | OpenSpec 项目配置固化在 `openspec/.openspec.yaml`，克隆自动可用 |

## §12 会话恢复

> 任务持久化到 `openspec/tasks/`，支持跨会话和跨机器续做。

| 编号 | 规则 |
|------|------|
| SR01 | 新会话启动时，检查 `openspec/tasks/` 是否有 `status: in_progress` 的任务，如有则提示运行 `/task resume <slug>` |
| SR02 | 任务完成后，运行 `/task archive <slug>` 归档 |
| SR03 | 任务切换时，先 `/task sync` 保存当前状态，再切换 |
| SR04 | `/task list` 随时可查看所有任务状态 |
| SR05 | `/task resume <slug>` 恢复任务时，自动读取上下文恢复提示和恢复指令 |
