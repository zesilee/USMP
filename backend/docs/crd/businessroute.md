# BusinessRoute CRD 使用文档

## 概述

BusinessRoute CRD 用于管理交换机上的静态路由配置，包括静态路由、默认路由、黑洞路由等功能，支持 BFD 检测和路由策略。

**API Group**: `biz.usmp.io/v1`

**资源名称**: `businessroutes`

**简称**: `br`, `routes`

## Spec 字段说明

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|-----|--------|------|
| `deviceID` | string | ✅ | - | 所属交换机设备 ID（对应 BusinessSwitch 的 name） |
| `type` | string | - | Static | 路由类型：`Static` / `Default` / `Blackhole` |
| `destinationCIDR` | string | ✅ | - | 目标网络（CIDR 格式，如 192.168.1.0/24） |
| `nextHopType` | string | - | IPAddress | 下一跳类型：`IPAddress` / `Interface` |
| `nextHopIP` | string | - | - | 下一跳 IP 地址（当 NextHopType=IPAddress 时必填） |
| `outInterface` | string | - | - | 出接口名称（当 NextHopType=Interface 时必填） |
| `preference` | uint8 | - | 60 | 路由优先级（值越小优先级越高，1-255） |
| `tag` | uint32 | - | 0 | 路由标签（用于路由策略匹配） |
| `description` | string | - | - | 描述信息 |
| `bfdEnabled` | bool | - | false | 是否启用 BFD 检测 |
| `bfdSessionName` | string | - | - | BFD 会话名称 |
| `permanent` | bool | - | false | 是否为永久路由（接口 Down 时不删除） |
| `advertise` | bool | - | false | 路由是否发布到其他协议 |

## Status 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `phase` | string | 同步阶段：`Pending` / `Syncing` / `Synced` / `Failed` |
| `lastSyncTime` | Time | 最后同步时间 |
| `message` | string | 同步消息或错误信息 |
| `retryCount` | int | 重试次数 |
| `errorType` | string | 错误类型：`Temporary` / `Permanent` |
| `routeStatus` | string | 路由实际状态：`Active` / `Inactive` / `Failed` |
| `outInterfaceStatus` | string | 出接口状态（Up/Down） |
| `nextHopReachable` | bool | 下一跳可达性 |
| `actualPreference` | uint8 | 路由的实际优先级 |
| `nextHopMac` | string | 下一跳的 MAC 地址 |

## 示例

### 基本静态路由（下一跳 IP）

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessRoute
metadata:
  name: static-route-10
spec:
  deviceID: switch-demo-01
  type: Static
  destinationCIDR: 10.0.0.0/8
  nextHopType: IPAddress
  nextHopIP: 192.168.1.254
  description: 办公网出口路由
  preference: 60
  tag: 100
```

### 默认路由

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessRoute
metadata:
  name: default-route
spec:
  deviceID: switch-demo-01
  type: Default
  destinationCIDR: 0.0.0.0/0
  nextHopType: IPAddress
  nextHopIP: 172.16.0.1
  description: 互联网出口默认路由
  preference: 60
  permanent: true
```

### 出接口静态路由

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessRoute
metadata:
  name: direct-route-vlan100
spec:
  deviceID: switch-demo-01
  type: Static
  destinationCIDR: 192.168.100.0/24
  nextHopType: Interface
  outInterface: Vlanif100
  description: 直连网段路由
  preference: 10
```

### 黑洞路由

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessRoute
metadata:
  name: blackhole-route
spec:
  deviceID: switch-demo-01
  type: Blackhole
  destinationCIDR: 10.20.0.0/16
  description: 废弃网段黑洞路由（防环路）
  preference: 250
```

### 带 BFD 检测的静态路由

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessRoute
metadata:
  name: bfd-route-core
spec:
  deviceID: switch-demo-01
  type: Static
  destinationCIDR: 172.31.0.0/16
  nextHopType: IPAddress
  nextHopIP: 192.168.255.254
  description: 核心链路路由（带 BFD 检测）
  preference: 30
  bfdEnabled: true
  bfdSessionName: core-link-bfd
  permanent: true
```

### 浮动静态路由（主备链路）

```yaml
# 主路由（高优先级）
apiVersion: biz.usmp.io/v1
kind: BusinessRoute
metadata:
  name: primary-route
spec:
  deviceID: switch-demo-01
  type: Static
  destinationCIDR: 10.100.0.0/16
  nextHopIP: 192.168.1.1
  preference: 40
  bfdEnabled: true
  description: 主链路路由

---
# 备路由（低优先级，主链路故障时生效）
apiVersion: biz.usmp.io/v1
kind: BusinessRoute
metadata:
  name: backup-route
spec:
  deviceID: switch-demo-01
  type: Static
  destinationCIDR: 10.100.0.0/16
  nextHopIP: 192.168.1.2
  preference: 80
  description: 备链路路由（浮动路由）
```

## kubectl 常用操作

### 查看所有路由

```bash
kubectl get businessroutes
kubectl get br  # 简写
```

输出示例：
```
NAME               DEVICE          DESTINATION       NEXTHOP          PHASE    STATUS  AGE
static-route-10    switch-demo-01  10.0.0.0/8        192.168.1.254    Synced   Active  1h
default-route      switch-demo-01  0.0.0.0/0         172.16.0.1       Synced   Active  45m
blackhole-route    switch-demo-01  10.20.0.0/16                        Synced   Active  30m
```

### 查看特定路由详情

```bash
kubectl describe br default-route
```

### 查看特定设备的所有路由

```bash
kubectl get br -o wide | grep switch-demo-01
```

### 查看所有默认路由

```bash
kubectl get br -o jsonpath='{.items[?(@.spec.type=="Default")].metadata.name}'
```

### 查看所有活跃路由

```bash
kubectl get br -o wide | grep Active
```

### 检查下一跳可达性

```bash
kubectl get br primary-route -o jsonpath='{.status.nextHopReachable}'
```

### 删除路由

```bash
kubectl delete br blackhole-route
```

> **注意**：删除 BusinessRoute CR 时，Controller 会从设备上移除对应的路由配置。

## 控制器行为

### Reconcile 流程

```
1. CR 创建/更新事件
   ↓
2. Finalizer 检查（删除时移除路由）
   ↓
3. 验证 CIDR 格式和下一跳配置
   ↓
4. 验证出接口存在（如果指定）
   ↓
5. 翻译器转换为厂商 YANG 模型
   ↓
6. NETCONF edit-config 下发
   ↓
7. 从设备读取路由表，验证路由已生效
   ↓
8. 检测下一跳可达性（ARP 检测）
   ↓
9. 更新 Status: RouteStatus=Active/Inactive
   ↓
10. 返回 RequeueAfter = 5分钟（定期检测可达性）
```

### Finalizer 清理

删除 CR 时，Controller 会：
1. 从设备路由表中移除该路由
2. 如果启用了 BFD，删除对应的 BFD 会话
3. 记录日志
4. 移除 Finalizer，完成删除

## 排错指南

### 路由下发失败

1. 查看 Status 错误信息：
   ```bash
   kubectl get br static-route-10 -o jsonpath='{.status.message}'
   ```

2. 检查下一跳 IP 是否可达：
   ```bash
   kubectl get br static-route-10 -o jsonpath='{.status.nextHopReachable}'
   ```

3. 验证 CIDR 格式是否正确（必须包含网络地址和掩码长度）

### 路由状态为 Inactive

1. 检查出接口状态：
   ```bash
   kubectl get br direct-route-vlan100 -o jsonpath='{.status.outInterfaceStatus}'
   ```

2. 检查下一跳 ARP 是否学习到：
   ```bash
   kubectl get br static-route-10 -o jsonpath='{.status.nextHopMac}'
   ```

3. 检查是否有更高优先级的同目的网段路由

### BFD 会话未建立

1. 确认对端设备已配置对应的 BFD 会话
2. 检查 BFD 参数是否匹配（检测间隔、倍数等）
3. 确认 BFD 控制报文可以在链路中传输

### 浮动路由未切换

1. 确认主路由优先级低于备路由
2. 确认主路由已启用 BFD 或能正确检测链路故障
3. 查看设备路由表确认路由切换情况

## 最佳实践

### 1. 路由优先级规划

```
直连路由:  0-10  (最高优先级，不可配置)
静态路由:  60    (默认)
OSPF:     110
BGP:      200
浮动路由:  80-100 (作为备用)
黑洞路由:  250   (最低优先级，兜底)
```

### 2. 高可用性配置

关键链路上启用 BFD 快速检测：
```yaml
bfdEnabled: true
preference: 30          # 提高优先级
permanent: true         # 接口 Down 不删除路由
```

### 3. 使用 Tag 进行路由策略管理

```yaml
# 标记需要发布到 BGP 的路由
tag: 100
advertise: true

# 在路由策略中匹配 tag 100 的路由进行发布
```

### 4. 使用 Label 分组管理

```yaml
metadata:
  labels:
    route-type: static
    network: office
    link: primary
    environment: production
```

批量操作示例：
```bash
# 查看所有办公网路由
kubectl get br -l network=office

# 批量移除测试环境路由
kubectl get br -l environment=test -o name | xargs kubectl delete
```

### 5. 黑洞路由防环路

对于不再使用的网段，配置黑洞路由防止环路：
```yaml
type: Blackhole
destinationCIDR: 10.20.0.0/16
preference: 250
```

### 6. 默认路由注意事项

- 避免在网络中存在多个默认路由造成环路
- 使用不同的 preference 实现主备默认路由
- 核心交换机配置默认路由指向出口防火墙/路由器

## 华为路由转换映射

| BusinessRoute Spec | Huawei YANG 字段 | 说明 |
|-------------------|-----------------|------|
| `destinationCIDR` | `ipRoutePrefix` | 目标网段 |
| `nextHopIP` | `nextHopAddress` | 下一跳 IP |
| `outInterface` | `outInterfaceName` | 出接口 |
| `preference` | `preferenceValue` | 优先级 |
| `tag` | `routeTag` | 路由标签 |
| `bfdEnabled` | `bfdEnable` | BFD 开关 |
