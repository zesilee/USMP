# devices-api - 行为契约

## 接口定义

### GET /api/v1/devices

**描述**：返回所有已注册设备列表及连接池统计信息

**参数**：无

**响应**：
| 状态码 | 含义 | Body |
|--------|------|------|
| 200 | 成功 | `{"code":0,"message":"...","data":{"devices":[...],"stats":{...}},"success":true}` |

**数据模型 — DeviceInfo**：
| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| ip | string | 必填，设备唯一标识 | "192.168.1.1" |
| port | int | 可选，默认830 | 830 |
| username | string | 必填 | "admin" |
| password | string | 必填，**明文返回** | "admin" |

**数据模型 — ClientPoolStats**：
| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| active_connections | int | 当前活跃连接数 | 3 |
| total_connections | int | 历史总连接数 | 10 |
| errors | int | 连接错误累计数 | 1 |

### POST /api/v1/devices

**描述**：注册新设备，默认端口830，注册后立即尝试NETCONF连接

**参数**：
| 参数 | 位置 | 类型 | 必填 | 约束 | 示例 |
|------|------|------|------|------|------|
| ip | body | string | 是 | 设备IP | "192.168.1.1" |
| port | body | int | 否 | 默认830 | 830 |
| username | body | string | 是 | NETCONF用户名 | "admin" |
| password | body | string | 是 | NETCONF密码 | "admin" |

**响应**：
| 状态码 | 含义 | Body |
|--------|------|------|
| 200(成功) | 设备添加成功 | `{"code":0,"message":"Device added successfully","data":null,"success":true}` |
| 200(错误) | 请求格式错误 | `{"code":400,"message":"Invalid request: ...","success":false}` |
| 200(错误) | 设备连接失败 | `{"code":500,"message":"Failed to connect to device: ...","success":false}` |

### DELETE /api/v1/devices/:ip

**描述**：删除设备注册信息并释放连接池资源

**参数**：
| 参数 | 位置 | 类型 | 必填 | 约束 | 示例 |
|------|------|------|------|------|------|
| ip | path | string | 是 | 设备IP | "192.168.1.1" |

**响应**：
| 状态码 | 含义 | Body |
|--------|------|------|
| 200(成功) | 设备删除成功 | `{"code":0,"message":"Device removed successfully","data":null,"success":true}` |

### GET /api/v1/devices/:ip/status

**描述**：查询设备NETCONF连接状态

**参数**：
| 参数 | 位置 | 类型 | 必填 | 约束 | 示例 |
|------|------|------|------|------|------|
| ip | path | string | 是 | 设备IP | "192.168.1.1" |

**响应**：
| 状态码 | 含义 | Body |
|--------|------|------|
| 200(成功) | 状态查询成功 | `{"code":0,"data":{"running":true,"connected":true/false},"success":true}` |
| 200(错误) | 设备不存在 | `{"code":404,"message":"Device not found","success":false}` |

## 业务规则

### BR-01: 设备列表返回

- Given: 系统中存在N个已注册设备
- When: 调用 GET /api/v1/devices
- Then: 返回所有设备信息及连接池统计，设备信息包含明文密码

### BR-02: 空设备列表

- Given: 系统中无已注册设备
- When: 调用 GET /api/v1/devices
- Then: 返回空数组 `"devices":[]` 和连接池统计

### BR-03: 设备注册-默认端口

- Given: 请求体中 port 为0或未提供
- When: 调用 POST /api/v1/devices
- Then: port 自动设为830

### BR-04: 设备注册-必填字段校验

- Given: 请求体缺少 ip/username/password 任一字段
- When: 调用 POST /api/v1/devices
- Then: 返回 code=400，message 包含 "Invalid request"

### BR-05: 设备注册-重复IP覆盖

- Given: 系统中已存在IP为X的设备
- When: 调用 POST /api/v1/devices 提交相同IP
- Then: 覆盖原有设备信息（不报错），然后尝试连接

### BR-06: 设备注册-连接失败仍保存

- Given: 设备IP/凭据正确但设备不在线
- When: 调用 POST /api/v1/devices
- Then: 设备信息已保存到内存，但返回 code=500 连接失败错误

### BR-07: 设备删除

- Given: 系统中存在IP为X的设备
- When: 调用 DELETE /api/v1/devices/:ip
- Then: 删除设备信息，调用 pool.Release(ip) 释放连接

### BR-08: 设备删除-不存在

- Given: 系统中不存在IP为X的设备
- When: 调用 DELETE /api/v1/devices/:ip
- Then: 仍返回成功（delete on map对不存在的key无操作）

### BR-09: 设备状态-在线

- Given: 设备IP存在且NETCONF连接活跃
- When: 调用 GET /api/v1/devices/:ip/status
- Then: 返回 `{"running":true,"connected":true}`

### BR-10: 设备状态-离线

- Given: 设备IP存在但NETCONF连接断开
- When: 调用 GET /api/v1/devices/:ip/status
- Then: 返回 `{"running":true,"connected":false}`

### BR-11: 设备状态-不存在

- Given: 设备IP未注册
- When: 调用 GET /api/v1/devices/:ip/status
- Then: 返回 code=404 "Device not found"

### BR-12: 统一响应格式

- Given: 任何API请求
- When: 返回响应
- Then: HTTP状态码始终为200，业务成功/失败通过JSON body中的code和success字段区分

## 数据模型

### DeviceInfo

| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| ip | string | 必填，唯一标识 | "192.168.1.1" |
| port | int | ≥0，默认830 | 830 |
| username | string | 必填 | "admin" |
| password | string | 必填，明文存储和返回 | "admin" |

**示例**：
```json
{
  "ip": "192.168.1.1",
  "port": 830,
  "username": "admin",
  "password": "admin"
}
```

### DeviceStatus

| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| running | bool | 始终为true（API服务运行中） | true |
| connected | bool | NETCONF连接是否活跃 | true |

**示例**：
```json
{
  "running": true,
  "connected": true
}
```

### ClientPoolStats

| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| active_connections | int | ≥0 | 3 |
| total_connections | int | ≥0 | 10 |
| errors | int | ≥0 | 1 |

**示例**：
```json
{
  "active_connections": 3,
  "total_connections": 10,
  "errors": 1
}
```