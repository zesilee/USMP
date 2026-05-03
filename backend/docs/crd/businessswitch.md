# BusinessSwitch CRD 使用文档

## 概述

BusinessSwitch CRD 用于管理交换机设备的注册、认证信息、探活状态等元数据。

**API Group**: `biz.usmp.io/v1`

**资源名称**: `businessswitches`

**简称**: `bs`, `switches`

## Spec 字段说明

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|-----|--------|------|
| `deviceIP` | string | ✅ | - | 设备 IP 地址（IPv4） |
| `vendor` | string | - | Huawei | 厂商类型：`Huawei` / `Cisco` / `H3C` / `Juniper` / `Unknown` |
| `model` | string | - | - | 设备型号，如 CE6857 |
| `port` | int | - | 830 | NETCONF 端口号 |
| `credentials` | Credentials | - | - | 认证信息 |
| `enabled` | bool | - | true | 是否启用自动同步 |
| `syncInterval` | int | - | 5 | 同步间隔（分钟） |
| `description` | string | - | - | 设备描述 |
| `location` | string | - | - | 设备位置，如机房、机架号 |
| `owner` | string | - | - | 负责人 |

### Credentials 子字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `username` | string | 登录用户名 |
| `password` | string | 登录密码（明文，建议使用 Secret） |
| `passwordSecretRef` | string | 密码 Secret 引用，格式: `secret-name/key` |

## Status 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `phase` | string | 同步阶段：`Pending` / `Syncing` / `Synced` / `Failed` |
| `onlineStatus` | string | 在线状态：`Online` / `Offline` / `Unknown` |
| `lastSeenTime` | Time | 最后探活时间 |
| `lastSyncTime` | Time | 最后同步时间 |
| `message` | string | 同步消息或错误信息 |
| `retryCount` | int | 重试次数 |
| `errorType` | string | 错误类型：`Temporary` / `Permanent` |
| `hardware` | HardwareStatus | 硬件状态 |
| `vlanCount` | int | VLAN 数量 |
| `interfaceCount` | int | 接口总数 |
| `onlineInterfaceCount` | int | 在线接口数 |

### HardwareStatus 子字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `serialNumber` | string | 序列号 |
| `hardwareVersion` | string | 硬件版本 |
| `softwareVersion` | string | 软件版本 |
| `uptime` | string | 运行时间 |
| `cpuUsage` | int | CPU 使用率（%） |
| `memoryUsage` | int | 内存使用率（%） |
| `temperature` | int | 温度（℃） |

## 示例

### 基础示例

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessSwitch
metadata:
  name: switch-demo-01
spec:
  deviceIP: 192.168.1.100
  vendor: Huawei
  model: CE6857
  port: 830
  credentials:
    username: admin
    password: Admin@123
  enabled: true
  syncInterval: 5
  description: 数据中心核心交换机 01
  location: 北京机房-A区-3机柜
  owner: network-team
```

### 使用 Secret 存储密码（推荐）

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: switch-credentials
type: Opaque
data:
  admin-password: QWRtaW5AMTIz  # base64 编码
---
apiVersion: biz.usmp.io/v1
kind: BusinessSwitch
metadata:
  name: switch-prod-01
spec:
  deviceIP: 10.0.0.1
  vendor: Huawei
  credentials:
    username: admin
    passwordSecretRef: switch-credentials/admin-password
  enabled: true
  syncInterval: 10
```

### 禁用自动同步

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessSwitch
metadata:
  name: switch-backup-01
spec:
  deviceIP: 192.168.1.200
  vendor: Huawei
  enabled: false  # 禁用，仅注册不探活
  description: 备用交换机
```

## kubectl 常用操作

### 查看所有交换机

```bash
kubectl get businessswitches
kubectl get bs  # 简写
```

输出示例：
```
NAME             IP             VENDOR   STATUS   PHASE    AGE
switch-demo-01   192.168.1.100  Huawei   Online   Synced   1h
switch-demo-02   192.168.1.101  Huawei   Offline  Failed   30m
```

### 查看详细状态

```bash
kubectl describe bs switch-demo-01
```

### 查看 CRD 定义

```bash
kubectl get crd businessswitches.biz.usmp.io
kubectl explain businessswitches.spec
```

### 删除交换机

```bash
kubectl delete bs switch-demo-01
```

## 控制器行为

### 探活机制

- 每隔 `syncInterval` 分钟，Controller 会通过 NETCONF 建立连接探测设备在线状态
- 探测成功：`onlineStatus = Online`，`lastSeenTime` 更新
- 探测失败：指数退避重试（5s/10s/20s/40s/60s）
- 连续 5 次失败：标记 `Phase = Failed`，并将 `onlineStatus = Offline`

### Finalizer 清理

删除 BusinessSwitch CR 时，Controller 会：
1. 释放 Client Pool 中的设备连接
2. 记录设备下线日志
3. 移除 Finalizer 后完成删除

## 排错指南

### 设备显示 Offline

1. 检查 `deviceIP` 和 `port` 是否正确
2. 检查网络连通性：`kubectl exec <controller-pod> -- ping <device-ip>`
3. 检查认证信息（username/password）
4. 查看 Controller 日志：`kubectl logs -f <controller-pod>`
5. 查看 CR Status 中的错误信息：
   ```bash
   kubectl get bs switch-demo-01 -o jsonpath='{.status.message}'
   ```

### 设备配置未同步

1. 检查 `spec.enabled` 是否为 `true`
2. 检查 `spec.syncInterval` 是否合理
3. 查看 Controller 日志
4. 查看 CR Status：
   ```bash
   kubectl describe bs switch-demo-01
   ```

## 与其他 CRD 的关联

```
BusinessSwitch
  └── deviceID (作为关联键)
       │
       ├── BusinessVlan.spec.deviceID → 这个 VLAN 属于哪个交换机
       ├── BusinessInterface.spec.deviceID → 这个接口属于哪个交换机
       ├── BusinessRoute.spec.deviceID → 这个路由属于哪个交换机
       └── NativeDeviceConfig.spec.deviceID → 这个原生配置属于哪个交换机
```

## 最佳实践

1. **使用 Secret 存储密码**，不要将明文密码写在 CR 中
2. **合理设置 syncInterval**：核心设备 5 分钟，边缘设备 15-30 分钟
3. **添加 label 进行分组管理**：
   ```yaml
   metadata:
     labels:
       environment: prod
       region: beijing
       rack: A3
   ```
4. **使用 owner 字段标记负责人**，便于故障通知
5. **定期备份 CR**：`kubectl get bs -o yaml > switches-backup.yaml`
