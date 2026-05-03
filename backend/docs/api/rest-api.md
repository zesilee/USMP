# USMP REST API 文档

## 概述

USMP (Unified Switch Management Platform) 提供 REST API 用于管理交换机设备的配置。API 采用**声明式配置模型**，用户只需提交期望的配置状态，平台自动协调设备实际状态与期望状态一致。

### Base URL

```
http://<server-addr>:<port>/api/v1
```

### 通用响应格式

所有 API 响应采用统一的 JSON 格式：

```json
{
  "success": true,
  "message": "操作成功",
  "data": {}
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `success` | boolean | 请求是否成功 |
| `message` | string | 结果描述信息 |
| `data` | object/array | 响应数据 |

### 错误响应示例

```json
{
  "success": false,
  "message": "Device not found",
  "data": null
}
```

HTTP 状态码：
- `200 OK` - 请求成功
- `400 Bad Request` - 请求参数错误
- `404 Not Found` - 资源不存在
- `500 Internal Server Error` - 服务器内部错误
- `503 Service Unavailable` - 设备连接不可用

---

## 设备管理 API

### 1. 获取设备列表

获取所有已注册的交换机设备列表和连接池统计信息。

```http
GET /api/v1/devices
```

**响应示例：**

```json
{
  "success": true,
  "message": "Devices retrieved successfully",
  "data": {
    "devices": [
      {
        "ip": "192.168.1.1",
        "port": 830,
        "username": "admin",
        "password": "admin"
      }
    ],
    "stats": {
      "active_connections": 1,
      "total_connections": 1,
      "errors": 0
    }
  }
}
```

---

### 2. 添加设备

注册新的交换机设备并建立连接。

```http
POST /api/v1/devices
Content-Type: application/json
```

**请求体：**

| 字段 | 类型 | 必填 | 默认值 | 说明 |
|------|------|-----|--------|------|
| `ip` | string | ✅ | - | 设备 IP 地址 |
| `port` | int | - | 830 | NETCONF 端口号 |
| `username` | string | ✅ | - | 登录用户名 |
| `password` | string | ✅ | - | 登录密码 |

**请求示例：**

```json
{
  "ip": "192.168.1.2",
  "port": 830,
  "username": "admin",
  "password": "Admin@123"
}
```

**响应示例：**

```json
{
  "success": true,
  "message": "Device added successfully",
  "data": null
}
```

**注意：** 添加设备时会立即尝试建立 NETCONF 连接。如果连接失败，设备信息仍会保存，但会返回错误消息。

---

### 3. 删除设备

移除设备并关闭连接。

```http
DELETE /api/v1/devices/:ip
```

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `ip` | string | 设备 IP 地址 |

**响应示例：**

```json
{
  "success": true,
  "message": "Device removed successfully",
  "data": null
}
```

---

### 4. 获取设备状态

获取设备连接状态。

```http
GET /api/v1/devices/:ip/status
```

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `ip` | string | 设备 IP 地址 |

**响应示例：**

```json
{
  "success": true,
  "message": "Status retrieved",
  "data": {
    "running": true,
    "connected": true
  }
}
```

---

## 配置管理 API

### 1. 获取配置

从设备获取指定 YANG 路径的配置数据。

```http
GET /api/v1/config/:ip/:path
```

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `ip` | string | 设备 IP 地址 |
| `path` | string | YANG 模型路径（如 `/ifm:ifm/interfaces`） |

**查询参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `force_refresh` | boolean | 是否强制刷新缓存（默认 false） |

**响应示例 - 获取接口配置：**

```json
{
  "success": true,
  "message": "Configuration retrieved",
  "data": {
    "data": {
      "interfaces": {
        "interface": {
          "GigabitEthernet0/0/1": {
            "name": "GigabitEthernet0/0/1",
            "description": "Uplink port",
            "adminStatus": 1,
            "mtu": 1500
          }
        }
      }
    }
  }
}
```

---

### 2. 设置配置（声明式）

提交期望的配置状态，平台自动协调设备状态与期望一致。这是 USMP 的核心声明式 API。

```http
POST /api/v1/config/:ip/:path
Content-Type: application/json
```

**路径参数：**

| 参数 | 类型 | 说明 |
|------|------|------|
| `ip` | string | 设备 IP 地址 |
| `path` | string | YANG 模型路径 |

**请求体：**

YANG 模型对应的 JSON 配置数据，结构需与目标 YANG 模型一致。

#### 示例 1: 设置 VLAN 配置

```http
POST /api/v1/config/192.168.1.1/huawei-vlan:vlan/vlans
```

```json
{
  "vlans": [
    {
      "id": 100,
      "name": "Business-VLAN-100",
      "description": "Business department network",
      "adminStatus": 1,
      "macLearning": 1,
      "statisticEnable": 1
    },
    {
      "id": 200,
      "name": "Server-VLAN",
      "description": "Server farm network",
      "adminStatus": 1
    }
  ]
}
```

#### 示例 2: 设置接口配置

```http
POST /api/v1/config/192.168.1.1/huawei-ifm:ifm/interfaces
```

```json
{
  "interface": {
    "GigabitEthernet0/0/10": {
      "name": "GigabitEthernet0/0/10",
      "description": "Server port",
      "adminStatus": 1,
      "mtu": 9216,
      "l2ModeEnable": true
    }
  }
}
```

#### 示例 3: 设置系统信息

```http
POST /api/v1/config/192.168.1.1/huawei-system:system
```

```json
{
  "system-info": {
    "sysName": "USMP-Core-Switch",
    "sysContact": "network-admin@company.com",
    "sysLocation": "Beijing-DC-Rack-A1"
  }
}
```

**响应示例：**

```json
{
  "success": true,
  "message": "Configuration accepted - reconciliation in progress",
  "data": {
    "status": "ACCEPTED",
    "path": "/huawei-vlan:vlan/vlans",
    "reconciliation": {
      "triggered": true,
      "message": "Configuration stored. Reconciliation will sync device state."
    }
  }
}
```

**工作流程：**

1. API 接收期望配置并进行类型验证
2. 配置存储到 ConfigStore（期望状态源）
3. 触发 Controller 异步协调
4. Controller 执行：
   - 从设备读取实际配置
   - 计算期望与实际的差异
   - 应用变更到设备
   - 提交配置（如果协议支持）

**注意：** 这是一个异步 API，返回 ACCEPTED 表示配置已存储，协调正在进行中。请通过 CRD 的 Status 字段或设备状态 API 查看最终结果。

---

## YANG 模型 API

### 1. 获取支持的 YANG 模块列表

获取平台支持的所有 YANG 模块信息。

```http
GET /api/v1/yang/modules
```

**响应示例：**

```json
{
  "success": true,
  "message": "YANG modules retrieved successfully",
  "data": [
    {
      "name": "Interfaces",
      "path": "/interfaces",
      "description": "Network interfaces configuration",
      "type": "container"
    },
    {
      "name": "VLANs",
      "path": "/vlans",
      "description": "VLAN configuration",
      "type": "container"
    },
    {
      "name": "System",
      "path": "/system",
      "description": "System information and configuration",
      "type": "container"
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 模块名称 |
| `path` | string | YANG 路径 |
| `description` | string | 模块描述 |
| `type` | string | 节点类型（container/list/leaf） |

---

## 枚举值参考

### AdminStatus (接口管理状态)

| 值 | 说明 |
|----|------|
| 1 | Up (启用) |
| 2 | Down (禁用) |

### EnableStatus (功能开关)

| 值 | 说明 |
|----|------|
| 1 | Enable (启用) |
| 2 | Disable (禁用) |

### PortType (接口类型)

| 值 | 类型 |
|----|------|
| 1 | Physical (物理接口) |
| 2 | Virtual (虚拟接口) |
| ... | ... |

---

## 使用示例

### cURL 示例

**添加设备：**

```bash
curl -X POST http://localhost:8080/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{
    "ip": "192.168.1.100",
    "port": 830,
    "username": "admin",
    "password": "Admin@123"
  }'
```

**获取设备列表：**

```bash
curl http://localhost:8080/api/v1/devices
```

**配置 VLAN：**

```bash
curl -X POST http://localhost:8080/api/v1/config/192.168.1.100/huawei-vlan:vlan/vlans \
  -H "Content-Type: application/json" \
  -d '{
    "vlans": [{
      "id": 100,
      "name": "Test-VLAN",
      "adminStatus": 1
    }]
  }'
```

### JavaScript/Node.js 示例

```javascript
const axios = require('axios');

const API_BASE = 'http://localhost:8080/api/v1';

// 添加设备
async function addDevice(ip, username, password) {
  try {
    const response = await axios.post(`${API_BASE}/devices`, {
      ip,
      port: 830,
      username,
      password
    });
    console.log('设备添加成功:', response.data.message);
  } catch (error) {
    console.error('设备添加失败:', error.response?.data?.message || error.message);
  }
}

// 配置 VLAN
async function configureVlan(ip, vlanId, name) {
  try {
    const response = await axios.post(
      `${API_BASE}/config/${ip}/huawei-vlan:vlan/vlans`,
      {
        vlans: [{
          id: vlanId,
          name,
          adminStatus: 1
        }]
      }
    );
    console.log('VLAN 配置已提交:', response.data.data.status);
  } catch (error) {
    console.error('VLAN 配置失败:', error.response?.data?.message || error.message);
  }
}

// 使用示例
addDevice('192.168.1.100', 'admin', 'Admin@123')
  .then(() => configureVlan('192.168.1.100', 100, 'Business-VLAN'));
```

### Python 示例

```python
import requests

API_BASE = 'http://localhost:8080/api/v1'

def add_device(ip, username, password, port=830):
    """添加设备"""
    response = requests.post(
        f'{API_BASE}/devices',
        json={
            'ip': ip,
            'port': port,
            'username': username,
            'password': password
        }
    )
    response.raise_for_status()
    return response.json()

def configure_interface(ip, if_name, description, mtu=1500):
    """配置接口"""
    response = requests.post(
        f'{API_BASE}/config/{ip}/huawei-ifm:ifm/interfaces',
        json={
            'interface': {
                if_name: {
                    'name': if_name,
                    'description': description,
                    'adminStatus': 1,
                    'mtu': mtu
                }
            }
        }
    )
    response.raise_for_status()
    return response.json()

# 使用示例
if __name__ == '__main__':
    # 添加设备
    add_device('192.168.1.100', 'admin', 'Admin@123')
    
    # 配置接口
    result = configure_interface(
        '192.168.1.100',
        'GigabitEthernet0/0/10',
        'Server port',
        9216
    )
    print(f"配置状态: {result['data']['status']}")
```

---

## 常见问题

### Q1: 配置提交后多久生效？

配置提交后，Controller 会立即触发协调。通常情况下：
- 简单配置（如 VLAN 创建）: 1-3 秒
- 复杂配置（多个接口/多条路由）: 3-10 秒
- 大型配置（整台设备）: 10-30 秒

可以通过 Kubernetes `kubectl get <crd>` 命令查看 Status 了解当前进度。

### Q2: 如果设备离线，配置会丢失吗？

不会。期望配置会持久存储在 ConfigStore 中。设备重新上线后，Controller 会自动检测并重新协调配置。

### Q3: 如何取消已提交的配置？

有两种方式：
1. 提交新的配置覆盖旧配置
2. 删除对应的 CRD 资源（会触发清理逻辑）

### Q4: 支持哪些 YANG 模块？

当前支持：
- `huawei-ifm` - 接口管理
- `huawei-vlan` - VLAN 配置
- `huawei-system` - 系统信息

其他厂商的 YANG 模块支持正在开发中。

### Q5: 配置冲突如何处理？

当多个配置同时修改同一资源时：
- 按提交顺序处理
- 后提交的配置会覆盖先提交的配置
- 最终以最后一次提交的期望状态为准

建议使用 Kubernetes CRD 的资源版本机制避免并发冲突。

---

## API 版本历史

| 版本 | 日期 | 变更内容 |
|------|------|---------|
| v1.0 | 2024-01-15 | 初始版本，支持设备管理和基础配置 |
