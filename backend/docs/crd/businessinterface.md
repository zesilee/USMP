# BusinessInterface CRD 使用文档

## 概述

BusinessInterface CRD 用于管理交换机接口的配置，包括接口模式、VLAN 成员、IP 地址、速率等功能。

**API Group**: `biz.usmp.io/v1`

**资源名称**: `businessinterfaces`

**简称**: `bi`, `ifaces`, `interfaces`

## Spec 字段说明

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|-----|--------|------|
| `deviceID` | string | ✅ | - | 所属交换机设备 ID（对应 BusinessSwitch 的 name） |
| `interfaceName` | string | ✅ | - | 接口名称（如 GigabitEthernet0/0/1） |
| `description` | string | - | - | 接口描述 |
| `mode` | string | - | Access | 接口模式：`Access` / `Trunk` / `Hybrid` / `L3` / `L2` |
| `adminStatus` | string | - | Up | 管理状态：`Up` / `Down` |
| `accessVlan` | uint16 | - | 1 | Access VLAN（仅 Access 模式有效） |
| `trunkAllowedVlans` | []VlanConfig | - | [] | Trunk 允许通过的 VLAN 列表 |
| `nativeVlan` | uint16 | - | 1 | Native VLAN ID（仅 Trunk/Hybrid 模式有效） |
| `ipAddress` | string | - | - | 三层接口 IP 地址（仅 L3 模式有效） |
| `netmask` | string | - | - | 子网掩码（仅 L3 模式有效） |
| `mtu` | uint32 | - | 1500 | MTU 值 |
| `speed` | uint32 | - | 0 | 速率配置（Mbps），0 表示自动协商 |
| `duplex` | string | - | auto | 双工模式：`auto` / `full` / `half` |
| `lldpEnabled` | bool | - | true | 是否启用 LLDP |
| `stormControlEnabled` | bool | - | false | 是否启用风暴控制 |

### VlanConfig 子字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `vlanID` | uint16 | VLAN ID |
| `isNative` | bool | 是否是 Native VLAN |

## Status 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `phase` | string | 同步阶段：`Pending` / `Syncing` / `Synced` / `Failed` |
| `lastSyncTime` | Time | 最后同步时间 |
| `message` | string | 同步消息或错误信息 |
| `retryCount` | int | 重试次数 |
| `errorType` | string | 错误类型：`Temporary` / `Permanent` |
| `operStatus` | string | 运行状态：`Up` / `Down` / `Testing` / `Unknown` / `Dormant` / `NotPresent` |
| `interfaceType` | string | 接口类型（从设备读取） |
| `physAddress` | string | 物理地址/MAC |
| `actualSpeed` | uint32 | 实际速率（Mbps） |
| `actualMTU` | uint32 | 实际 MTU |
| `counters` | InterfaceCounters | 统计信息 |
| `actualAccessVlan` | uint16 | 设备上实际配置的 Access VLAN |
| `actualTrunkVlans` | []uint16 | 设备上实际配置的 Trunk VLAN 列表 |
| `actualNativeVlan` | uint16 | 设备上实际配置的 Native VLAN |

### InterfaceCounters 子字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `inOctets` | uint64 | 入方向字节数 |
| `outOctets` | uint64 | 出方向字节数 |
| `inPackets` | uint64 | 入方向数据包数 |
| `outPackets` | uint64 | 出方向数据包数 |
| `inErrors` | uint32 | 入方向错误数 |
| `outErrors` | uint32 | 出方向错误数 |
| `inDiscards` | uint32 | 入方向丢弃数 |
| `outDiscards` | uint32 | 出方向丢弃数 |

## 示例

### Access 模式接口

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessInterface
metadata:
  name: access-port-01
spec:
  deviceID: switch-demo-01
  interfaceName: GigabitEthernet0/0/1
  description: 服务器接入端口
  mode: Access
  adminStatus: Up
  accessVlan: 100
  mtu: 1500
  speed: 1000
  duplex: full
  lldpEnabled: true
```

### Trunk 模式接口

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessInterface
metadata:
  name: trunk-port-01
spec:
  deviceID: switch-demo-01
  interfaceName: GigabitEthernet0/0/24
  description: 上行 Trunk 端口
  mode: Trunk
  adminStatus: Up
  nativeVlan: 1
  trunkAllowedVlans:
    - vlanID: 10
      isNative: false
    - vlanID: 20
      isNative: false
    - vlanID: 100
      isNative: false
  mtu: 9216
  speed: 10000
  stormControlEnabled: true
```

### 三层接口（L3 模式）

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessInterface
metadata:
  name: vlanif-100
spec:
  deviceID: switch-demo-01
  interfaceName: Vlanif100
  description: 业务网段网关
  mode: L3
  adminStatus: Up
  ipAddress: 192.168.100.1
  netmask: 255.255.255.0
  mtu: 1500
```

### 禁用的接口

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessInterface
metadata:
  name: unused-port-48
spec:
  deviceID: switch-demo-01
  interfaceName: GigabitEthernet0/0/48
  description: 未使用端口（管理性关闭）
  mode: Access
  adminStatus: Down
  accessVlan: 999
```

## kubectl 常用操作

### 查看所有接口

```bash
kubectl get businessinterfaces
kubectl get bi  # 简写
```

输出示例：
```
NAME             DEVICE          INTERFACE              MODE     PHASE    STATUS  AGE
access-port-01   switch-demo-01  GigabitEthernet0/0/1  Access   Synced   Up      1h
trunk-port-01    switch-demo-01  GigabitEthernet0/0/24 Trunk    Synced   Up      30m
vlanif-100       switch-demo-01  Vlanif100              L3       Synced   Up      15m
```

### 查看特定接口详情

```bash
kubectl describe bi trunk-port-01
```

### 查看接口实时统计信息

```bash
kubectl get bi access-port-01 -o jsonpath='{.status.counters}'
```

### 查看特定设备的所有接口

```bash
kubectl get bi -o wide | grep switch-demo-01
```

### 查看所有 Down 状态的接口

```bash
kubectl get bi -o wide | grep -E 'Down|Failed'
```

### 删除接口配置

```bash
kubectl delete bi unused-port-48
```

> **注意**：删除 BusinessInterface CR 时，Controller 会将接口恢复到默认配置（Access 模式，VLAN 1，Shutdown）。

## 控制器行为

### Reconcile 流程

```
1. CR 创建/更新事件
   ↓
2. Finalizer 检查（删除时恢复默认配置）
   ↓
3. 验证接口模式与配置项的匹配性
   ↓
4. 翻译器转换为厂商 YANG 模型
   ↓
5. NETCONF edit-config 下发
   ↓
6. 从设备读取实际接口状态
   ↓
7. 更新 counters 统计信息
   ↓
8. 计算期望与实际配置的差异
   ↓
9. 更新 Status: Phase=Synced
   ↓
10. 30秒后再次 Reconcile 刷新统计信息
```

### Finalizer 清理

删除 CR 时，Controller 会：
1. 将接口设置为 Access 模式，VLAN 1
2. Shutdown 接口（adminStatus=Down）
3. 清除所有 IP、VLAN、速率配置
4. 移除 Finalizer，完成删除

## 排错指南

### 接口配置下发失败

1. 查看 Status 错误信息：
   ```bash
   kubectl get bi trunk-port-01 -o jsonpath='{.status.message}'
   ```

2. 检查错误类型：
   ```bash
   kubectl get bi trunk-port-01 -o jsonpath='{.status.errorType}'
   ```

3. 检查设备是否在线：
   ```bash
   kubectl get bs switch-demo-01
   ```

### 接口物理状态 Down

1. 检查管理状态：
   ```bash
   kubectl get bi access-port-01 -o jsonpath='{.spec.adminStatus}'
   ```

2. 检查实际运行状态：
   ```bash
   kubectl get bi access-port-01 -o jsonpath='{.status.operStatus}'
   ```

3. 检查速率和双工模式配置是否与对端匹配

### Trunk VLAN 成员未生效

1. 检查接口模式是否为 Trunk：
   ```bash
   kubectl get bi trunk-port-01 -o jsonpath='{.spec.mode}'
   ```

2. 查看设备实际配置：
   ```bash
   kubectl get bi trunk-port-01 -o jsonpath='{.status.actualTrunkVlans}'
   ```

3. 确认 VLAN 是否已在设备上创建

### IP 地址配置失败

1. 确认接口模式为 L3：
   ```bash
   kubectl get bi vlanif-100 -o jsonpath='{.spec.mode}'
   ```

2. 检查 IP 格式是否正确
3. 检查该 IP 是否已被其他接口使用

## 最佳实践

### 1. 接口命名规范

```
物理端口: GigabitEthernet0/0/<编号> / 10GE0/0/<编号>
逻辑接口: Vlanif<vlan-id> / LoopBack<编号> / Eth-Trunk<编号>
```

### 2. VLAN 规划原则

- Access 端口：只属于一个 VLAN，配置 `accessVlan`
- Trunk 端口：配置 `trunkAllowedVlans`，建议明确列出
- Native VLAN：使用 VLAN 1，不承载业务数据

### 3. 性能优化配置

```yaml
# 服务器接入端口
speed: 10000
duplex: full
mtu: 9216          # 巨帧，存储/备份场景使用
stormControlEnabled: true
```

### 4. 使用 Label 进行分组管理

```yaml
metadata:
  labels:
    port-type: server-access
    location: rack-a1
    device: switch-demo-01
    environment: production
```

批量操作示例：
```bash
# 查看所有服务器接入端口
kubectl get bi -l port-type=server-access

# 批量下线测试环境端口
kubectl get bi -l environment=test -o name | xargs kubectl delete
```

### 5. 接口状态监控

定期检查接口错误计数器：
```bash
kubectl get bi -o json | jq '.items[] | select(.status.counters.inErrors > 0) | .metadata.name'
```

当 `inErrors` / `outErrors` 持续增长时，及时排查物理链路问题。
