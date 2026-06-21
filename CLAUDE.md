# USMP - Claude Code 开发规范

## 项目定位
无数据库、高并发、模型驱动的交换机设备管理平台，基于 yang-controller-runtime 实现声明式配置管理。**禁止更换架构**。

## 技术栈

| 层 | 选型 | 核心依赖 | 核心约束 |
|---|---|---|---|
| 后端 | Go 1.21+ / yang-controller-runtime / Gin | ygot、scrapligo（NETCONF）| Manager→Controller→Reconciler→Source，**禁止使用旧 Actor 模型** |
| 模型 | YANG + ygot | openconfig/ygot | 自动生成强类型结构体，禁止手写，遵循 YANG 树形层级（container/leaf/list/enum），禁止滥用 interface{} |
| 协议 | NETCONF (SSH 830) + gNMI (9339/9340) | RFC6241 + openconfig/gnmi | 支持 Get/Set/Subscribe，**禁止 Telnet/SNMP** |
| 缓存 | TTL+LRU 内存缓存 | 协程安全 | 无数据库，Key=设备IP+YANG路径，TTL 30s，配置下发后主动失效 |
| 前端 | Vue3 + Element Plus | Axios | YANG 模型自动渲染，禁止手写固定表单 |

## 架构约束

### yang-controller-runtime 分层
- **Manager**：全局生命周期管理，schema 加载、client 连接池、controller 注册、插件管理
- **Controller**：每 YANG 模块一个 Controller，处理事件队列，调用 Reconciler
- **Reconciler**：对齐 desired↔actual 配置（差异比对+配置推送），保持无状态
- **EventSource**：产生 reconcile 事件（周期轮询、gNMI 订阅、文件变更）
- **ClientPool**：设备连接池，断线自动重连、超时重试、异常错误处理
- 框架处理所有 boilerplate：schema 解析、连接管理、diff 计算、协议编码、限频重试、事件排队
- 用户只需实现 Reconciler 接口，不需要处理并发和连接管理

### 数据存储
- **禁止数据库**（MySQL/Redis/SQLite 等），仅允许 TTL+LRU 内存缓存 + 本地 JSON 元信息文件
- 运行配置：实时通过 NETCONF/gNMI 从交换机读取，**不持久化落盘**，缓存过期自动重新拉取
- 配置下发后主动失效对应缓存

### 前端
- 所有配置页面由 YANG 模型自动渲染，前端无状态，不存储配置，所有数据来自后端 Controller API
- YANG 类型自动映射：boolean→开关、enum→下拉、list→表格
- 配置编辑→提交→下发，全程联动后端 Controller + NETCONF
- 展示设备状态、缓存状态、下发结果、异常信息（设备离线、NETCONF 失败等）

## 开发流程

### OpenSpec 规范开发（主流程）

```
/opsx:explore → /opsx:propose → /opsx:apply → /opsx:sync → /opsx:archive
```

1. **explore**：探索需求、澄清问题、调查代码（禁止写代码）
2. **propose**：创建 change + 生成 proposal/design/tasks 全部制品
3. **apply**：按 tasks 逐项实现，标记完成 `- [x]`
4. **sync**：将 delta spec 合并到主 spec
5. **archive**：归档已完成 change

### TDD 约束
- **测试先行**：先写测试用例（正常/异常/并发），再写实现代码，**禁止先写代码后补测试**
- **代码量**：单次输出 ≤500行，超出必须拆分到下一个迭代
- **集成测试**：新增 YANG 模块必须添加 NETCONF 模拟网元集成测试（`*_integration_test.go`），`testing.Short()` 跳过
- **Code Review**：代码评审不通过，禁止提交
- **流程不可跳过**：OpenSpec→测试→代码→Review→Commit，禁止合并流程、省略步骤

### Worktree 隔离开发

```
EnterWorktree → 开发+测试+Commit → PR/Merge → ExitWorktree
```

- **强制 worktree**：新功能开发必须在 worktree 中进行，禁止在 main 上直接开发
- 每个 change/feature 对应一个 worktree
- hotfix 允许在 main 上操作但必须即时提交
- 合并前确保所有测试通过

### 提交规范
- 每个原子功能完成+测试通过后**立即提交**，禁止积累多个功能
- What/Why/How 三段式格式（`git-what-why-how-commit` 技能），**禁止简化、省略任何一段**：
  - **What**：明确本次变更的具体功能点/BUG 修复内容，不模糊、不冗余
  - **Why**：说明变更的业务背景、解决的痛点、架构必要性，禁止无理由提交
  - **How**：简要说明技术实现逻辑、改动范围、核心交互流程，贴合本次 ≤500行代码变更
- 单次 Commit 仅对应一个原子功能

## 技能映射

| 触发场景 | 技能 | 说明 |
|----------|------|------|
| 新 YANG 控制器开发 | `yang-controller-runtime-dev` | 架构合规 |
| YANG→Go 结构体 | `yang-ygot-generate` | 自动生成 |
| 配置缓存开发 | `go-ttl-lru-memory-cache` | TTL+LRU 并发安全 |
| NETCONF 对接 | `netconf-switch-protocol` | SSH 830 |
| 集成测试 | `netconf-sim-integration-test` | 模拟网元端到端验证 |
| TDD 开发 | `tdd-test-driven-dev` | 测试先行 |
| 代码评审 | `go-code-review-check` | 提交前强制 |
| 提交规范 | `git-what-why-how-commit` | 三段式 Commit |
| YANG 动态表单页面 | `frontend-yang-dynamic-form` | 功能型前端 |
| UI/UX 视觉设计 | `web-design-engineer` | 视觉型前端（页面美化/数据可视化/交互原型） |
| 规范开发全流程 | `/opsx:*` | OpenSpec explore→propose→apply→sync→archive |

### 前端技能触发规则
- **功能型**（动态表单、YANG 驱动页面）→ `frontend-yang-dynamic-form`
- **视觉型**（美化、可视化、交互原型）→ `web-design-engineer`
  - 必须先声明设计系统（色彩、排版、间距、阴影、动效）经确认后再写代码
  - 尽早出 v0 草案：核心结构 + 关键 token + 模块占位符
  - 提供至少 2 个设计变体供选择（保守+激进）
- **不触发设计技能**：纯逻辑开发（状态管理/API 对接）、工程化（构建配置/测试框架/性能优化）、纯功能（表单校验/路由/权限）

## 异常处理
- **设备离线**：NETCONF 自动重连，前端展示离线状态，API 返回明确错误
- **缓存过期**：自动重新拉取配置，更新缓存
- **Controller 故障**：不影响其他模块，Manager 自动重启故障 Controller
- **NETCONF 异常**：配置下发失败时前端提示错误，缓存不更新，保留原配置
- **前端异常**：表单校验失败不提交，展示 YANG 模型约束提示
- 所有异常必须有降级处理，禁止程序崩溃

## 通用约束
- 禁止引入无关第三方依赖
- 所有代码必须协程安全、无数据竞态、无内存泄漏，做好 panic 防护
- 遵循 Go 代码规范，命名统一、注释清晰，禁止冗余代码和过度封装
- 严禁 AI 风格陈词滥调（紫粉蓝渐变、左边框圆角卡片、滥用 Inter/Roboto）
- 无真实图标/图片时使用规范占位符，禁止用 emoji 替代图标

## 交付标准
1. 后端：可运行的 Go 项目（Controller 系统、API 接口、NETCONF 对接）
2. 前端：可运行的 Vue3 项目（动态表单、树形菜单、配置下发）
3. 测试：所有模块单元测试 + 异常测试 + 并发测试 + NETCONF 模拟网元集成测试
4. 合规：符合本文档全部约束，无数据库、无违规代码
