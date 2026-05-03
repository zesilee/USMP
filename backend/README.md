# USMP Backend - 统一交换机管理平台后端

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Release-green.svg)]()

## 概述

USMP (Unified Switch Management Platform) 是一个基于 Kubernetes CRD 的声明式交换机配置管理平台，采用 YANG 模型驱动，支持多厂商设备。

**核心设计理念：**
- 🎯 **声明式配置** - 用户只需声明期望状态，平台自动协调到实际状态
- 🧩 **模型驱动** - 基于 YANG 标准，强类型配置，自动校验
- ☁️ **云原生架构** - 100% Kubernetes Native，CRD + Controller 模式
- 🔌 **无数据库** - 所有状态存储在 etcd，无额外运维成本

## 架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                    Kubernetes API Server                        │
│  ┌───────────────────────────────────────────────────────────┐  │
│  │  Custom Resource Definitions (CRD)                         │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐ │  │
│  │  │ Business │ │ Business │ │ Business │ │ NativeDevice │ │  │
│  │  │  Switch  │ │   Vlan   │ │Interface │ │    Config    │ │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────────┘ │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              ↓ Watch
┌─────────────────────────────────────────────────────────────────┐
│                      Controller Manager                         │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────────┐     │
│  │  Switch  │ │   Vlan   │ │Interface │ │NativeDevice  │     │
│  │ Controller│ │Controller│ │Controller│ │ Controller   │     │
│  └──────────┘ └──────────┘ └──────────┘ └──────────────┘     │
│  ┌───────────────────────────────────────────────────────────┐ │
│  │                    Translation Engine                     │ │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐                  │ │
│  │  │  Huawei  │ │  Cisco   │ │  H3C     │                  │ │
│  │  │  Trans   │ │  Trans   │ │  Trans   │                  │ │
│  │  └──────────┘ └──────────┘ └──────────┘                  │ │
│  └───────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
                              ↓ NETCONF
┌─────────────────────────────────────────────────────────────────┐
│                    Network Devices (Switches)                    │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                         │
│  │  CE6857  │ │  Nexus   │ │  S6800   │                         │
│  │ Huawei   │ │  Cisco   │ │   H3C    │                         │
│  └──────────┘ └──────────┘ └──────────┘                         │
└─────────────────────────────────────────────────────────────────┘
```

## 项目结构

```
backend/
├── api/                    # CRD API 定义
│   └── v1/               # v1 API 版本
│       ├── businessswitch_types.go
│       ├── businessvlan_types.go
│       ├── businessinterface_types.go
│       ├── businessroute_types.go
│       └── nativedeviceconfig_types.go
├── cmd/
│   └── controller/
│       └── main.go       # Controller Manager 入口
├── internal/
│   ├── api/              # REST API 处理
│   │   ├── server.go
│   │   ├── device_handler.go
│   │   ├── config_handler.go
│   │   └── yang_handler.go
│   └── controllers/      # CRD Controller 实现
│       ├── businessswitch_controller.go
│       ├── businessvlan_controller.go
│       ├── businessinterface_controller.go
│       ├── businessroute_controller.go
│       └── nativedeviceconfig_controller.go
├── pkg/
│   └── yang-runtime/     # YANG 运行时框架
│       ├── manager/
│       ├── client/       # NETCONF 客户端
│       └── ...
├── config/
│   ├── crd/              # CRD 定义
│   ├── samples/          # 示例 CR
│   └── rbac/             # RBAC 配置
├── docs/
│   ├── architecture/     # 架构文档
│   ├── api/              # API 文档
│   └── crd/              # CRD 使用文档
└── go.mod
```

## 快速开始

### 前置条件

- Kubernetes 集群 1.24+
- Go 1.21+
- Kubectl 已配置

### 1. 安装 CRD

```bash
# 安装所有 CRD
kubectl apply -f config/crd/bases/
```

### 2. 安装 RBAC 配置

```bash
kubectl apply -f config/rbac/role.yaml
```

### 3. 部署 Controller

```bash
# 构建镜像
docker build -t usmp-controller:latest .

# 部署（需要先准备部署 YAML）
kubectl apply -f config/manager/manager.yaml
```

### 4. 测试

```bash
# 创建示例交换机
kubectl apply -f config/samples/biz_v1_businessswitch.yaml

# 创建示例 VLAN
kubectl apply -f config/samples/biz_v1_businessvlan.yaml

# 查看状态
kubectl get businessswitches
kubectl get businessvlans
```

## CRD 资源概览

| 资源 | 简称 | API Group | 用途 |
|------|-----|-----------|------|
| BusinessSwitch | bs, switches | biz.usmp.io/v1 | 交换机设备注册与探活 |
| BusinessVlan | bv, vlans | biz.usmp.io/v1 | VLAN 配置管理 |
| BusinessInterface | bi, ifaces | biz.usmp.io/v1 | 接口配置管理 |
| BusinessRoute | br, routes | biz.usmp.io/v1 | 静态路由配置 |
| NativeDeviceConfig | ndc | biz.usmp.io/v1 | 原生配置透传（逃生舱） |

### BusinessSwitch 示例

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessSwitch
metadata:
  name: switch-core-01
spec:
  deviceIP: 192.168.1.100
  vendor: Huawei
  model: CE6857
  port: 830
  credentials:
    username: admin
    passwordSecretRef: switch-credentials/admin-password
  enabled: true
  syncInterval: 5
  description: 核心交换机 01
  location: 北京机房-A区-3机柜
  owner: network-team
```

### BusinessVlan 示例

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessVlan
metadata:
  name: vlan-100
spec:
  deviceID: switch-core-01
  vlanID: 100
  name: Business-VLAN
  description: 业务部门 VLAN
  type: Common
  adminStatus: Up
  macLearningEnabled: true
  statisticEnabled: true
```

## 文档索引

### 📚 用户文档

| 文档 | 说明 |
|------|------|
| [BusinessSwitch CRD 使用指南](docs/crd/businessswitch.md) | 交换机设备管理 |
| [BusinessVlan CRD 使用指南](docs/crd/businessvlan.md) | VLAN 配置管理 |
| [BusinessInterface CRD 使用指南](docs/crd/businessinterface.md) | 接口配置管理 |
| [BusinessRoute CRD 使用指南](docs/crd/businessroute.md) | 路由配置管理 |
| [NativeDeviceConfig CRD 使用指南](docs/crd/nativedeviceconfig.md) | 原生配置透传 |

### 🔌 API 文档

| 文档 | 说明 |
|------|------|
| [REST API 文档](docs/api/rest-api.md) | 完整的 REST API 接口参考 |

### 🏛️ 架构文档

| 文档 | 说明 |
|------|------|
| [架构概览](docs/architecture/overview.md) | 整体架构设计、核心组件、工作流程 |

## 核心特性

### ✅ 声明式配置

只需定义期望的配置状态，Controller 自动协调：

```bash
# 应用配置
kubectl apply -f my-vlan.yaml

# 查看同步状态
kubectl get bv vlan-100 -o jsonpath='{.status.phase}'
# Output: Synced
```

### ✅ 自动差异检测

自动检测期望配置与实际配置的差异：

```bash
# 查看差异
kubectl get bv vlan-100 -o jsonpath='{.status.diff}'
```

### ✅ 错误分类与重试

- **Temporary 错误** - 指数退避重试（5s → 10s → 20s → 40s → 60s）
- **Permanent 错误** - 停止重试，标记失败，每 60 分钟重新尝试

### ✅ Finalizer 清理

删除 CR 时自动清理设备上的配置：

```bash
kubectl delete bv vlan-100
# Controller 会自动向设备发送删除 VLAN 的配置
```

## 开发指南

### 本地开发

```bash
# Clone 项目
git clone https://github.com/your-org/usmp.git
cd usmp/backend

# 安装依赖
go mod download

# 运行测试
go test ./... -v

# 本地运行 Controller
go run cmd/controller/main.go
```

### 添加新的 CRD

```bash
# 1. 创建类型定义
# api/v1/newresource_types.go

# 2. 生成 CRD 配置
# (使用 controller-gen 或手动编写)

# 3. 创建 Controller
# internal/controllers/newresource_controller.go

# 4. 注册到 Manager
# cmd/controller/main.go
```

### 运行测试

```bash
# 单元测试
go test ./internal/... -v

# 集成测试（需要 NETCONF 模拟器）
go test ./test/integration/... -v -tags=integration

# 覆盖率
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 故障排查

### 常见问题

**Q: Controller 无法连接到设备？**
```bash
# 检查设备状态
kubectl get bs switch-name -o wide

# 查看详细状态
kubectl describe bs switch-name

# 检查网络连通性
kubectl exec -it <controller-pod> -- ping <device-ip>
```

**Q: 配置下发失败？**
```bash
# 查看 Status 中的错误信息
kubectl get bv vlan-name -o jsonpath='{.status.message}'

# 查看错误类型
kubectl get bv vlan-name -o jsonpath='{.status.errorType}'

# 查看 Controller 日志
kubectl logs -l control-plane=usmp-controller
```

**Q: CRD 安装失败？**
```bash
# 检查 CRD 是否已存在
kubectl get crd | grep usmp

# 强制重新安装
kubectl replace -f config/crd/bases/ --force
```

### 日志分析

```bash
# 查看 Controller 日志
kubectl logs -f deployment/usmp-controller-manager

# 只看 VLAN 相关日志
kubectl logs -f deployment/usmp-controller-manager | grep -i vlan

# 只看错误日志
kubectl logs -f deployment/usmp-controller-manager | grep -i error
```

## 性能指标

- **单 Controller 支持设备数**: 500+
- **配置下发延迟**: < 3s (P95)
- **同步间隔**: 5min (可配置)
- **内存占用**: < 512MB
- **CPU 占用**: < 1 core

## 路线图

- [x] v1.0 - CRD 基础框架、VLAN/接口/路由配置
- [x] v1.1 - 原生配置透传、加密配置支持
- [ ] v1.2 - Cisco/H3C 多厂商支持
- [ ] v1.3 - 配置版本管理与回滚
- [ ] v1.4 - 配置审计与变更历史
- [ ] v1.5 - 配置模板与批量操作
- [ ] v2.0 - gNMI 协议支持、Telemetry 采集

## 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启 Pull Request

## 许可证

Apache License 2.0 - 详见 [LICENSE](LICENSE) 文件。

## 联系方式

- 项目维护者: Network Team
- 邮箱: network-team@company.com
- Slack: #usmp-dev

---

**Made with ❤️ by the Network Automation Team**
