# USMP - Universal Switch Management Platform

> 无数据库、高并发、模型驱动的交换机设备管理平台

## 📖 项目简介

USMP 是一个基于 **yang-controller-runtime（自研）** + **YANG 模型驱动** 的网络交换机配置管理平台。声明式配置管理：框架自动对齐 desired ↔ actual（diff + NETCONF 下发），开发者仅需实现 Reconciler 业务逻辑；前端由 YANG Schema 自动渲染动态表单，新增 YANG 模块零前端代码。

### 核心设计理念

| 设计原则 | 说明 |
|---------|------|
| **无数据库** | 运行配置实时从设备读取，仅 TTL+LRU 内存缓存；持久元信息经 K8s CRD（仅当载体，R03） |
| **声明式对账** | Manager → Controller → Reconciler → EventSource，desired ↔ actual 自动收敛（R01） |
| **模型驱动全流程** | YANG → 自研 yanggen 生成 Go 结构体 → 后端 API → 前端自动渲染动态表单（R04/R05） |
| **NETCONF 标准** | 自研 netconfcore 引擎，SSH 830，遵循 RFC6241（R02） |
| **多实例 PaaS 就绪** | K8s 内多实例部署，禁本地持久文件，选主统一走 leader.Gate |

## 🏗️ 整体架构

```
┌──────────────────────────────────────────────────────┐
│        React 19 前端 · YANG 模型驱动动态渲染           │
│   /yang/schema → 动态表单/表格/左树（R05 零手写表单）   │
└──────────────────────────────────────────────────────┘
                    ↓ HTTP REST API
┌──────────────────────────────────────────────────────┐
│              Beego · 后端 API 层 (:8080)              │
└──────────────────────────────────────────────────────┘
                    ↓
┌──────────────────────────────────────────────────────┐
│         yang-controller-runtime（自研框架）            │
│  Manager → Controller(每 YANG 模块) → Reconciler      │
│  EventSource(轮询/CRD watch) · ClientPool(断线重连)    │
│  TTL+LRU 配置缓存 · 持久元信息经 K8s CRD               │
└──────────────────────────────────────────────────────┘
                    ↓ NETCONF (SSH 830, 自研 netconfcore)
┌──────────────────────────────────────────────────────┐
│         物理交换机 / netconfsim 模拟网元               │
└──────────────────────────────────────────────────────┘
```

## 🛠️ 技术栈

| 层级 | 技术 | 用途 |
|------|------|------|
| 后端语言 | Go 1.22（交付钉死） | 静态类型，高并发 |
| 控制器框架 | yang-controller-runtime（自研） | 声明式对账，用户只写 Reconciler |
| Web 框架 | Beego v2 | REST API 路由层 |
| NETCONF 客户端 | netconfcore（自研） | RFC6241，SSH 830，断线重连 |
| YANG 工具链 | 自研 yanggen（构建期用 goyang） | YANG → Go 结构体自动生成（R04） |
| 缓存 | TTL+LRU 内存缓存 | Key=设备IP+YANG路径，TTL 30s，下发后失效 |
| 前端 | React 19 + TypeScript + EviewUI（经 `src/ui` 适配层） | YANG Schema 自动渲染动态表单（R05） |
| 前端状态/路由 | zustand + react-router | 外网测试走 antd 测试镜像后端 |
| 测试 | go test + vitest + Playwright + netconfsim | 分层军规见 CLAUDE.md §5.6 |

## 🚀 快速开始

### ⚡ 一键激活开发环境（克隆后必执行）

```bash
git clone https://github.com/zesilee/USMP.git
cd USMP
make setup
```

`make setup` 自动完成：Git Hooks 激活 + Go/前端依赖安装 + 基线测试 + 拦截体系验证。
**未执行 `make setup` 会导致提交拦截和 CI 检查不生效。**

详见 [TEAM_HANDBOOK.md](TEAM_HANDBOOK.md) 和 [docs/compliance/SETUP_GUIDE.md](docs/compliance/SETUP_GUIDE.md)。

### 本地全栈热循环（免 docker）

```bash
make dev     # go build 后端(:8080) + vite dev 前端(:3000, HMR)，Ctrl-C 同停
```

### Docker 全栈（含模拟网元）

```bash
docker compose up -d
# 前端 http://localhost:3002 · 后端 API http://localhost:8080/api/v1 · 模拟器 :830
```

### Kind 集群一键部署

```bash
scripts/kind-deploy.sh
```

## 📁 项目结构

```
USMP/
├── backend/                  # Go 后端（backend/main.go 唯一生产入口）
│   ├── internal/             # API 层、controller、cache、生成代码
│   ├── pkg/yang-runtime/     # 自研框架：manager/controller/reconcile/client/...
│   │   └── client/netconfcore/   # 自研 NETCONF 引擎
│   ├── simulator/netconfsim/ # NETCONF 模拟网元（集成测试用）
│   └── tools/                # 构建期工具（yanggen/rpcgen/crdgen 等，独立 module）
├── frontend/                 # React 19 前端
│   ├── src/ui/               # UI 适配层（EviewUI 桥 / antd 测试镜像，@ui-backend 单点切换）
│   ├── test/                 # F1/F2/F3 分层测试（见 frontend/TESTING.md）
│   └── tests/                # F4 Playwright 部署冒烟
├── snd/                      # 华为 CE 系列 YANG 模型源与 i18n 资源（生成基线）
├── openspec/                 # 规格与变更管理（spec-first，R17）
├── deploy/                   # K8s 清单与 CRD
├── docs/                     # CICD、部署、调研、项目记忆（docs/memory/）
└── scripts/                  # 构建/部署/门禁脚本
```

## ✅ 测试

```bash
cd backend && go test ./...        # 后端全量（含 -race 与模拟网元集成测试）
cd frontend && npm test            # 前端单测（F1/F2，happy-dom）
cd frontend && npm run test:browser  # F3 真浏览器组件测试
make e2e-local                     # 起 docker 全栈 → Playwright 冒烟（F4）
```

测试分层军规（改动类型 → 必补层）见 [CLAUDE.md §5.6](CLAUDE.md)；前端权威规范见 [frontend/TESTING.md](frontend/TESTING.md)。

## 📚 更多文档

- [CLAUDE.md](CLAUDE.md) — 开发规范与架构红线（R01-R18）
- [TEAM_HANDBOOK.md](TEAM_HANDBOOK.md) — 开发流程、自审清单、安全合入
- [docs/CICD.md](docs/CICD.md) — CI/CD 与自托管 Runner
- [docs/DEPLOY-WSL-CN.md](docs/DEPLOY-WSL-CN.md) — 内网 WSL 部署
- [openspec/SPEC_CONVENTIONS.md](openspec/SPEC_CONVENTIONS.md) — 规格书写约定
