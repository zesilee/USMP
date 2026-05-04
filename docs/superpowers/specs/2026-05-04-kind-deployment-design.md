# USMP Kind 集群部署设计文档

**日期**: 2026-05-04  
**作者**: USMP Team  
**版本**: v1.0  
**目标环境**: 开发/演示环境

## 1. 概述

本文档描述了将 USMP（Universal Switch Management Platform）完整部署到 Kind（Kubernetes IN Docker）集群的架构设计和实现方案。目标是提供一个一键启动的本地开发和演示环境。

## 2. 设计目标

### 2.1 核心目标
- ✅ **一键部署**: 一条命令即可启动完整环境
- ✅ **开发友好**: 支持快速迭代和调试
- ✅ **演示就绪**: 适合功能展示和客户演示
- ✅ **与 E2E 测试复用**: 尽量复用现有 E2E 测试的配置
- ✅ **极简架构**: NodePort 方式，无需 Ingress/MetalLB 等额外组件

### 2.2 非目标
- ❌ 高可用配置（单实例即可）
- ❌ HTTPS 支持（演示环境用 HTTP 即可）
- ❌ 持久化存储（配置都存在 CRD 中，etcd 足够）

## 3. 整体架构

### 3.1 部署架构图

```
                            ┌─────────────────────────────────────────────────────┐
                            │                   Kind 集群                          │
                            │  ┌───────────────────────────────────────────────┐  │
用户浏览器 ──:30081─────────┤  │         usmp-system Namespace                  │  │
   │                        │  │                                               │  │
   │                        │  │  ┌──────────┐     ┌──────────────────────┐   │  │
   │                        │  │  │  前端    │     │  Controller Manager  │   │  │
   │                        │  │  │  Vue3    │     │  (Go + Gin)          │   │  │
   │                        │  │  └──────────┘     └──────────┬───────────┘   │  │
   │                        │  │       │                         │             │  │
   │                        │  │       │                         │ NETCONF     │  │
   │                        │  │       │                         ▼             │  │
   │                        │  │       │              ┌──────────────────┐    │  │
   └────────:6443───────────┼──┼───────┴─────────────▶│  NETCONF 模拟器  │    │  │
       K8s API Server       │  │    K8s API 访问       └──────────────────┘    │  │
                            │  └───────────────────────────────────────────────┘  │
                            └─────────────────────────────────────────────────────┘
```

### 3.2 组件清单

| 组件 | 类型 | 镜像 | 容器端口 | NodePort | 说明 |
|------|------|------|---------|----------|------|
| **前端** | Deployment | nginx:alpine + 静态资源 | 80 | 30081 | Vue3 动态表单界面 |
| **Controller** | Deployment | usmp-controller:latest | 8080/8081 | 30080/30081 | Operator 控制器 + REST API + Health Check |
| **NETCONF 模拟器** | Deployment | golang:1.21-alpine | 830 | 30830 | 模拟交换机设备 |
| **CRD** | CRD | - | - | - | 5 种业务 CRD |
| **前端 SA** | ServiceAccount | - | - | - | 前端访问 K8s API 用 |
| **前端 ClusterRole** | ClusterRole | - | - | - | CRD 读写权限 |

### 3.3 访问矩阵

| 目标 | 宿主机访问地址 | 集群内部访问 |
|------|---------------|-------------|
| 前端界面 | http://localhost:30081 | http://frontend.usmp-system.svc.cluster.local |
| 后端 API | http://localhost:30080 | http://controller.usmp-system.svc.cluster.local:8080 |
| K8s API Server | https://localhost:6443 | https://kubernetes.default.svc |
| NETCONF 模拟器 | localhost:30830 | netconf-simulator.usmp-system.svc:830 |

## 4. 详细设计

### 4.1 前端部署设计

#### 4.1.1 构建策略
- **多阶段构建**: Node.js 构建 → Nginx 运行
- **Dockerfile 位置**: `frontend/Dockerfile`

```dockerfile
# 构建阶段
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# 运行阶段
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
```

#### 4.1.2 K8s API 访问配置
前端需要从浏览器直接访问 K8s API，通过以下方式：
1. Kind 集群默认将 API Server 暴露在 `localhost:6443`
2. 前端使用 ServiceAccount Token 进行认证
3. 通过 ConfigMap 注入 K8s API 地址和 SA Token

#### 4.1.3 Nginx 配置要点
```nginx
events {
    worker_connections 1024;
}

http {
    server {
        listen 80;
        root /usr/share/nginx/html;
        index index.html;
        
        # SPA 路由支持
        location / {
            try_files $uri $uri/ /index.html;
        }
        
        # 静态资源缓存
        location ~* \.(js|css|png|jpg|jpeg|gif|ico)$ {
            expires 1y;
            add_header Cache-Control "public, immutable";
        }
    }
}
```

### 4.2 Controller 部署设计

#### 4.2.1 复用现有配置
- 复用 `backend/Dockerfile`
- 复用 `backend/config/rbac/` 中的 RBAC 配置
- 复用 `backend/config/manager/manager.yaml` 作为基础

#### 4.2.2 部署要点
- **健康检查**: livenessProbe + readinessProbe 使用 `/healthz` 和 `/readyz`
- **资源限制**: requests: 100m CPU / 128Mi Memory, limits: 500m CPU / 256Mi Memory
- **日志级别**: debug 级别，便于开发调试
- **镜像拉取策略**: IfNotPresent（本地加载镜像）

### 4.3 NETCONF 模拟器设计

#### 4.3.1 复用现有配置
直接复用 `backend/test/e2e/config/netconf-simulator.yaml`，但调整：
- Namespace 从 `usmp-e2e` 改为 `usmp-system`
- 保留内联 Go 代码的方式，无需额外镜像
- NodePort 保持 30830 不变

### 4.4 RBAC 设计

#### 4.4.1 前端 ClusterRole
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: usmp-frontend-role
rules:
  # 业务 CRD 完整读写权限
  - apiGroups: ["biz.usmp.io"]
    resources: ["businessvlans", "businessinterfaces", "businessroutes", "businessswitches"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  
  # 原生配置 CRD 完整读写权限
  - apiGroups: ["core.usmp.io"]
    resources: ["nativedeviceconfigs"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  
  # CRD 定义读取权限（用于 Schema 解析）
  - apiGroups: ["apiextensions.k8s.io"]
    resources: ["customresourcedefinitions"]
    verbs: ["get", "list", "watch"]
```

#### 4.4.2 Controller RBAC
- 由 controller-gen 自动生成，位于 `backend/config/rbac/role.yaml`
- Kustomize 直接引用

## 5. 目录结构

```
usmp/
├── backend/
│   ├── deploy/
│   │   ├── kind-cluster.yaml          # Kind 集群配置
│   │   └── manifests/
│   │       ├── namespace.yaml         # usmp-system 命名空间
│   │       ├── kustomization.yaml     # Kustomize 入口
│   │       ├── controller/
│   │       │   ├── service_account.yaml
│   │       │   ├── role.yaml (引用 config/rbac)
│   │       │   ├── role_binding.yaml
│   │       │   ├── deployment.yaml
│   │       │   └── service.yaml
│   │       ├── frontend/
│   │       │   ├── service_account.yaml
│   │       │   ├── cluster_role.yaml
│   │       │   ├── cluster_role_binding.yaml
│   │       │   ├── configmap.yaml
│   │       │   ├── deployment.yaml
│   │       │   └── service.yaml
│   │       └── netconf-simulator/
│   │           ├── configmap.yaml
│   │           ├── deployment.yaml
│   │           └── service.yaml
│   └── Makefile (新增 kind-* 目标)
│
└── frontend/
    ├── Dockerfile
    └── nginx.conf
```

## 6. 部署流程

### 6.1 快速开始
```bash
# 1. 构建镜像
cd backend && make docker-build
cd frontend && docker build -t usmp-frontend:latest .

# 2. 创建集群并部署
cd backend && make kind-up

# 3. 加载镜像到 Kind
kind load docker-image usmp-controller:latest --name usmp-dev
kind load docker-image usmp-frontend:latest --name usmp-dev

# 4. 访问前端
open http://localhost:30081
```

### 6.2 Makefile 目标

| 目标 | 说明 |
|------|------|
| `make kind-up` | 创建集群 + 部署所有组件 |
| `make kind-cluster` | 仅创建 Kind 集群 |
| `make kind-deploy` | 部署/更新所有组件 |
| `make kind-clean` | 删除 Kind 集群 |
| `make kind-status` | 查看集群状态 |
| `make kind-logs` | 查看 Controller 日志 |
| `make kind-load-images` | 加载本地镜像到 Kind |

## 7. 故障排查

### 7.1 前端无法连接 K8s API
- 检查浏览器是否能访问 https://localhost:6443
- 检查 ServiceAccount Token 是否正确注入
- 检查 Kind 集群上下文是否正确

### 7.2 Controller 无法连接 NETCONF 模拟器
- 检查模拟器 Pod 是否就绪：`kubectl -n usmp-system get pods`
- 检查 DNS 解析：`netconf-simulator.usmp-system.svc.cluster.local`
- 检查端口 830 是否可达

### 7.3 镜像拉取失败
- 确保镜像已加载到 Kind：`make kind-load-images`
- 检查 imagePullPolicy: IfNotPresent

## 8. 后续优化方向（可选）

1. **Tilt/Skaffold 支持**: 实现开发模式的热重载
2. **Ingress 选项**: 提供可选的 Ingress 部署方式
3. **数据持久化**: 模拟器配置持久化（如需要）
4. **多节点部署**: 支持多节点 Kind 集群
5. **监控集成**: Prometheus + Grafana 可选部署
