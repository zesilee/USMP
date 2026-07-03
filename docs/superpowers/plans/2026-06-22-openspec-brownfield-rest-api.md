# OpenSpec Brownfield 反向补齐 - REST API 层实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 对 USMP 项目已有 REST API 代码做事实还原，补齐 OpenSpec 规范文档（spec.md/design.md/tasks.md）

**Architecture:** 按业务域分 3 个 OpenSpec change（devices-api → config-api → yang-api），每个 change 输出三件套。SSE 端点经代码确认不存在于 REST API 层，从本次范围排除。

**Tech Stack:** Markdown 文档，内容从 Go 代码中提取，使用 Given-When-Then 格式

---

## File Structure

```
openspec/specs/
├── devices-api/
│   ├── spec.md          # 设备管理接口行为契约
│   ├── design.md        # 设备管理架构设计
│   └── tasks.md         # 补全清单
├── config-api/
│   ├── spec.md          # 配置读写接口行为契约
│   ├── design.md        # 配置读写架构设计
│   └── tasks.md         # 补全清单
└── yang-api/
    ├── spec.md          # YANG模块接口行为契约
    ├── design.md        # YANG模块架构设计
    └── tasks.md         # 补全清单
```

---

### Task 1: 创建 devices-api/spec.md

**Files:**
- Create: `openspec/specs/devices-api/spec.md`
- Reference: `backend/internal/api/device_handler.go`
- Reference: `backend/internal/api/response.go`

- [ ] **Step 1: 编写 devices-api/spec.md**

从代码还原以下内容：

```markdown
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
```

- [ ] **Step 2: 提交**

```bash
git add openspec/specs/devices-api/spec.md
git commit -m "docs(openspec): 补齐 devices-api 行为契约 spec"
```

---

### Task 2: 创建 devices-api/design.md

**Files:**
- Create: `openspec/specs/devices-api/design.md`
- Reference: `backend/internal/api/device_handler.go`
- Reference: `backend/internal/api/server.go`

- [ ] **Step 1: 编写 devices-api/design.md**

```markdown
# devices-api - 架构设计

## 请求处理流程

### GET /api/v1/devices
```
Request → DeviceHandler.ListDevices()
  → mu.RLock() 读锁
  → 遍历 devices map 组装列表
  → manager.GetClientPool().Stats() 获取连接池统计
  → mu.RUnlock()
  → Success(devices + stats)
```

### POST /api/v1/devices
```
Request → DeviceHandler.AddDevice()
  → ShouldBindJSON() 解析请求体
  → 校验必填字段 (ip/username/password)
  → port=0 时默认830
  → mu.Lock() 写锁
  → devices[ip] = DeviceInfo{...} （覆盖写）
  → mu.Unlock()
  → pool.Get(DeviceConnectionInfo) 尝试NETCONF连接
  → 连接成功 → Success
  → 连接失败 → Error(500) （设备信息已保存）
```

### DELETE /api/v1/devices/:ip
```
Request → DeviceHandler.RemoveDevice()
  → c.Param("ip") 获取路径参数
  → mu.Lock() 写锁
  → delete(devices, ip)
  → mu.Unlock()
  → pool.Release(ip) 释放连接池资源
  → Success
```

### GET /api/v1/devices/:ip/status
```
Request → DeviceHandler.GetStatus()
  → c.Param("ip") 获取路径参数
  → mu.RLock() 读锁
  → 检查设备是否存在
  → mu.RUnlock()
  → 不存在 → Error(404)
  → 存在 → pool.Get(DeviceConnectionInfo) 获取/创建连接
  → cli.IsConnected() 判断连接状态
  → Success({running:true, connected:bool})
```

## 依赖关系

| 依赖 | 用途 | 调用位置 |
|------|------|----------|
| manager.Manager | 获取ClientPool | 所有handler |
| ClientPool.Get() | 获取/创建设备连接 | AddDevice, GetStatus |
| ClientPool.Release() | 释放设备连接 | RemoveDevice |
| ClientPool.Stats() | 连接池统计 | ListDevices |
| sync.RWMutex | 设备map并发安全 | 所有handler |
| client.DeviceConnectionInfo | 设备连接参数结构 | AddDevice, GetStatus |

## 错误处理策略

- **统一HTTP 200**：所有响应的HTTP状态码均为200，业务错误通过JSON body的`code`字段区分
- **成功**：`code=0, success=true`
- **客户端错误**：`code=400 (参数校验), code=404 (设备不存在)`
- **服务端错误**：`code=500 (连接失败), code=503 (设备离线)`
- **静默处理**：删除不存在的设备不报错，map delete对不存在的key为no-op
- **部分成功**：添加设备时连接失败，设备信息已保存但返回错误

## 数据存储

- 设备信息存储在内存 `map[string]DeviceInfo`（非持久化）
- 并发保护：`sync.RWMutex`（读操作RLock，写操作Lock）
- 默认预置一个测试设备 `192.168.1.1`（hardcoded in NewDeviceHandler）
- 无JSON文件持久化（与CLAUDE.md描述的"本地JSON元信息文件"不一致）
```

- [ ] **Step 2: 提交**

```bash
git add openspec/specs/devices-api/design.md
git commit -m "docs(openspec): 补齐 devices-api 架构设计 design"
```

---

### Task 3: 创建 devices-api/tasks.md

**Files:**
- Create: `openspec/specs/devices-api/tasks.md`

- [ ] **Step 1: 编写 devices-api/tasks.md**

```markdown
# devices-api - 补全清单

## spec 与代码差异

- [ ] **响应格式不符合RESTful惯例**：所有错误响应HTTP状态码均为200，应使用标准HTTP状态码（400/404/500/503）
- [ ] **密码明文返回**：GET /devices 响应中 password 明文返回，应脱敏处理
- [ ] **密码明文存储**：设备凭据在内存map中明文存储，无加密
- [ ] **设备信息无持久化**：代码使用内存map存储设备，重启后丢失；CLAUDE.md要求"本地JSON元信息文件"存储
- [ ] **默认测试设备**：NewDeviceHandler硬编码预置192.168.1.1设备，生产环境不应存在
- [ ] **删除设备不校验存在性**：DELETE不存在的IP返回成功，可能造成调用方误判
- [ ] **添加设备连接失败仍保存**：AddDevice连接失败后设备信息已写入内存，行为不原子
- [ ] **GetStatus每次创建连接**：GetStatus调用pool.Get()获取连接，可能触发新建连接而非复用
- [ ] **无IP格式校验**：AddDevice未校验ip字段是否为合法IPv4/IPv6地址
- [ ] **无端口范围校验**：AddDevice未校验port是否在合法范围(1-65535)

## 后续改进建议

- [ ] 引入标准HTTP状态码，错误响应使用对应HTTP码
- [ ] GET响应中密码字段脱敏（返回"***"或omit）
- [ ] 实现设备元信息JSON文件持久化
- [ ] 移除默认测试设备hardcode
- [ ] DELETE不存在的设备返回404
- [ ] AddDevice连接失败时回滚设备信息写入
- [ ] 添加IP格式和端口范围校验
```

- [ ] **Step 2: 提交**

```bash
git add openspec/specs/devices-api/tasks.md
git commit -m "docs(openspec): 补齐 devices-api 补全清单 tasks"
```

---

### Task 4: 创建 config-api/spec.md

**Files:**
- Create: `openspec/specs/config-api/spec.md`
- Reference: `backend/internal/api/config_handler.go`

- [ ] **Step 1: 编写 config-api/spec.md**

```markdown
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
```

- [ ] **Step 2: 提交**

```bash
git add openspec/specs/config-api/spec.md
git commit -m "docs(openspec): 补齐 config-api 行为契约 spec"
```

---

### Task 5: 创建 config-api/design.md

**Files:**
- Create: `openspec/specs/config-api/design.md`

- [ ] **Step 1: 编写 config-api/design.md**

```markdown
# config-api - 架构设计

## 请求处理流程

### GET /api/v1/config/:ip/*path
```
Request → ConfigHandler.GetConfig()
  → c.Param("ip"), c.Param("path")
  → c.Query("force_refresh") 解析但忽略
  → pool.Get(DeviceConnectionInfo{IP:ip})
  → 获取失败 → Error(500)
  → cli.IsConnected() == false → Error(503)
  → context.WithTimeout(10s)
  → cli.Get(path, WithDatastore("running"))
  → NETCONF失败 → Error(500)
  → Success({data: result.Data})
```

### POST /api/v1/config/:ip/*path
```
Request → ConfigHandler.SetConfig()
  → c.Param("ip"), c.Param("path")
  → ShouldBindJSON(&data) 解析请求body
  → 解析失败 → Error(400)
  → convertToTypedStruct(path, data)
    → path含"system:" → convertMapToHuaweiSystem
    → path含"ifm:ifm"+"interfaces" → convertMapToHuaweiIfm
    → path含"vlan:"+"vlan/vlans" → convertMapToHuaweiVlan
    → 其他 → 原始map回退
  → 转换失败 → Error(400)
  → configStore.Set(ip, path, desiredConfig)
  → 写入失败 → Error(500)
  → manager.TriggerReconcile(ip, path)
  → Success({status:"ACCEPTED", reconciliation:{triggered, message}})
```

## 依赖关系

| 依赖 | 用途 | 调用位置 |
|------|------|----------|
| manager.Manager | 获取ClientPool/ConfigStore | GetConfig, SetConfig |
| ClientPool.Get() | 获取设备NETCONF连接 | GetConfig |
| client.Device | NETCONF协议操作 | GetConfig |
| ConfigStore.Set() | 存储期望配置 | SetConfig |
| Manager.TriggerReconcile() | 触发异步Reconciliation | SetConfig |
| generated/huawei | YANG强类型结构体 | SetConfig(convertToTypedStruct) |
| context.WithTimeout | NETCONF操作10秒超时 | GetConfig |

## 错误处理策略

- **GET错误链**：pool.Get失败 → 500; 设备离线 → 503; NETCONF Get失败 → 500; 超时 → 500
- **POST错误链**：JSON解析失败 → 400; YANG类型转换失败 → 400; ConfigStore写入失败 → 500
- **声明式语义**：POST成功仅表示配置已接受，不代表设备已配置；实际配置由Reconciler异步完成
- **类型转换容错**：未知YANG路径回退到原始map，不阻断请求

## YANG类型转换路由

| 路径关键字 | 转换函数 | 目标结构体 |
|-----------|----------|-----------|
| `system:` | convertMapToHuaweiSystem | HuaweiSystem_System |
| `ifm:ifm` + `interfaces` | convertMapToHuaweiIfm | HuaweiIfm_Ifm_Interfaces |
| `vlan:` + `vlan`/`vlans` | convertMapToHuaweiVlan | HuaweiVlan_Vlan_Vlans |
| 其他 | 无转换（原始map） | map[string]interface{} |
```

- [ ] **Step 2: 提交**

```bash
git add openspec/specs/config-api/design.md
git commit -m "docs(openspec): 补齐 config-api 架构设计 design"
```

---

### Task 6: 创建 config-api/tasks.md

**Files:**
- Create: `openspec/specs/config-api/tasks.md`

- [ ] **Step 1: 编写 config-api/tasks.md**

```markdown
# config-api - 补全清单

## spec 与代码差异

- [ ] **force_refresh未实现**：GET接口声明了force_refresh查询参数但代码中仅解析不使用（TODO注释）
- [ ] **GetConfig设备信息不完整**：pool.Get()只传IP，未传port/username/password，连接创建可能失败
- [ ] **无设备存在性校验**：GetConfig/SetConfig未校验设备IP是否已注册
- [ ] **响应格式不符合RESTful**：所有HTTP状态码为200，错误用JSON code区分
- [ ] **类型转换字段名大小写兼容**：convertToTypedStruct对字段名做了大量case兼容（ifName/Interface/vlans/Vlan），增加维护成本
- [ ] **VLAN结构体拼写错误**：代码中UnkownUnicastDiscard应为UnknownUnicastDiscard
- [ ] **无请求体大小限制**：SetConfig未限制请求body大小，可能造成内存问题
- [ ] **Reconcile失败无反馈**：SetConfig触发Reconcile后无后续状态追踪，客户端无法知道配置是否成功下发

## 后续改进建议

- [ ] 实现force_refresh缓存失效逻辑
- [ ] GetConfig传递完整设备连接信息
- [ ] 添加设备存在性校验
- [ ] 引入标准HTTP状态码
- [ ] 统一YANG字段命名约定，减少case兼容代码
- [ ] 修复UnkownUnicastDiscard拼写错误
- [ ] 添加请求体大小限制
- [ ] 提供Reconcile状态查询接口
```

- [ ] **Step 2: 提交**

```bash
git add openspec/specs/config-api/tasks.md
git commit -m "docs(openspec): 补齐 config-api 补全清单 tasks"
```

---

### Task 7: 创建 yang-api/spec.md

**Files:**
- Create: `openspec/specs/yang-api/spec.md`
- Reference: `backend/internal/api/yang_handler.go`

- [ ] **Step 1: 编写 yang-api/spec.md**

```markdown
# yang-api - 行为契约

## 接口定义

### GET /api/v1/yang/modules

**描述**：返回所有已支持的YANG模块列表

**参数**：无

**响应**：
| 状态码 | 含义 | Body |
|--------|------|------|
| 200 | 成功 | `{"code":0,"data":[...],"success":true}` |

**数据模型 — YangModuleInfo**：
| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| name | string | 模块名 | "huawei-ifm" |
| title | string | 中文标题 | "华为接口管理" |
| vendor | string | 固定"huawei" | "huawei" |
| path | string | YANG根路径 | "/ifm" |
| description | string | 模块描述 | "Network interfaces configuration" |
| type | string | 根节点类型（数字字符串） | "1" |

### GET /api/v1/yang/schema/:module

**描述**：返回指定YANG模块的动态表单Schema定义

**参数**：
| 参数 | 位置 | 类型 | 必填 | 约束 | 示例 |
|------|------|------|------|------|------|
| module | path | string | 是 | YANG模块名 | "huawei-ifm" |

**响应**：
| 状态码 | 含义 | Body |
|--------|------|------|
| 200 | 成功 | `{"code":0,"data":{...},"success":true}` |

**数据模型 — YangSchema**：
| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| module | string | 模块名 | "huawei-ifm" |
| title | string | 中文标题 | "华为接口管理" |
| vendor | string | 固定"huawei" | "huawei" |
| fields | FieldDef[] | 表单字段定义 | 见下方 |
| listCols | FieldDef[] | 列表视图列定义 | 见下方 |

**数据模型 — FieldDef**：
| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| path | string | 字段路径 | "ifName" |
| type | string | 字段类型(string/number/enum/boolean) | "string" |
| label | string | 中文标签 | "接口名称" |
| placeholder | string | 可选，占位提示 | "例如: GE0/0/1" |
| required | bool | 可选，是否必填 | true |
| pattern | string | 可选，正则校验 | "^[0-9]+$" |
| default | any | 可选，默认值 | 1500 |
| options | Option[] | 可选，enum选项 | 见下方 |
| group | string | 可选，分组名 | "基本信息" |
| minimum | int | 可选，最小值 | 1 |
| maximum | int | 可选，最大值 | 4094 |
| readonly | bool | 可选，只读标记 | false |

**数据模型 — Option**：
| 字段 | 类型 | 约束 | 示例 |
|------|------|------|------|
| label | string | 选项中文显示 | "启用" |
| value | any | 选项值 | "up" |

## 业务规则

### BR-01: 模块列表-有已加载模块

- Given: Manager中已加载YANG模块
- When: 调用 GET /api/v1/yang/modules
- Then: 从Schema.Modules()遍历返回实际模块列表

### BR-02: 模块列表-无已加载模块

- Given: Manager中无已加载YANG模块
- When: 调用 GET /api/v1/yang/modules
- Then: 返回硬编码的示例模块列表（huawei-ifm + huawei-vlan）

### BR-03: Schema-已知模块

- Given: 请求模块名为 "huawei-ifm" / "Interfaces" / "huawei-vlan" / "VLANs"
- When: 调用 GET /api/v1/yang/schema/:module
- Then: 返回预定义的完整Schema（含fields和listCols）

### BR-04: Schema-未知模块

- Given: 请求模块名不匹配任何预定义模块
- When: 调用 GET /api/v1/yang/schema/:module
- Then: 返回通用Schema（仅含name+description两个字段）

## 数据模型

### YangModuleInfo 示例

```json
{
  "name": "huawei-ifm",
  "title": "华为接口管理",
  "vendor": "huawei",
  "path": "/ifm",
  "description": "Network interfaces configuration",
  "type": "1"
}
```

### YangSchema 示例（huawei-vlan）

```json
{
  "module": "huawei-vlan",
  "title": "华为 VLAN 配置",
  "vendor": "huawei",
  "fields": [
    {"path": "vlanId", "type": "number", "label": "VLAN ID", "required": true, "minimum": 1, "maximum": 4094, "group": "基本信息"},
    {"path": "vlanName", "type": "string", "label": "VLAN 名称", "placeholder": "例如: VLAN-100", "group": "基本信息"},
    {"path": "description", "type": "string", "label": "描述", "group": "基本信息"},
    {"path": "portList", "type": "string", "label": "端口列表", "placeholder": "例如: GE0/0/1,GE0/0/2", "group": "端口配置"}
  ],
  "listCols": [
    {"path": "vlanId", "type": "number", "label": "VLAN ID"},
    {"path": "vlanName", "type": "string", "label": "VLAN 名称"}
  ]
}
```
```

- [ ] **Step 2: 提交**

```bash
git add openspec/specs/yang-api/spec.md
git commit -m "docs(openspec): 补齐 yang-api 行为契约 spec"
```

---

### Task 8: 创建 yang-api/design.md

**Files:**
- Create: `openspec/specs/yang-api/design.md`

- [ ] **Step 1: 编写 yang-api/design.md**

```markdown
# yang-api - 架构设计

## 请求处理流程

### GET /api/v1/yang/modules
```
Request → YangHandler.ListModules()
  → manager.GetSchema().Modules() 遍历已加载模块
  → 每个模块提取: Name, Root.Name, Root.Description, Root.Type
  → vendor固定为"huawei"
  → 无模块 → 返回硬编码示例列表(huawei-ifm, huawei-vlan)
  → Success(modules)
```

### GET /api/v1/yang/schema/:module
```
Request → YangHandler.GetSchema()
  → c.Param("module") 获取模块名
  → switch module:
    → "huawei-ifm" / "Interfaces" → 预定义IFM Schema
    → "huawei-vlan" / "VLANs" → 预定义VLAN Schema
    → default → 通用Schema(name + description)
  → Success(schema)
```

## 依赖关系

| 依赖 | 用途 | 调用位置 |
|------|------|----------|
| manager.Manager | 获取Schema | ListModules |
| Schema.Modules() | 遍历已加载YANG模块 | ListModules |

## 错误处理策略

- **无错误场景**：两个端点在任何输入下均返回200成功
- **未知模块回退**：请求不存在的模块名时返回通用Schema而非错误
- **空模块回退**：无已加载模块时返回硬编码示例

## 硬编码数据

当前Schema全部硬编码在handler中，未从YANG模型文件动态生成：
- ListModules: 无模块时返回2个示例
- GetSchema: IFM和VLAN各有一套预定义FieldDef，未知模块返回通用2字段Schema
```

- [ ] **Step 2: 提交**

```bash
git add openspec/specs/yang-api/design.md
git commit -m "docs(openspec): 补齐 yang-api 架构设计 design"
```

---

### Task 9: 创建 yang-api/tasks.md

**Files:**
- Create: `openspec/specs/yang-api/tasks.md`

- [ ] **Step 1: 编写 yang-api/tasks.md**

```markdown
# yang-api - 补全清单

## spec 与代码差异

- [ ] **Schema硬编码**：GetSchema的IFM/VLAN Schema全部硬编码在handler中，未从YANG模型文件动态生成
- [ ] **模块列表硬编码回退**：ListModules在无模块时返回2个固定示例，非真实数据
- [ ] **vendor固定为huawei**：代码中vendor字段写死为"huawei"，不支持其他厂商
- [ ] **type字段为数字字符串**：YangModuleInfo.type是Root.Type()的数字字符串表示，语义不明确
- [ ] **模块名别名映射**："Interfaces"映射到huawei-ifm，"VLANs"映射到huawei-vlan，映射关系硬编码
- [ ] **未知模块无错误**：请求不存在的模块返回通用Schema而非404
- [ ] **FieldDef与ygot结构体不同步**：Schema中的字段定义与generated/huawei结构体手动维护，容易不一致

## 后续改进建议

- [ ] 从YANG模型文件动态生成Schema（ygot→FieldDef自动映射）
- [ ] 移除硬编码示例模块，无模块时返回空列表
- [ ] 支持多厂商vendor字段
- [ ] 未知模块返回404而非通用Schema
- [ ] FieldDef自动从ygot结构体注解生成，保证同步
```

- [ ] **Step 2: 提交**

```bash
git add openspec/specs/yang-api/tasks.md
git commit -m "docs(openspec): 补齐 yang-api 补全清单 tasks"
```

---

## Plan Self-Review

**1. Spec coverage:** 设计文档中的4个change（devices-api/config-api/yang-api/sse-api），前3个已有完整task覆盖。sse-api经代码确认REST API层中不存在SSE端点，已从范围排除。

**2. Placeholder scan:** 无TBD/TODO/placeholder，每个step包含完整文档内容。

**3. Type consistency:** 所有数据模型字段名与代码中的Go struct tag一致（ip/port/username/password/active_connections/total_connections/errors/running/connected/status/triggered/message）。

**4. 文件路径:** 所有openspec/specs/下文件路径与设计文档一致。
