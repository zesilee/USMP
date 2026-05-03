# NativeDeviceConfig CRD 使用文档

## 概述

NativeDeviceConfig CRD 是 USMP 平台的"逃生舱"机制，用于直接透传原生配置（CLI/YANG/XML/JSON）到设备，不经过翻译引擎。适用于：
- 翻译引擎暂不支持的厂商特有功能
- 调试和排障场景
- 批量脚本执行
- 复杂配置直接下发

**API Group**: `biz.usmp.io/v1`

**资源名称**: `nativedeviceconfigs`

**简称**: `ndc`, `nativeconfigs`

## Spec 字段说明

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|-----|--------|------|
| `deviceID` | string | ✅ | - | 所属交换机设备 ID（对应 BusinessSwitch 的 name） |
| `format` | string | ✅ | - | 配置格式：`CLI` / `YANG` / `XML` / `JSON` |
| `content` | string | ✅ | - | 配置内容 |
| `executionMode` | string | - | Once | 执行模式：`Once` / `Persistent` |
| `encrypted` | bool | - | false | 配置是否加密 |
| `encryptionAlgorithm` | string | - | - | 加密算法（如 AES-256） |
| `keySecretRef` | string | - | - | 密钥 Secret 引用 |
| `description` | string | - | - | 配置描述 |
| `priority` | int | - | 50 | 优先级（控制下发顺序，数字越小越先） |
| `group` | string | - | - | 配置分组（用于批量管理） |
| `timeoutSeconds` | int | - | 60 | 执行超时时间（秒） |
| `maxRetries` | int | - | 3 | 失败重试次数 |
| `saveBeforeApply` | bool | - | false | 是否在下发前保存配置 |
| `saveAfterApply` | bool | - | false | 是否在下发后保存配置 |

## Status 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `phase` | string | 同步阶段：`Pending` / `Syncing` / `Synced` / `Failed` |
| `lastSyncTime` | Time | 最后同步时间 |
| `message` | string | 同步消息或错误信息 |
| `retryCount` | int | 重试次数 |
| `errorType` | string | 错误类型：`Temporary` / `Permanent` |
| `executionStatus` | string | 执行状态：`Pending` / `Running` / `Succeeded` / `Failed` / `Skipped` |
| `configHash` | string | 配置内容的 SHA256 哈希值（用于检测变化） |
| `deviceResponse` | string | 设备返回的响应 |
| `executionStartTime` | Time | 执行开始时间 |
| `executionEndTime` | Time | 执行结束时间 |
| `executionDurationMs` | int64 | 执行耗时（毫秒） |
| `appliedOnDevice` | bool | 配置已在设备上生效 |

## 示例

### CLI 配置（登录 Banner）

```yaml
apiVersion: biz.usmp.io/v1
kind: NativeDeviceConfig
metadata:
  name: cli-banner-config
  labels:
    app: usmp
    environment: demo
spec:
  deviceID: switch-demo-01
  format: CLI
  executionMode: Once
  content: |
    header shell %
    Welcome to Huawei Switch - Managed by USMP
    ==========================================
    Unauthorized access is prohibited
    ==========================================
    %
  description: 设置交换机登录 Banner
  saveAfterApply: true
  timeoutSeconds: 30
```

### YANG 配置（VLAN 配置）

```yaml
apiVersion: biz.usmp.io/v1
kind: NativeDeviceConfig
metadata:
  name: yang-vlan-100-config
  labels:
    app: usmp
    environment: demo
spec:
  deviceID: switch-demo-01
  format: YANG
  executionMode: Persistent
  content: |
    <config>
      <vlan xmlns="urn:huawei:params:xml:ns:yang:huawei-vlan">
        <vlans>
          <vlan>
            <id>100</id>
            <name>Business-VLAN-100</name>
            <description>Business Application VLAN</description>
            <admin-status>up</admin-status>
            <mac-learning>enable</mac-learning>
          </vlan>
        </vlans>
      </vlan>
    </config>
  description: VLAN 100 配置（持续同步模式）
  priority: 10
```

### SNMP 基础配置（CLI）

```yaml
apiVersion: biz.usmp.io/v1
kind: NativeDeviceConfig
metadata:
  name: cli-snmp-config
  labels:
    app: usmp
    environment: prod
spec:
  deviceID: switch-demo-01
  format: CLI
  executionMode: Once
  content: |
    snmp-agent
    snmp-agent community read public@123
    snmp-agent community write private@456
    snmp-agent sys-info version v2c v3
    snmp-agent trap enable
    snmp-agent target-host trap address udp-domain 10.0.0.10 params securityname public v2c
  description: SNMP 监控基础配置
  priority: 10
  timeoutSeconds: 30
  saveAfterApply: true
```

### 批量配置分组（系统优化）

```yaml
apiVersion: biz.usmp.io/v1
kind: NativeDeviceConfig
metadata:
  name: system-optimization-base
  labels:
    group: system-optimization
    priority: critical
spec:
  deviceID: switch-demo-01
  format: CLI
  executionMode: Once
  content: |
    # 关闭不需要的服务
    undo http server enable
    undo telnet server enable
    
    # 启用 SSH
    ssh server enable
    stelnet server enable
    
    # 日志配置
    info-center enable
    info-center loghost 10.0.0.20 facility local7
  description: 系统安全基础配置
  priority: 5
  maxRetries: 5
  saveAfterApply: true

---
apiVersion: biz.usmp.io/v1
kind: NativeDeviceConfig
metadata:
  name: system-optimization-stp
  labels:
    group: system-optimization
spec:
  deviceID: switch-demo-01
  format: CLI
  executionMode: Once
  content: |
    # STP 优化配置
    stp mode mstp
    stp enable
    stp root primary
    stp bpdu-protection
  description: STP 生成树优化配置
  priority: 10
```

### 加密敏感配置

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: device-config-key
type: Opaque
data:
  aes-key: dGVzdC1rZXktMTIzNDU2Nzg5MA==  # base64 编码的加密密钥
---
apiVersion: biz.usmp.io/v1
kind: NativeDeviceConfig
metadata:
  name: encrypted-password-config
spec:
  deviceID: switch-demo-01
  format: CLI
  executionMode: Once
  encrypted: true
  encryptionAlgorithm: AES-256-CBC
  keySecretRef: device-config-key/aes-key
  content: |
    U2FsdGVkX1+...  # 加密后的配置内容
  description: 包含敏感信息的加密配置
  saveAfterApply: true
```

## kubectl 常用操作

### 查看所有原生配置

```bash
kubectl get nativedeviceconfigs
kubectl get ndc  # 简写
```

输出示例：
```
NAME                    DEVICE          FORMAT  PHASE    STATUS   AGE
cli-banner-config       switch-demo-01  CLI     Synced   Succeeded  1h
yang-vlan-100-config    switch-demo-01  YANG    Synced   Succeeded  30m
cli-snmp-config         switch-demo-01  CLI     Synced   Succeeded  15m
```

### 查看特定配置详情

```bash
kubectl describe ndc cli-banner-config
```

### 查看设备返回的响应

```bash
kubectl get ndc cli-snmp-config -o jsonpath='{.status.deviceResponse}'
```

### 查看执行耗时

```bash
kubectl get ndc cli-snmp-config -o jsonpath='{.status.executionDurationMs}'
```

### 按分组查看配置

```bash
kubectl get ndc -l group=system-optimization
```

### 查看所有失败的配置

```bash
kubectl get ndc -o wide | grep Failed
```

### 重新触发配置执行

配置内容不变时，Controller 不会重复下发。如需重新执行，更新 annotation：

```bash
kubectl annotate ndc cli-banner-config usmp.io/retrigger-at=$(date +%s) --overwrite
```

### 批量执行配置组

```bash
# 查看系统优化组的所有配置
kubectl get ndc -l group=system-optimization

# 按优先级顺序重新触发
for pri in 5 10 15 20; do
  kubectl get ndc -l group=system-optimization -o name | while read name; do
    kubectl annotate $name usmp.io/retrigger-at=$(date +%s) --overwrite
  done
  sleep 5
done
```

### 删除配置

```bash
kubectl delete ndc cli-banner-config
```

> **注意**：删除 NativeDeviceConfig CR 时，**不会**自动回滚配置。如需回滚，请手动下发恢复配置。

## 控制器行为

### Reconcile 流程

```
1. CR 创建/更新事件
   ↓
2. 计算配置内容 SHA256 哈希
   ↓
3. 哈希未变化且 executionMode=Once → 跳过，结束
   ↓
4. 按 priority 排序，确定执行顺序
   ↓
5. 更新 Status: ExecutionStatus = Running
   ↓
6. 解密配置（如果 encrypted=true）
   ↓
7. 建立 NETCONF/SSH 连接
   ↓
8. 执行 saveBeforeApply（如果启用）
   ↓
9. 下发配置内容
   ↓
10. 捕获设备响应
    ↓
11. 执行 saveAfterApply（如果启用）
    ↓
12. 记录执行耗时，更新 Status: ExecutionStatus = Succeeded/Failed
    ↓
13. executionMode=Persistent → 5分钟后再次校验
    executionMode=Once → 结束
```

### 执行模式说明

| 模式 | 行为 | 适用场景 |
|------|------|---------|
| **Once** | 仅执行一次，内容不变不重复执行 | 一次性脚本、初始化配置 |
| **Persistent** | 定期校验并重新下发，确保配置始终存在 | 关键配置、需要保持的状态 |

### 优先级排序

多个配置下发到同一设备时，按 `priority` 字段从小到大顺序执行。
相同优先级的配置按创建时间顺序执行。

## 排错指南

### 配置执行失败

1. 查看错误信息：
   ```bash
   kubectl get ndc cli-snmp-config -o jsonpath='{.status.message}'
   ```

2. 查看设备响应：
   ```bash
   kubectl get ndc cli-snmp-config -o jsonpath='{.status.deviceResponse}'
   ```

3. 检查设备连接状态：
   ```bash
   kubectl get bs switch-demo-01
   ```

### CLI 命令执行报错

1. **检查命令格式**：华为 CLI 命令需要在正确的视图下执行
   ```
   系统视图命令: 直接执行
   接口视图命令: 需要先进入接口视图
   VLAN 视图命令: 需要先进入 VLAN 视图
   ```

2. **检查命令前缀**：某些命令需要 `undo` 前缀来取消配置

3. **查看完整错误输出**：
   ```bash
   kubectl get ndc failed-config -o yaml | grep -A 20 deviceResponse
   ```

### YANG 配置下发失败

1. 检查 XML 格式是否正确，标签是否闭合
2. 检查命名空间（xmlns）是否正确
3. 确认 YANG 模型版本与设备版本匹配
4. 使用 `yanglint` 工具验证 YANG 配置

### 配置未重复执行

1. 检查执行模式：
   ```bash
   kubectl get ndc my-config -o jsonpath='{.spec.executionMode}'
   ```

2. 如果是 Once 模式，需要手动触发重新执行：
   ```bash
   kubectl annotate ndc my-config usmp.io/retrigger-at=$(date +%s) --overwrite
   ```

### 加密配置解密失败

1. 确认 Secret 存在且格式正确：
   ```bash
   kubectl get secret device-config-key
   ```

2. 检查 keySecretRef 格式是否正确（`secret-name/key-name`）

3. 确认加密算法与密钥匹配

## 最佳实践

### 1. 优先使用业务 CRD

NativeDeviceConfig 是逃生舱机制，应优先使用：
- BusinessVlan → VLAN 配置
- BusinessInterface → 接口配置
- BusinessRoute → 路由配置

仅在业务 CRD 不支持时使用 NativeDeviceConfig。

### 2. 配置分组管理

使用 Label 进行分组：
```yaml
metadata:
  labels:
    group: security-hardening
    priority: high
    env: production
```

批量操作：
```bash
# 重新执行所有安全加固配置
kubectl get ndc -l group=security-hardening -o name | while read name; do
  kubectl annotate $name usmp.io/retrigger-at=$(date +%s) --overwrite
done
```

### 3. 优先级设置原则

```
priority: 1-10   → 基础系统配置（AAA、SNMP、日志）
priority: 11-30  → 网络协议配置（STP、LACP、BFD）
priority: 31-50  → 业务配置（VLAN、接口、路由）
priority: 51-80  → 优化和调优配置
priority: 81+    → 验证和监控配置
```

### 4. 敏感信息处理

- 密码、密钥等敏感信息使用加密配置
- 使用 Secret 存储加密密钥
- 避免在 Git 中提交明文密码

### 5. 配置保存策略

- 关键配置：`saveAfterApply: true`
- 临时调试配置：不保存
- 批量配置：最后一个配置设置 saveAfterApply

### 6. 调试技巧

执行前先在单台设备测试：
```bash
# 创建测试配置
kubectl apply -f test-config.yaml

# 查看执行结果
watch kubectl get ndc test-config -o wide

# 如失败，查看详细响应
kubectl get ndc test-config -o jsonpath='{.status.deviceResponse}' | jq
```

### 7. 与业务 CRD 配合使用

场景：需要配置业务 CRD 不支持的高级功能
```yaml
# 1. 先使用业务 CRD 配置基础 VLAN
# BusinessVlan CR 创建 VLAN 100

# 2. 再使用 NativeDeviceConfig 配置 VLAN 高级功能
apiVersion: biz.usmp.io/v1
kind: NativeDeviceConfig
metadata:
  name: vlan-100-advanced
spec:
  deviceID: switch-demo-01
  format: CLI
  executionMode: Persistent
  content: |
    vlan 100
      traffic-statistic enable
      broadcast-suppression 80
  priority: 30
  description: VLAN 100 流量统计和广播抑制
```

## 注意事项

⚠️ **配置不会自动回滚**：删除 CR 不会移除已下发的配置，如需回滚请手动下发恢复配置

⚠️ **幂等性保证**：CLI 命令需要保证幂等，重复执行不产生副作用

⚠️ **执行顺序**：同一设备多个配置按 priority 顺序执行，相同 priority 按创建时间

⚠️ **超时设置**：复杂配置适当增加 timeoutSeconds，避免执行中断

⚠️ **Persistent 模式**：会定期重复执行，确保配置不被手动修改覆盖
