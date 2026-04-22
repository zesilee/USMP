# 交换机设备管理平台 - Claude Code 开发规范
本文件用于指导 Claude Code 按项目架构、全局规则、技能集，全程遵循 Plan 模式+TDD+小步迭代，所有开发行为需严格匹配已加载的 skills 和 rules。

## 一、项目概述
### 项目目标
开发一套 **无数据库、高并发、模型驱动** 的交换机设备管理平台，支持设备管控、配置读写、动态表单展示，基于 **yang-controller-runtime**（controller-runtime 风格架构）实现声明式配置管理。框架处理所有 boilerplate，开发者仅需编写 Reconciler 业务逻辑。

### 核心架构（不可变更）
- 后端：Go 1.21+，**yang-controller-runtime** 框架（Kubernetes controller-runtime 架构风格）
- 配置模型：YANG 模型 + ygot 自动生成强类型结构体
- 协议：NETCONF（RFC6241）+ gNMI 双协议支持
- 缓存：TTL+LRU 内存缓存（无数据库）
- 前端：基于 YANG 模型自动生成动态表单，无本地存储

### 技术栈（固定）
| 模块 | 技术选型 | 核心依赖 |
|------|----------|----------|
| 后端框架 | Go + Gin | yang-controller-runtime、ygot、scrapligo（NETCONF）|
| controller-runtime 架构 | Manager → Controller → Reconciler → Source | 全局生命周期管理、每 YANG 模块一个 Controller |
| 配置模型 | YANG + ygot | openconfig/ygot（自动生成结构体） |
| 协议通信 | NETCONF + gNMI | RFC6241 标准 + openconfig/gnmi |
| 缓存 | 内存缓存 | TTL+LRU、协程安全 |
| 前端 | Vue3 + Element Plus | 动态表单、树形菜单、Axios |

## 二、开发流程规范（**严格遵循，违者重罚**）
全程执行 **Plan 模式 + TDD 测试驱动 + 小步迭代**，步骤不可跳过、不可合并：
1.  **需求拆分**：每个迭代仅做 **1 个原子功能**，**必须**拆分到可在单次迭代中完成，输出 Iteration Plan
2.  **测试先行**：先编写单元测试用例（覆盖正常/异常/并发场景），测试不写不写实现代码
3.  **代码实现**：**单次输出代码 ≤ 500 行**，这是硬性约束，超过必须拆分到下一个迭代
    - 即使是一个大功能，也要拆分多次迭代，每次不超过 500 行
    - 不许一次性输出上千行代码，必须小步前进
4.  **代码评审**：自动执行 Code Review，不通过则整改
5.  **集成测试**：**所有新增 YANG 模块业务功能，必须添加基于 NETCONF 模拟网元的集成测试**
    - 集成测试放在对应业务包 `*_integration_test.go`
    - 必须覆盖正常流程和至少一个异常场景
    - 所有集成测试必须执行成功才能提交代码
    - 使用 `if testing.Short() { t.Skip() }` 跳过集成测试让日常单元测试更快
6.  **提交代码**：**每次迭代完成一个完整原子功能且所有测试通过后，必须立即提交代码**，不许积累多个功能再提交
    - 使用 `git-what-why-how-commit` 规范生成标准三段式 Commit
    - 一个原子功能一个 Commit，方便追溯和回滚
7.  **迭代循环**：完成一个原子功能，进入下一个迭代

## 三、核心架构约束（与 rules.md 一致）
### 1. yang-controller-runtime 架构约束（核心）
- **Manager**：全局生命周期管理，负责 schema 加载、client 连接池、controller 注册、插件管理
- **Controller**：每个 YANG 模块一个 Controller，处理事件队列，调用 Reconciler
- **Reconciler**：用户实现，对齐 desired ↔ actual 配置（差异比对 + 配置推送）
- **EventSource**：产生 reconcile 事件（周期轮询、gNMI 订阅、文件变更）
- **ClientPool**：设备连接池，自动重连，复用连接
- 框架处理所有 boilerplate：schema 解析、连接管理、diff 计算、协议编码、限频重试、事件排队
- 用户只需要实现 Reconciler 接口，不需要处理并发和连接管理

### 2. 配置管理约束
- 无任何数据库（禁止使用 MySQL、Redis、SQLite 等）
- 设备元信息：仅存储在本地 JSON 文件（不存储运行配置）
- 运行配置：实时通过 NETCONF 从交换机读取，存入 TTL 内存缓存
- 缓存规则：Key = 设备IP + YANG节点路径，TTL默认30秒，配置下发后主动失效
- YANG 模型：所有配置结构体由 ygot 自动生成，不手写

### 3. 前端约束
- 所有界面由 YANG 模型自动生成，不手写固定表单
- 支持 YANG 类型自动映射（boolean→开关、enum→下拉、list→表格等）
- 前端无状态，不存储任何配置，所有数据来自后端 Actor API
- 配置编辑→提交→下发，全程联动后端 YANG Actor + NETCONF
- 展示设备状态、缓存状态、下发结果、异常信息（设备离线、NETCONF失败等）

## 四、技能集关联（10个技能，自动联动）
| 技能名称 | 核心作用 | 联动模块 |
|----------|----------|----------|
| **yang-controller-runtime-dev** | 基于 yang-controller-runtime 开发 YANG 模块控制器 | 所有新 YANG 控制器开发，遵循架构规范 |
| yang-ygot-generate | YANG 模型→Go 强类型结构体 | 所有配置读写、NETCONF/gNMI 序列化/反序列化 |
| go-ttl-lru-memory-cache | 高性能内存缓存（TTL+LRU） | 配置缓存、desired state 存储 |
| netconf-switch-protocol | NETCONF 协议对接交换机 | 设备连接、配置读写 |
| **netconf-sim-integration-test** | NETCONF 模拟网元生成集成测试 | **所有新增业务必须添加**，端到端验证 |
| tdd-test-driven-dev | TDD 测试驱动，先测试后代码 | 所有模块（框架、controller、client） |
| go-code-review-check | 自动代码评审，确保合规性 | 所有代码提交前强制评审 |
| git-what-why-how-commit | 标准三段式 Commit 规范 | **每次迭代完成必须提交**，小步迭代 |
| frontend-yang-dynamic-form | 基于 YANG 自动生成前端表单 | 后端 API 接口，实现前后端联动 |

## 五、开发优先级（迭代顺序）
1.  **框架实现**：yang-controller-runtime 核心架构（已完成）
2.  **API 迁移**：REST API 迁移到新框架（已完成）
3.  **移除旧架构**：删除 legacy Actor 实现（已完成）
4.  **新增 YANG 模块控制器**：每个 YANG 模块一个 Controller + Reconciler
    - OpenConfig Interfaces 接口配置
    - OpenConfig VLANs VLAN 配置
    - OpenConfig System 系统信息
5.  **端到端测试**：真实设备连接测试、配置读写验证
6.  **异常处理**：设备离线、重连、缓存过期、故障恢复

## 六、Claude Code 操作说明
1.  自动加载：启动后自动加载 `.claude/rules.md` 和 `.claude/skills/*` 所有技能
2.  技能触发：根据开发场景自动匹配对应技能（无需手动指定）
3.  流程约束：严格执行 Plan→测试→代码→Review→Commit，不跳过任何步骤
4.  代码约束：单次代码 ≤ 500 行，贴合所有 rules 和 skills 规范
5.  联动要求：前后端、Actor、NETCONF、缓存需自动联动，确保数据一致性

## 七、异常场景处理规范
1.  设备离线：NETCONF 自动重连，前端展示离线状态，API 返回明确错误
2.  缓存过期：自动通过 NETCONF 重新拉取配置，更新缓存
3.  Actor 故障：DeviceActor 故障不影响其他设备，ManagerActor 自动重启故障 Actor
4.  NETCONF 异常：配置下发失败时，前端提示错误，缓存不更新，保留原配置
5.  前端异常：表单校验失败不提交，展示 YANG 模型约束提示

## 八、交付标准
1.  后端：可运行的 Go 项目（Actor 系统、API 接口、NETCONF 对接）
2.  前端：可运行的 Vue3 项目（动态表单、树形菜单、配置下发）
3.  测试用例：
    - 所有模块的单元测试、异常测试、并发测试
    - **所有新增业务必须包含基于 NETCONF 模拟网元的集成测试**
    - 所有集成测试必须执行成功才能提交
4.  文档：代码注释、API 文档、部署说明
5.  合规性：符合所有 rules 和 skills 规范，无数据库、无违规代码