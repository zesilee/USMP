# USMP - Universal Switch Management Platform

> 无数据库、高并发、模型驱动的交换机设备管理平台

## 📖 项目简介

USMP 是一个基于 **yang-controller-runtime**（Kubernetes controller-runtime 风格架构）+ **YANG 模型驱动** 的网络交换机配置管理平台。采用声明式配置管理，框架处理所有 boilerplate，开发者仅需编写 Reconciler 业务逻辑。

### 核心设计理念

| 设计原则 | 说明 |
|---------|------|
| **无数据库** | 仅 JSON 存储设备元信息，运行配置全靠 NETCONF 实时拉取 + TTL+LRU 内存缓存 |
| **Controller 架构** | Manager → Controller → Reconciler → Source，每 YANG 模块一个 Controller |
| **模型驱动全流程** | YANG → ygot 自动生成 Go 结构体 → 后端暴露模型结构 → 前端自动生成动态表单 |
| **缓存一致性** | 配置下发成功后主动失效对应缓存，下次读取自动拉取最新配置 |
| **NETCONF 标准** | 遵循 RFC6241，支持 get-config/edit-config/commit，断线自动重连 |

## 🏗️ 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                 Vue3 前端 · 自动生成动态表单                    │
└─────────────────────────────────────────────────────────────┘
                              ↓ HTTP REST API
┌─────────────────────────────────────────────────────────────┐
│                     Gin · 后端 API 层                         │
└─────────────────────────────────────────────────────────────┘
                              ↓ 调用
┌─────────────────────────────────────────────────────────────┐
│           Manager · 全局生命周期管理                           │
│  • Controller 注册、Schema 加载、Client 连接池管理              │
└─────────────────────────────────────────────────────────────┘
           ┌───────────────┬───────────────┬───────────────┐
           ↓               ↓               ↓               ↓
┌──────────────────┐┌──────────────────┐┌──────────────────┐
│  VLAN Controller ││  IF Controller   ││  Sys Controller  │  ← 每YANG模块一个
│  Reconciler      ││  Reconciler      ││  Reconciler      │
│  Event Queue     ││  Event Queue     ││  Event Queue     │
└──────────────────┘└──────────────────┘└──────────────────┘
           │
           ├───────────────┬───────────────┬───────────────┐
           ↓               ↓               ↓               ↓
┌────────────────────┐┌────────────────────┐┌────────────────────┐
│  Periodic Source   ││  File Watcher      ││  gNMI Subscribe     │  ← 事件源
│  周期性轮询        ││  文件变更监听      ││  订阅推送           │
└────────────────────┘└────────────────────┘└────────────────────┘
           │
           ↓ Reconcile 请求
┌─────────────────────────────────────────────────────────────┐
│              TTL+LRU · 内存缓存 (无数据库)                      │
│              Key = 设备IP + YANG路径 · TTL自动过期             │
└─────────────────────────────────────────────────────────────┘
           │
           ↓ 配置读写
┌─────────────────────────────────────────────────────────────┐
│              NETCONF · RFC6241 标准协议                        │
│          get-config / edit-config / commit / auto-reconnect    │
└─────────────────────────────────────────────────────────────┘
           │
           ↓ TCP/SSH 连接
┌─────────────────────────────────────────────────────────────┐
│                    物理交换机设备                              │
└─────────────────────────────────────────────────────────────┘
```

## 🛠️ 技术栈

| 层级 | 技术 | 用途 |
|------|------|------|
| 后端语言 | Go 1.21+ | 静态类型，高并发 |
| Controller 架构 | yang-controller-runtime | 声明式配置管理，类似 K8s controller-runtime |
| Web 框架 | [Gin](https://github.com/gin-gonic/gin) | REST API |
| YANG 代码生成 | [ygot](https://github.com/openconfig/ygot) | YANG → Go 强类型结构体 |
| NETCONF 客户端 | [scrapligo](https://github.com/scrapli/scrapligo) | NETCONF 协议对接 |
| 缓存 | 自研 TTL+LRU | 内存缓存，无外部依赖 |
| 前端 | Vue3 + TypeScript + Element Plus | 动态表单自动生成 |

## 🚀 快速开始

### 编译后端

```bash
git clone https://github.com/leezesi/usmp.git
cd usmp/backend
go mod tidy
go build -o usmp .
```

### 运行后端

```bash
./usmp
```

默认监听 `0.0.0.0:8080`，API 基础路径 `/api/v1`。

### 启动前端（开发模式）

```bash
cd frontend
npm install
npm run dev
```

## 📁 项目结构

```
usmp/
├── backend/                       # Go 后端代码
│   ├── main.go                    # 程序入口
│   ├── go.mod/go.sum              # Go 模块定义
│   ├── cmd/
│   │   └── test-server/           # E2E 测试服务器
│   ├── internal/
│   │   ├── api/                   # REST API 层
│   │   ├── cache/                 # TTL+LRU 内存缓存
│   │   ├── controller/vlan/       # VLAN Controller + Reconciler
│   │   ├── types/                 # 公共类型定义
│   │   ├── yang/                  # YANG 代码生成工具
│   │   └── generated/openconfig/  # ygot 自动生成的 Go 结构体
│   ├── pkg/
│   │   └── yang-runtime/          # 核心框架（Manager/Controller/Source/Client）
│   └── simulator/                 # NETCONF 模拟网元
│       ├── netconfsim/            # 完整 SSH 模拟器
│       └── netsim/                # 简单 REST 模拟器
├── frontend/                      # Vue3 前端项目
│   ├── package.json
│   ├── vite.config.ts
│   ├── playwright.config.ts
│   ├── index.html
│   ├── src/                       # 源代码
│   └── tests/                     # E2E 测试
├── scripts/                       # 公共脚本
│   └── e2e-test.sh               # E2E 测试启动脚本
├── spec/                          # YANG 规范文档
└── yang-models/                   # YANG 模型文件
```

## 🌐 API 端点

### 设备管理

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/api/v1/devices` | 列出所有设备 |
| GET | `/api/v1/devices/:ip/status` | 获取设备状态 |

### 配置读写

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/api/v1/config/:ip/:path` | 获取配置（自动缓存）|
| POST | `/api/v1/config/:ip/:path` | 下发配置 |

### YANG 模型

| 方法 | 路径 | 功能 |
|------|------|------|
| GET | `/api/v1/yang/modules` | 列出所有支持的 YANG 模块 |

## ✅ 测试

### 后端单元测试

```bash
cd backend
go test ./... -v
```

### 后端集成测试（包含 NETCONF 模拟器）

```bash
cd backend
go test ./... -v                    # 包含集成测试
go test ./... -v -short            # 跳过集成测试，只跑单元测试
```

### 前端单元测试

```bash
cd frontend
npm run test
```

### E2E 端到端测试

```bash
# 方式 1：使用自动化脚本（推荐）
./scripts/e2e-test.sh

# 方式 2：手动启动后端 + 前端测试
cd backend && go run ./cmd/test-server/main.go &
cd frontend && npm run e2e
```

## 🎯 核心特性

### 1. yang-controller-runtime 架构

- **声明式配置**：只需要定义 desired state，框架自动对齐 actual state
- **自动重试**：配置失败自动指数退避重试
- **事件驱动**：支持 Periodic/File/gNMI 多种事件源
- **并发安全**：Controller 队列机制，避免竞态

### 2. 故障隔离

- 一台设备故障不影响其他设备
- Client 连接池自动管理，断线自动重连
- 配置下发失败不影响缓存，保留原配置

### 3. 无数据库

- 无需安装运行 MySQL/Redis 等任何数据库
- 设备元信息存在本地 JSON 文件
- 运行配置全在内存，重启自动从设备重新拉取

### 4. 动态表单

- 前端完全根据 YANG 模型自动生成表单控件
  - `boolean` → Switch 开关
  - `enumeration` → Select 下拉框
  - `int`/`uint` → Input 数字框
  - `string` → Input 文本框
  - `list` → Table 表格
- 无需手写表单，新增 YANG 模块自动生效

## 📝 开发流程

本项目严格遵循 TDD 测试驱动开发 + 小步迭代：

1. `Plan` → 需求拆分，每个迭代一个原子功能
2. `Test` → 先写单元测试（正常/异常/并发场景）
3. `Code` → 实现功能（单次 ≤ 500 行）
4. `Review` → 自动代码评审
5. `Integration Test` → 添加基于 NETCONF 模拟网元的集成测试
6. `Commit` → 标准 What/Why/How 三段式提交

## 📄 License

MIT License

## 🙏 致谢

- [openconfig](https://github.com/openconfig) - YANG 模型标准
- [ygot](https://github.com/openconfig/ygot) - YANG Go 代码生成
- [scrapligo](https://github.com/scrapli/scrapligo) - NETCONF 客户端
