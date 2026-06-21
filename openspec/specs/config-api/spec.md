# config-api - 行为契约

## 接口定义

### GET /api/v1/config/:ip/*path

**描述**：从设备读取指定YANG路径的运行配置

**参数**：
| 参数 | 位置 | 类型 | 必填 | 约束 | 示例 |
|------|------|------|------|------|------|
| ip | path | string | 是 | 设备IP | "192.168.1.1" |
| path | path | string | 是 | YANG节点路径（含前导/） | "/ifm:ifm/interfaces" |
| force_refresh | query | string | 否 | "true"强制刷新（**未实现**） | "true" |

**响应**：
| 状态码 | 含义 | Body |
|--------|------|------|
| 200(成功) | 配置读取成功 | `{"code":0,"data":{"data":{...}},"success":true}` |
| 200(错误) | 获取客户端失败 | `{"code":500,"message":"Failed to get device client: ...","success":false}` |
| 200(错误) | 设备未连接 | `{"code":503,"message":"Device is not connected","success":false}` |
| 200(错误) | NETCONF操作失败 | `{"code":500,"message":"Failed to get configuration: ...","success":false}` |

### POST /api/v1/config/:ip/*path

**描述**：声明式配置下发——存储期望配置并触发异步Reconciliation

**参数**：
| 参数 | 位置 | 类型 | 必填 | 约束 | 示例 |
|------|------|------|------|------|------|
| ip | path | string | 是 | 设备IP | "192.168.1.1" |
| path | path | string | 是 | YANG节点路径 | "/vlan:vlans" |
| body | body | object | 是 | JSON配置数据 | `{"vlans":[...]}` |

**响应**：
| 状态码 | 含义 | Body |
|--------|------|------|
| 200(成功) | 配置已接受 | `{"code":0,"data":{"status":"ACCEPTED","path":"/vlan:vlans","reconciliation":{"triggered":bool,"message":"..."}},"success":true}` |
| 200(错误) | 请求格式错误 | `{"code":400,"message":"Invalid request: ...","success":false}` |
| 200(错误) | YANG类型转换失败 | `{"code":400,"message":"Failed to parse configuration: ...","success":false}` |
| 200(错误) | ConfigStore写入失败 | `{"code":500,"message":"Failed to store configuration: ...","success":false}` |

## 业务规则

### BR-01: 配置读取-正常

- Given: 设备在线且NETCONF连接活跃
- When: 调用 GET /api/v1/config/:ip/*path
- Then: 通过NETCONF GetConfig(running)读取配置，10秒超时，返回配置数据

### BR-02: 配置读取-设备离线

- Given: 设备NETCONF连接断开
- When: 调用 GET /api/v1/config/:ip/*path
- Then: pool.Get()失败返回 code=500，或 cli.IsConnected()=false 返回 code=503

### BR-03: 配置读取-超时

- Given: NETCONF GetConfig操作超过10秒
- When: 调用 GET /api/v1/config/:ip/*path
- Then: context超时，返回 code=500

### BR-04: 配置读取-force_refresh未实现

- Given: 请求包含 force_refresh=true 查询参数
- When: 调用 GET /api/v1/config/:ip/*path
- Then: 参数被解析但忽略，行为与无参数相同（代码中有TODO注释）

### BR-05: 配置下发-声明式

- Given: 有效的YANG路径和JSON配置数据
- When: 调用 POST /api/v1/config/:ip/*path
- Then: JSON数据转换为强类型YANG结构体 → 存入ConfigStore → 触发Reconcile → 返回ACCEPTED

### BR-06: 配置下发-类型转换路由

- Given: 请求path包含特定YANG模块关键字
- When: 调用 POST /api/v1/config/:ip/*path
- Then: 按路径关键字路由到对应转换函数：
  - 含"system:" → convertMapToHuaweiSystem
  - 含"ifm:ifm"+"interfaces" → convertMapToHuaweiIfm
  - 含"vlan:"+"vlan/vlans" → convertMapToHuaweiVlan
  - 其他 → 原始map回退

### BR-07: 配置下发-Reconcile触发

- Given: 配置已成功存入ConfigStore
- When: manager.TriggerReconcile(ip, path) 被调用
- Then: 返回值 triggered 表示是否找到对应Controller；无论结果如何配置已存储

### BR-08: 配置下发-无效JSON

- Given: 请求body不是合法JSON
- When: 调用 POST /api/v1/config/:ip/*path
- Then: ShouldBindJSON失败，返回 code=400 "Invalid request"

## 数据模型

### ConfigGetResponse

| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| data | object | NETCONF返回的配置数据 | {"interfaces":{...}} |

**示例**：
```json
{
  "data": {
    "interfaces": {
      "interface": [
        {"name": "GE0/0/1", "adminStatus": 1}
      ]
    }
  }
}
```

### ConfigSetRequest

| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| (root) | object | 任意JSON配置数据 | {"vlans": [...]} |

**示例（VLAN）**：
```json
{
  "vlans": [
    {"id": 100, "name": "mgmt", "description": "管理VLAN"}
  ]
}
```

### ConfigSetResponse

| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| status | string | 固定"ACCEPTED" | "ACCEPTED" |
| path | string | 请求的YANG路径 | "/vlan:vlans" |
| reconciliation.triggered | bool | 是否触发到Controller | true |
| reconciliation.message | string | 说明信息 | "Configuration stored..." |

**示例**：
```json
{
  "status": "ACCEPTED",
  "path": "/vlan:vlans",
  "reconciliation": {
    "triggered": true,
    "message": "Configuration stored. Reconciliation will sync device state."
  }
}
```