# BusinessVlan CRD 使用文档

## 概述

BusinessVlan CRD 用于管理交换机上的 VLAN 业务配置，包括 VLAN 基本属性、端口成员、MAC 学习等功能。

**API Group**: `biz.usmp.io/v1`

**资源名称**: `businessvlans`

**简称**: `bv`, `vlans`

## Spec 字段说明

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|-----|--------|------|
| `vlanID` | int | ✅ | - | VLAN ID，范围 1-4094 |
| `deviceID` | string | ✅ | - | 所属交换机设备 ID（对应 BusinessSwitch 的 name） |
| `name` | string | - | - | VLAN 名称，最大 31 字符 |
| `description` | string | - | - | VLAN 描述，最大 255 字符 |
| `type` | string | - | Common | VLAN 类型：`Common` / `Super` / `Sub` |
| `adminStatus` | string | - | Up | 管理状态：`Up` / `Down` |
| `taggedPorts` | []string | - | [] | Tagged 端口名称列表 |
| `untaggedPorts` | []string | - | [] | Untagged 端口名称列表 |
| `macLearningEnabled` | bool | - | true | MAC 地址学习开关 |
| `statisticEnabled` | bool | - | false | 统计功能开关 |
| `broadcastDiscardEnabled` | bool | - | false | 广播丢弃开关 |

## Status 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `phase` | string | 同步阶段：`Pending` / `Syncing` / `Synced` / `Failed` |
| `lastSyncTime` | Time | 最后同步时间 |
| `message` | string | 同步消息或错误信息 |
| `configVersion` | int64 | 配置版本号（乐观锁） |
| `retryCount` | int | 重试次数 |
| `errorType` | string | 错误类型：`Temporary` / `Permanent` |
| `actual` | VlanStatus | 设备上实际的 VLAN 状态 |
| `diff` | []string | 期望与实际的差异列表 |

### VlanStatus 子字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `vlanID` | int | 实际 VLAN ID |
| `name` | string | 实际 VLAN 名称 |
| `description` | string | 实际描述 |
| `type` | string | 实际类型 |
| `operStatus` | string | 运行状态 |
| `ports` | []PortStatus | 端口状态列表 |
| `macCount` | int | MAC 地址数 |

## 示例

### 基础 VLAN 配置

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessVlan
metadata:
  name: vlan-10
spec:
  vlanID: 10
  deviceID: switch-demo-01
  name: Business-VLAN-10
  description: 业务部门 VLAN
  adminStatus: Up
  macLearningEnabled: true
  statisticEnabled: true
```

### 带端口成员的 VLAN

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessVlan
metadata:
  name: vlan-100
spec:
  vlanID: 100
  deviceID: switch-demo-01
  name: Server-VLAN
  description: 服务器接入 VLAN
  type: Common
  adminStatus: Up
  taggedPorts:
    - GigabitEthernet0/0/1
    - GigabitEthernet0/0/2
    - GigabitEthernet0/0/3
  untaggedPorts:
    - GigabitEthernet0/0/10
    - GigabitEthernet0/0/11
  macLearningEnabled: true
  statisticEnabled: true
  broadcastDiscardEnabled: false
```

### Super VLAN + Sub VLAN

```yaml
# Super VLAN
apiVersion: biz.usmp.io/v1
kind: BusinessVlan
metadata:
  name: vlan-super-1000
spec:
  vlanID: 1000
  deviceID: switch-demo-01
  name: Super-VLAN-1000
  type: Super
  adminStatus: Up
---
# Sub VLAN 1
apiVersion: biz.usmp.io/v1
kind: BusinessVlan
metadata:
  name: vlan-sub-1001
spec:
  vlanID: 1001
  deviceID: switch-demo-01
  name: Sub-VLAN-1001
  type: Sub
  adminStatus: Up
```

### 禁用的 VLAN

```yaml
apiVersion: biz.usmp.io/v1
kind: BusinessVlan
metadata:
  name: vlan-legacy-999
spec:
  vlanID: 999
  deviceID: switch-demo-01
  name: Legacy-VLAN
  description: 已废弃，仅保留配置
  adminStatus: Down  # 管理性关闭
  macLearningEnabled: false
```

## kubectl 常用操作

### 查看所有 VLAN

```bash
kubectl get businessvlans
kubectl get bv  # 简写
```

输出示例：
```
NAME           VLANID   DEVICE          PHASE    LASTSYNC          AGE
vlan-10        10       switch-demo-01  Synced   2024-01-15T10:30  1h
vlan-100       100      switch-demo-01  Synced   2024-01-15T10:30  30m
vlan-legacy-999 999     switch-demo-01  Failed   2024-01-15T10:25  5m
```

### 查看特定 VLAN 详情

```bash
kubectl describe bv vlan-100
```

### 查看期望与实际的差异

```bash
kubectl get bv vlan-100 -o jsonpath='{.status.diff}'
```

### 批量查看所有失败的 VLAN

```bash
kubectl get bv -o wide | grep Failed
```

### 删除 VLAN

```bash
kubectl delete bv vlan-legacy-999
```

> **注意**：删除 BusinessVlan CR 时，Controller 会自动从设备上删除对应的 VLAN 配置。

## 控制器行为

### Reconcile 流程

```
1. CR 创建/更新事件
   ↓
2. Finalizer 检查（删除时清理设备配置）
   ↓
3. 翻译器 Validate() 验证配置
   ↓
4. 翻译器 TranslateVlan() 转为华为 YANG
   ↓
5. NETCONF edit-config 下发
   ↓
6. 从设备读取实际配置
   ↓
7. 计算 Diff（期望 vs 实际）
   ↓
8. 更新 Status: Phase=Synced + Actual状态 + Diff
   ↓
9. 5分钟后再次 Reconcile 校验配置一致性
```

### Finalizer 清理

删除 CR 时，Controller 会：
1. 向设备发送删除 VLAN 的 edit-config
2. 等待设备确认
3. 记录日志
4. 移除 Finalizer，完成删除

## 排错指南

### VLAN 配置下发失败

1. 查看 Status 错误信息：
   ```bash
   kubectl get bv vlan-100 -o jsonpath='{.status.message}'
   ```

2. 查看错误类型：
   ```bash
   kubectl get bv vlan-100 -o jsonpath='{.status.errorType}'
   ```

3. 查看 Controller 日志：
   ```bash
   kubectl logs -f <controller-pod> | grep vlan-100
   ```

4. 检查 deviceID 对应的 BusinessSwitch 是否在线：
   ```bash
   kubectl get bs switch-demo-01
   ```

### 端口成员未更新

1. 检查端口名称格式是否正确（华为使用 `GigabitEthernet0/0/1` 格式）
2. 检查端口是否存在于交换机上
3. 检查端口是否已被其他 VLAN 使用

### VLAN 在设备上存在但 CR 显示 Failed

1. 手动触发重同步：
   ```bash
   kubectl annotate bv vlan-100 kubectl.kubernetes.io/restartedAt=$(date +%Y%m%d-%H%M%S) --overwrite
   ```

2. 或直接删除 CR 重建

## 翻译引擎行为

### 华为 VLAN 转换映射

| BusinessVlan Spec | Huawei YANG 字段 | 说明 |
|-------------------|-----------------|------|
| `vlanID` | `/vlans/vlan/id` | VLAN ID |
| `name` | `/vlans/vlan/name` | VLAN 名称 |
| `description` | `/vlans/vlan/description` | 描述 |
| `adminStatus` | `/vlans/vlan/admin-status` | Up/Down → 1/2 |
| `macLearningEnabled` | `/vlans/vlan/mac-learning` | true/false → 1/2 |
| `statisticEnabled` | `/vlans/vlan/statistic-enable` | true/false → 1/2 |
| `broadcastDiscardEnabled` | `/vlans/vlan/broadcast-discard` | true/false → 1/2 |

## 最佳实践

1. **VLAN ID 规划**：
   - 1-100: 用户 VLAN
   - 101-200: 服务器 VLAN
   - 201-500: 特殊用途（存储、监控等）
   - 501-4094: 预留扩展

2. **命名规范**：
   - 统一前缀：`vlan-<ID>` 作为 CR 名称
   - VLAN 名称清晰表达用途
   - 包含环境标识（prod/staging/dev）

3. **批量管理**：
   ```bash
   # 导出所有 VLAN 配置
   kubectl get bv -o yaml > all-vlans.yaml

   # 批量删除测试环境 VLAN
   kubectl get bv -l environment=test -o name | xargs kubectl delete
   ```

4. **重要 VLAN 配置变更前备份**：
   ```bash
   kubectl get bv vlan-core -o yaml > vlan-core-backup-$(date +%Y%m%d).yaml
   ```

5. **使用 label 进行分类**：
   ```yaml
   metadata:
     labels:
       environment: prod
       vlan-type: user
       department: finance
   ```
