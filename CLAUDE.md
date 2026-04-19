# 交换机设备管理平台 - Claude Code 开发规范
本文件用于指导 Claude Code 按项目架构、全局规则、技能集，全程遵循 Plan 模式+TDD+小步迭代，所有开发行为需严格匹配已加载的 skills 和 rules。

## 一、项目概述
### 项目目标
开发一套 **无数据库、高并发、模型驱动** 的交换机设备管理平台，支持设备管控、配置读写、动态表单展示，基于 Actor 模型实现设备隔离、故障隔离，通过 YANG 模型实现前后端配置联动。

### 核心架构（不可变更）
- 后端：Go 1.21+，protoactor-go 实现双层 Actor 架构（DeviceActor + YANG Object Actor）
- 配置模型：YANG 模型 + ygot 自动生成强类型结构体
- 协议：NETCONF（RFC6241），SSH 830端口
- 缓存：TTL+LRU 内存缓存（无数据库）
- 前端：基于 YANG 模型自动生成动态表单，无本地存储

### 技术栈（固定）
| 模块 | 技术选型 | 核心依赖 |
|------|----------|----------|
| 后端框架 | Go + Gin | protoactor-go、ygot、scrapligo（NETCONF） |
| Actor 架构 | protoactor-go | 双层 Actor（设备+YANG对象）、异步消息 |
| 配置模型 | YANG + ygot | openconfig/ygot（自动生成结构体） |
| 协议通信 | NETCONF | RFC6241 标准、SSH 830端口 |
| 缓存 | 内存缓存 | TTL+LRU、协程安全 |
| 前端 | Vue3 + Element Plus | 动态表单、树形菜单、Axios |

## 二、开发流程规范（严格遵循）
全程执行 Plan 模式 + TDD 测试驱动，步骤不可跳过、不可合并：
1.  需求拆分：每个迭代仅做1个原子功能，输出 Iteration Plan
2.  测试先行：先编写单元测试用例（覆盖正常/异常/并发场景）
3.  代码实现：单次输出代码 ≤ 500 行，贴合技能规范
4.  代码评审：自动执行 Code Review，不通过则整改
5.  提交规范：Review 通过后，生成 What/Why/How 标准 Commit
6.  迭代循环：完成一个原子功能，进入下一个迭代

## 三、核心架构约束（与 rules.md 一致）
### 1. Actor 架构约束（核心）
- 一台交换机 = 1 个 DeviceActor（顶层Actor）
- 设备内每个 YANG 配置对象 = 1 个独立 YANG Object Actor（子Actor）
- Actor 树形结构：ManagerActor → DeviceActor → YANG Object Actor
- 每个 YANG Actor 独立持有：配置（ygot结构体）、TTL缓存、NETCONF同步状态
- DeviceActor 销毁时，自动销毁其下所有 YANG Object Actor
- 所有 Actor 仅通过异步消息通信，无直接调用，保证并发安全

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

## 四、技能集关联（8个技能，自动联动）
| 技能名称 | 核心作用 | 联动模块 |
|----------|----------|----------|
| go-protoactor-device | 实现双层 Actor 架构（设备+YANG对象） | ManagerActor、DeviceActor、YANG Actor 生命周期管理 |
| yang-ygot-generate | YANG 模型→Go 强类型结构体 | 所有配置读写、NETCONF 序列化/反序列化 |
| go-ttl-lru-memory-cache | 高性能内存缓存（TTL+LRU） | DeviceActor、YANG Actor、配置读取优化 |
| netconf-switch-protocol | NETCONF 协议对接交换机 | DeviceActor、YANG Actor、配置读写 |
| tdd-test-driven-dev | TDD 测试驱动，先测试后代码 | 所有模块（Actor、缓存、NETCONF、前端） |
| go-code-review-check | 自动代码评审，确保合规性 | 所有代码提交前强制评审 |
| git-what-why-how-commit | 标准三段式 Commit 规范 | 所有代码提交，小步迭代 |
| frontend-yang-dynamic-form | 基于 YANG 自动生成前端表单 | 后端 YANG Actor、API 接口，实现前后端联动 |

## 五、开发优先级（迭代顺序）
1.  项目初始化：Actor 系统、ManagerActor、全局缓存、基础API
2.  DeviceActor 开发：设备启停、YANG 子 Actor 自动创建
3.  YANG 模型 + ygot 生成：结构体生成、序列化/反序列化
4.  NETCONF 客户端：连接、get-config/edit-config/commit 操作
5.  缓存联动：配置读取缓存、下发失效缓存
6.  前端动态表单：YANG 模型映射、配置展示/编辑/下发
7.  异常处理：设备离线、NETCONF 重连、缓存过期、Actor 故障

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
3.  测试用例：所有模块的单元测试、异常测试、并发测试
4.  文档：代码注释、API 文档、部署说明
5.  合规性：符合所有 rules 和 skills 规范，无数据库、无违规代码