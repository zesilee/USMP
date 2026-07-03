# USMP - Universal Switch Management Platform

> 无数据库、高并发、模型驱动的交换机设备管理平台

## 📖 项目简介

USMP 是一个基于 **Kubernetes CRD + Operator 架构** + **YANG 模型驱动** 的网络交换机配置管理平台。采用声明式配置管理，Kubernetes Operator 自动对齐 desired ↔ actual state，开发者仅需编写 Reconciler 业务逻辑。

### 核心设计理念

| 设计原则 | 说明 |
|---------|------|
| **无数据库** | 所有配置均为 Kubernetes CRD，etcd 持久化存储 |
| **Operator 架构** | Manager → Controller → Reconciler，每业务类型一个 Controller |
| **模型驱动全流程** | YANG → CRD → 后端 API → 前端自动生成动态表单 |
| **声明式配置** | `kubectl apply` 即可下发配置到交换机 |
| **NETCONF 标准** | 遵循 RFC6241，支持 get-config/edit-config/commit |

## 🏗️ 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                 Vue3 前端 · CRD 动态表单                      │
│              自动解析 YANG Schema → 动态表单                    │
└─────────────────────────────────────────────────────────────┘
                              ↓ HTTP REST API
┌─────────────────────────────────────────────────────────────┐
│                     Gin · 后端 API 层                         │
└─────────────────────────────────────────────────────────────┘
                              ↓ 调用 Kubernetes API
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes API Server                    │
│                    CRD 存储、Watch、状态机                          │
└─────────────────────────────────────────────────────────────┘
           ┌───────────────┬───────────────┬───────────────┐
           ↓               ↓               ↓               ↓
┌──────────────────┐┌──────────────────┐┌──────────────────┐
│BusinessSwitch    ││ BusinessVlan      ││BusinessInterface │  ← 每CRD类型
│ Controller      ││ Controller      ││ Controller      │
│ Reconciler      ││ Reconciler      ││ Reconciler      │
└──────────────────┘└──────────────────┘└──────────────────┘
           │
           ├───────────────────────────────────────────────┐
           ↓ Reconcile 循环
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
| Operator 框架 | kubebuilder + controller-runtime | Kubernetes 原生 Operator |
| Web 框架 | [Gin](https://github.com/gin-gonic/gin) | REST API |
| CRD 定义 | kubebuilder | 自动生成 CRD YAML |
| NETCONF 客户端 | [scrapligo](https://github.com/scrapli/scrapligo) | NETCONF 协议对接 |
| 前端 | Vue3 + TypeScript + Element Plus | CRD Schema 解析 + 动态表单 |
| E2E 测试 | Kind + Ginkgo + envtest | 端到端集成测试 |

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

### 编译后端

```bash
git clone https://github.com/leezesi/usmp.git
cd usmp/backend
go mod tidy
go build -o usmp .
```

### 安装 CRD 到 Kubernetes 集群

```bash
cd backend
make install
```

### 运行 Controller

```bash
cd backend
make run
```

### 启动前端（开发模式）

```bash
cd frontend
npm install
npm run dev
```

### Kind 集群一键部署（推荐）⭐

推荐使用 `make deploy` 一键部署，包含完整的服务状态校验：

```bash
# ✨ 一键部署（推荐）
cd backend
make deploy        # 自动完成：环境检查 -> 创建集群 -> 加载镜像 -> 部署 -> 服务校验

# 📊 独立校验部署状态
make kind-verify   # 20+ 项检查：集群/Namespace/CRD/Controller/前端/模拟器/RBAC

# 📋 其他常用命令
make kind-status   # 查看集群状态
make kind-logs     # 查看 Controller 日志
make kind-clean    # 清理集群

# 🎯 访问地址
# 前端界面: http://localhost:30081
# 后端 API: http://localhost:30080
# NETCONF 模拟器: localhost:30830
```

<details>
<summary>🔧 手动分步部署（高级用户）</summary>

```bash
# 1. 构建镜像
cd backend
make docker-build
make docker-build-frontend

# 2. 创建 Kind 集群
make kind-cluster

# 3. 加载镜像到集群
make kind-load-images

# 4. 部署组件
make kind-deploy

# 5. 校验服务状态
make kind-verify
```

</details>

## 📁 项目结构

```
usmp/
├── backend/                       # Go 后端代码
│   ├── main.go                    # 程序入口
│   ├── go.mod/go.sum              # Go 模块定义
│   ├── api/                       # CRD API 定义
│   │   ├── biz/v1/                # 业务 CRD (Switch/VLAN/Interface)
│   │   └── core/v1/               # 核心 CRD (NativeDeviceConfig)
│   ├── cmd/
│   │   └── test-server/           # E2E 测试服务器
│   ├── internal/
│   │   ├── api/                   # REST API 层
│   │   ├── controller/             # Controller + Reconciler
│   │   └── types/                # 公共类型定义
│   ├── pkg/
│   │   └── netconf/                # NETCONF 客户端
│   ├── config/                    # Kustomize 配置
│   │   ├── crd/                   # CRD 定义
│   │   ├── rbac/                  # RBAC 配置
│   │   ├── manager/               # Controller Manager
│   │   └── samples/               # CRD 示例
│   └── test/
│       ├── e2e/                    # E2E 端到端测试
│       │   ├── e2e_suite_test.go  # 测试套件
│       │   ├── *test.go           # 各 CRD 测试
│       │   ├── config/              # Kind 集群配置
│       │   └── run_e2e.sh         # 自动化测试脚本
│       └── integration/           # 集成测试
├── frontend/                      # Vue3 前端项目
│   ├── package.json
│   ├── vite.config.ts
│   ├── src/                       # 源代码
│   │   ├── composables/            # CRD Schema 解析 + 动态表单
│   │   └── components/             # UI 组件
│   └── tests/                     # E2E 测试
├── docs/
│   └── superpowers/
│       └── plans/                # 实现计划文档
└── scripts/                       # 公共脚本
```

## 🌐 CRD API 资源

### 业务 CRD

| CRD | API | 功能 |
|-----|-----|------|
| BusinessSwitch | `biz.usmp.io/v1` | 交换机设备管理 |
| BusinessVlan | `biz.usmp.io/v1` | VLAN 配置管理 |
| BusinessInterface | `biz.usmp.io/v1` | 接口配置管理 |
| BusinessRoute | `biz.usmp.io/v1` | 路由配置管理 |

### 核心 CRD

| CRD | API | 功能 |
|-----|-----|------|
| NativeDeviceConfig | `core.usmp.io/v1` | 原生 CLI 配置下发 |

## ✅ 测试

### 后端单元测试

```bash
cd backend
go test ./... -v
```

### 后端集成测试

```bash
cd backend
go test ./... -v                    # 包含集成测试
go test ./... -v -short            # 跳过集成测试，只跑单元测试
```

## 🧪 E2E 端到端测试（Kubernetes Native）

本项目提供 **两级 E2E 测试策略**，覆盖从快速验证到完整集群测试。

### 测试策略对比

| 特性 | Go E2E (envtest) | Kind E2E (完整集群) |
|------|------------------|---------------------|
| **速度** | ⚡ 极快 (~30秒 | 🐢 较慢 (~3-5分钟) |
| **环境** | 内存 etcd + kube-apiserver | 完整 Kubernetes 集群 |
| **Controller** | 同进程运行 | Pod 中运行 |
| **NETCONF 模拟器** | 不启动 | 完整 Deployment |
| **适用场景** | 日常开发、CI | 发布前验证、生产级测试 |
| **命令** | `make test-e2e-go` | `make test-e2e-kind` |

---

### 🚀 方式一：Go E2E (envtest) - 快速验证

使用 controller-runtime 的 envtest 框架，启动内存中的 etcd + kube-apiserver，Controller 同进程运行。

#### 环境准备

```bash
cd backend

# 安装 Ginkgo
make install-ginkgo

# 安装 envtest 二进制文件（etcd, kube-apiserver）
make install-envtest
```

#### 运行测试

```bash
# 基础版本
make test-e2e-go

# 详细输出
make test-e2e-go-verbose
```

测试覆盖：

| CRD 类型 | 测试覆盖 |
|----------|---------|
| BusinessSwitch | ✅ CRUD + 状态验证 |
| BusinessVlan | ✅ CRUD + 批量创建 |
| BusinessInterface | ✅ CRUD + Access/Trunk 模式 |
| BusinessRoute | ✅ CRUD + 静态路由配置 |
| NativeDeviceConfig | ✅ CRUD + 原生配置 |

---

### 🌐 方式二：Kind E2E - 完整 Kubernetes 集群

使用 Kind 创建完整 Kubernetes 集群，部署 CRD、Controller、NETCONF 模拟器，进行生产级端到端测试。

#### 环境准备

安装依赖：

- [Docker](https://docs.docker.com/get-docker/)
- [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)

#### 一键运行

```bash
cd backend
make test-e2e-kind
```

脚本自动执行：

1. 创建 Kind 集群 (`usmp-e2e`)
2. 部署所有 CRD
3. 部署 NETCONF 模拟器
4. 构建 Controller Docker 镜像
5. 加载镜像到 Kind 集群
6. 部署 Controller Manager
7. 运行 Ginkgo E2E 测试
8. 清理集群

#### 手动调试模式

```bash
# 只设置集群，不运行测试
make e2e-kind-setup

# 查看集群状态
make e2e-status

# 查看 Controller 日志
make e2e-logs

# 查看 NETCONF 模拟器日志
make e2e-simulator-logs

# 导出 kubeconfig 进行手动测试
make kind-export-kubeconfig
export KUBECONFIG=~/.kube/usmp-e2e-config

# 手动测试 CRD
kubectl apply -f config/samples/biz_v1_businessswitch.yaml
kubectl get businessswitches

# 清理
kind delete cluster --name usmp-e2e
```

---

### 📊 完整测试套件

```bash
# 运行所有测试（单元 + 集成 + E2E）
make test-all-full

# 查看测试帮助
make e2e-test-info
```

---

### 🔍 调试与故障排查

#### 常见问题

**问题 1：envtest 二进制文件不存在**

```
error: unable to start test environment: unable to find either etcd, kube-apiserver, or kubectl
```

**解决**：

```bash
make install-envtest
# 或手动
export TEST_ASSET_PATH=/usr/local/kubebuilder/bin
```

**问题 2：Kind 集群创建失败

**检查 Docker 资源是否足够，或使用：

```bash
docker system prune -a
```

**问题 3：Controller 无法启动**

```bash
# 查看 Pod 状态
kubectl --context=kind-usmp-e2e -n usmp-system get pods

# 查看详细事件
kubectl --context=kind-usmp-e2e -n usmp-system describe pod <pod-name>

# 查看日志
make e2e-logs
```

**问题 4：NETCONF 模拟器连接失败**

```bash
# 检查模拟器状态
kubectl --context=kind-usmp-e2e -n usmp-e2e get pods -l app=netconf-simulator

# 查看模拟器日志
make e2e-simulator-logs

# 端口转发测试
kubectl --context=kind-usmp-e2e -n usmp-e2e port-forward svc/netconf-simulator 830:830
```

---

### 📝 编写新的 E2E 测试

新增 CRD 测试步骤：

1. 在 `test/e2e/` 下创建 `yourcrd_test.go`
2. 使用 Ginkgo BDD 风格编写测试
3. 使用 `k8sClient` 进行 CRUD 操作
4. 使用 `Eventually` 等待异步状态更新
5. 在 `e2e_suite_test.go` 中注册 Schema

示例：

```go
var _ = Describe("YourCRD E2E Test", func() {
    It("应该成功创建资源", func() {
        obj := &yourv1.YourCRD{
            ObjectMeta: metav1.ObjectMeta{
                Name: "test-obj",
                Namespace: "usmp-e2e-test",
            },
            Spec: yourv1.YourCRDSpec{/* ... */},
        }
        Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
    })
})
```

---

### 🔄 CI/CD 集成

GitHub Actions 示例：

```yaml
name: E2E Tests
on: [pull_request]
jobs:
  e2e-go:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: cd backend && make install-envtest
      - run: cd backend && make test-e2e-go

  e2e-kind:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run Kind E2E
        uses: helm/kind-action@v1
        with:
          cluster_name: usmp-e2e
      - run: cd backend && make test-e2e-kind
```

---

## 🎯 核心特性

### 1. Kubernetes Native Operator

- **声明式配置**：`kubectl apply` 下发配置
- **自动 Reconcile**：Controller 自动对齐 desired ↔ actual state
- **状态机**：Pending → Updating → Ready → Failed 状态流转
- **事件记录**：Kubernetes Events 记录配置变更历史

### 2. 故障隔离

- 一台设备故障不影响其他设备
- NETCONF 连接池自动管理，断线自动重连
- 配置下发失败自动重试，保留原配置

### 3. CRD 动态表单（前端）

- 前端自动解析 CRD OpenAPI Schema
- 自动生成表单控件：
  - `boolean` → Switch 开关
  - `enum` → Select 下拉框
  - `int`/`uint` → Input 数字框
  - `string` → Input 文本框
  - `object` → 分组表单
  - `array` → Table 表格
- 无需手写表单，新增 CRD 字段自动生效

### 4. 两级 E2E 测试保障

- envtest 快速验证，日常开发秒级反馈
- Kind 完整集群测试，发布前生产级验证
- 所有 CRD 类型全覆盖测试
- CRUD + 状态机完整流程验证

## 📝 开发流程

本项目严格遵循 TDD 测试驱动开发 + 小步迭代：

1. `Plan` → 需求拆分，每个迭代一个原子功能
2. `Test` → 先写单元测试（正常/异常/并发场景）
3. `Code` → 实现功能（单次 ≤ 500 行）
4. `Review` → 自动代码评审
5. `E2E Test` → 添加 Kind + envtest 端到端测试
6. `Commit` → 标准 What/Why/How 三段式提交

## 📄 License

MIT License

## 🙏 致谢

- [Kubernetes SIGs](https://github.com/kubernetes-sigs) - controller-runtime, kubebuilder, envtest, kind
- [openconfig](https://github.com/openconfig) - YANG 模型标准
- [scrapligo](https://github.com/scrapli/scrapligo) - NETCONF 客户端
- [onsi/ginkgo](https://github.com/onsi/ginkgo) - BDD 测试框架
