# USMP 架构概览

USMP (Unified Switch Management Platform) 是一个基于 Kubernetes CRD 的声明式交换机配置管理平台。

## 设计理念

- **声明式配置**：用户只需声明期望状态，控制器自动调和到实际状态
- **模型驱动**：基于 YANG 模型的配置管理，支持多厂商适配
- **云原生架构**：100% Kubernetes Native，使用 CRD + Controller 模式
- **无数据库**：所有状态存储在 etcd（Kubernetes API Server 背后），无需额外数据库

## 整体架构

```
┌─────────────────────────────────────────────────────────────────────┐
│                            Kubernetes API Server                      │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  Custom Resource Definitions (CRD)                             │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │  │
│  │  │BusinessSwitch│  │BusinessVlan  │  │BusinessIface │ ...    │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘         │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                              ↓  Watch (ListAndWatch)
┌─────────────────────────────────────────────────────────────────────┐
│                      Controller Manager (本项目)                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │
│  │  SwitchCtrl  │  │   VlanCtrl   │  │   IfaceCtrl  │  ...          │
│  └──────────────┘  └──────────────┘  └──────────────┘               │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │                     Translation Engine                         │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │  │
│  │  │Huawei Trans  │  │Cisco Trans   │  │H3C Trans    │         │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘         │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                              ↓  NETCONF/SSH
┌─────────────────────────────────────────────────────────────────────┐
│                        Network Devices (交换机)                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐               │
│  │ Huawei CE68  │  │ Cisco Nexus  │  │ H3C S6800    │  ...          │
│  └──────────────┘  └──────────────┘  └──────────────┘               │
└─────────────────────────────────────────────────────────────────────┘
```

## 核心组件说明

### 1. Controller 层

每个 CRD 对应一个 Controller，负责：
- 监听 CR 资源的创建/更新/删除事件
- 执行 Reconcile 调和逻辑
- 处理 Finalizer 清理逻辑
- 错误分类 + 指数退避重试
- Status 状态回写

#### Controller 公共能力

| 能力 | 说明 |
|------|------|
| **Finalizer** | 删除前的资源清理 |
| **状态管理** | Pending → Syncing → Synced/Failed |
| **错误分类** | Temporary（可重试）/ Permanent（不可重试） |
| **指数退避** | 5s → 10s → 20s → 40s → 60s，最大 5 次 |
| **状态回写** | 设备实际状态回填到 CR Status |

### 2. 翻译引擎 (Translator)

翻译引擎是业务 CRD 到厂商原生配置的核心转换层：

```
BusinessVlanSpec
      ↓
[Translator]
      ↓
HuaweiVlan_Vlan_Vlans (YANG Go Struct)
      ↓
[NETCONF Encoding]
      ↓
<edit-config> XML Payload
      ↓
交换机设备
```

#### 翻译器接口

```go
type Translator interface {
    Vendor() VendorType
    TranslateVlan(spec interface{}) (interface{}, error)
    TranslateInterface(spec interface{}) (interface{}, error)
    TranslateRoute(spec interface{}) (interface{}, error)
    TranslateSystem(spec interface{}) (interface{}, error)
    Validate(configType ConfigType, spec interface{}) error
}
```

### 3. NETCONF Client Pool

设备连接池，管理到所有交换机的 NETCONF 连接：
- 连接复用，避免每次配置都建立新连接
- 自动重连机制
- 连接数限制和超时控制

## CRD 设计原则

### 5 个业务 CRD 的定位

| CRD | 定位 | 用途举例 |
|-----|------|---------|
| **BusinessSwitch** | 交换机管理元数据 | 设备IP、认证信息、厂商型号、同步间隔 |
| **BusinessVlan** | VLAN 业务配置 | VLAN ID、名称、描述、端口列表、MAC 学习开关 |
| **BusinessInterface** | 接口业务配置 | Access/Trunk 模式、Native VLAN、MTU、速率 |
| **BusinessRoute** | 静态路由配置 | 目的网段、下一跳、优先级、BFD 检测 |
| **NativeDeviceConfig** | 原生配置透传 | 厂商特有命令、调试脚本、翻译引擎不支持的场景 |

### CRD 分层设计

```
┌─────────────────────────────────────────────────────┐
│   NativeDeviceConfig (原生配置 - 逃生舱)              │
│     CLI/YANG/XML/JSON 直接透传                       │
├─────────────────────────────────────────────────────┤
│   BusinessRoute / BusinessInterface / BusinessVlan   │
│     业务语义抽象 + 翻译引擎 → 厂商 YANG               │
├─────────────────────────────────────────────────────┤
│   BusinessSwitch (设备管理)                           │
│     设备注册、认证信息、探活状态                      │
└─────────────────────────────────────────────────────┘
```

## Reconcile 调和流程

以 BusinessVlan 为例：

```
1. 用户 kubectl apply 创建/更新 BusinessVlan CR
   ↓
2. VLAN Controller 接收到 Watch 事件，进入 Reconcile
   ↓
3. 检查 DeletionTimestamp → 有则执行 Finalizer 清理
   ↓
4. 添加 Finalizer（如果没有）
   ↓
5. 初始化 Status → Phase = Pending
   ↓
6. 调用翻译器 Validate() 验证配置合法性
   ✗ 验证失败 → Phase = Failed + ErrorType = Permanent → 结束
   ↓ ✓
7. 调用翻译器 TranslateVlan() 转为厂商 YANG
   ↓
8. Phase = Syncing，更新 Status
   ↓
9. 通过 NETCONF Client Pool 获取设备连接
   ↓
10. 下发 edit-config 配置
    ✗ 下发失败 → 错误分类 + 指数退避重入 Reconcile
    ↓ ✓
11. 从设备读取实际配置，回填 Status 字段
    - ActualVlanID
    - ActualPorts
    - OperStatus
    ↓
12. Phase = Synced，LastSyncTime = now，更新 Status
    ↓
13. 返回 RequeueAfter = 5分钟（定期校验配置一致性）
```

## 错误处理机制

### 错误分类

| 错误类型 | 触发条件 | 处理策略 |
|---------|---------|---------|
| **Temporary** | 网络超时、连接池满、设备临时忙 | 指数退避重试（5次） |
| **Permanent** | 认证失败、配置语法错误、权限不足 | 标记失败，每60分钟重试一次 |

### 指数退避算法

```
重试次数: 1 →  5秒
重试次数: 2 → 10秒
重试次数: 3 → 20秒
重试次数: 4 → 40秒
重试次数: 5 → 60秒
达到5次 → 停止重试，Phase = Failed，RequeueAfter = 60分钟
```

## 技术选型

| 组件 | 技术选型 | 说明 |
|------|---------|------|
| **CRD 框架** | controller-runtime | Kubernetes 官方 Controller 框架 |
| **配置模型** | YANG + ygot | 标准化网络配置模型，自动生成 Go 结构体 |
| **南向协议** | NETCONF (RFC 6241) | 交换机标准管理协议，支持事务、校验回滚 |
| **前端框架** | Vue 3 + Element Plus | 现代化前端，动态表单基于 YANG Schema 生成 |
| **存储** | Kubernetes etcd | 无额外数据库，所有 CR 状态存储在 etcd |

## 扩展路径

### 多厂商支持

新增厂商只需：
1. 实现 `Translator` 接口
2. 注册厂商翻译器到 Factory
3. 厂商特有协议适配（如 Cisco 的 SSH + CLI）

### 新配置类型

新增配置类型只需：
1. 定义 CRD Spec/Status
2. 实现 Controller Reconcile
3. 在翻译引擎中添加对应转换方法

### 北向扩展

支持的北向对接方式：
- Kubectl / Kubernetes Client Go
- Terraform Provider（基于 CRD 自动生成）
- Kubernetes Operator（封装更复杂的业务场景）
- API Gateway（可选，通过 Kubernetes Aggregation Layer）
