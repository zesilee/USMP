# CLAUDE.md 全面优化设计

## 背景

当前 CLAUDE.md (120行) + rules.md (60行) 存在三大问题：
1. **冗余**：架构约束、开发流程、提交规范在两文件中重复3次
2. **缺失**：无 OpenSpec 规范开发流程、无 Worktree 隔离开发流程、无 Superpowers 工作流整合
3. **低效**：散文体描述模糊，AI 需读2文件才能获得完整规则，关键约束散落各处

## 决策

| 决策项 | 选择 | 理由 |
|--------|------|------|
| rules.md 处理 | 合并到 CLAUDE.md 后删除 | 单一权威文档，无冲突风险 |
| 开发流程 | OpenSpec 为主流程 | 与项目已安装的 opsx 技能对齐 |
| Worktree | 强制隔离开发 | 保护 main 分支，规范化分支管理 |
| 文档风格 | 结构化精简体 | 表格+列表替代散文，AI 解析效率高 |

## 新 CLAUDE.md 结构

```
# USMP - Claude Code 开发规范

## 项目定位
## 技术栈
## 架构约束
## 开发流程
  ### OpenSpec 规范开发
  ### TDD 约束
  ### Worktree 隔离开发
  ### 提交规范
## 技能映射
## 异常处理
## 当前迭代优先级
```

预估行数：~90-100行（当前 180行，减少 ~45%）

## 各章节设计

### 1. 项目定位（1行）

单句定义：无数据库、模型驱动的交换机设备管理平台。

### 2. 技术栈（表格，~5行）

| 层 | 选型 | 核心约束 |
|---|---|---|
| 后端 | Go 1.21+ / yang-controller-runtime | Manager→Controller→Reconciler→Source |
| 模型 | YANG + ygot | 自动生成强类型，禁止手写 |
| 协议 | NETCONF (RFC6241) + gNMI | 禁止 Telnet/SNMP |
| 缓存 | TTL+LRU 内存 | 无数据库，Key=设备IP+YANG路径 |
| 前端 | Vue3 + Element Plus | YANG 模型自动渲染，禁止手写表单 |

### 3. 架构约束（~15行）

用简短列表替代当前散文描述：
- yang-controller-runtime 分层职责（Manager/Controller/Reconciler/EventSource/ClientPool 各1行）
- 数据存储：禁止数据库，仅 TTL+LRU 缓存 + JSON 元信息文件
- 运行配置：实时 NETCONF 读取，缓存 TTL 30s，下发后主动失效
- 前端：YANG 模型驱动渲染，前端无状态

### 4. 开发流程（核心章节，~35行）

#### 4.1 OpenSpec 规范开发流程

```
/opsx:explore → /opsx:propose → /opsx:apply → /opsx:sync → /opsx:archive
```

- explore：探索需求、澄清问题（禁止写代码）
- propose：创建 change + 生成 proposal/design/tasks
- apply：按 tasks 逐项实现，标记完成
- sync：将 delta spec 合并到主 spec
- archive：归档已完成 change

#### 4.2 TDD 约束

- 测试先行：先写测试用例（正常/异常/并发），再写实现
- 代码量：单次输出 ≤500行，超出必须拆分迭代
- 集成测试：新增 YANG 模块必须添加 NETCONF 模拟网元集成测试

#### 4.3 Worktree 隔离开发

强制流程：
1. EnterWorktree 创建隔离工作区
2. 在 worktree 中完成开发+测试
3. PR/Merge 回主干
4. ExitWorktree 清理

规则：
- 禁止在 main 上直接开发新功能
- 每个 change/feature 对应一个 worktree
- hotfix 允许在 main 上操作但必须即时提交

#### 4.4 提交规范

- 每个原子功能完成+测试通过后立即提交
- What/Why/How 三段式格式
- 单次 Commit 仅对应一个原子功能

### 5. 技能映射（精简表格，~15行）

| 触发场景 | 技能 | 说明 |
|----------|------|------|
| 新 YANG 控制器开发 | yang-controller-runtime-dev | 架构合规 |
| YANG→Go 结构体 | yang-ygot-generate | 自动生成 |
| 配置缓存 | go-ttl-lru-memory-cache | TTL+LRU 并发安全 |
| NETCONF 对接 | netconf-switch-protocol | SSH 830 |
| 集成测试 | netconf-sim-integration-test | 模拟网元 |
| TDD 开发 | tdd-test-driven-dev | 测试先行 |
| 代码评审 | go-code-review-check | 提交前强制 |
| 提交规范 | git-what-why-how-commit | 三段式 |
| YANG 动态表单 | frontend-yang-dynamic-form | 功能型前端 |
| UI/UX 设计 | web-design-engineer | 视觉型前端 |
| 规范开发 | /opsx:* | OpenSpec 全流程 |

### 6. 异常处理（~8行）

每条1行：设备离线/缓存过期/Controller故障/NETCONF异常/前端异常

### 7. 当前迭代优先级（~8行）

保留现有优先级列表，标记已完成项

## 实施步骤

1. 编写新 CLAUDE.md（结构化精简体，~90-100行）
2. 删除 rules.md
3. 验证新 CLAUDE.md 无信息遗漏
4. 提交

## 风险

- 信息遗漏：合并两文件时可能遗漏约束 → 逐条对照 checklist
- AI 适配：新格式是否影响 AI 解析 → 结构化格式对 AI 更友好，风险低
