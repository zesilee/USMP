# config-api — 设备配置读写北向接口

## Purpose

config-api 是 Stack B 北向 REST 接口，提供设备运行配置的**读**（`GET /api/v1/config/:ip/*path`，带 TTL 缓存 + `force_refresh` 绕缓存回读）与**声明式下发**（`POST …` → 存入 ConfigStore → 触发异步对账）。连接信息（IP/端口/凭据/协议）统一由共享 DeviceStore 解析（见 [[device-store]]）。

## Requirements

### Requirement: BR-01 配置读取（缓存优先）

`GET /api/v1/config/:ip/*path` SHALL 优先返回运行缓存（§8 TTL 30s）中的新鲜配置；缓存未命中时 SHALL 经共享 DeviceStore 解析的连接从设备回读（NETCONF get-config，running 数据源），并回填缓存。回读结果 SHALL 为 RFC7951 结构（如 `{"interface":[{"name":…}]}`），可被前端列表化，而非裸 XML 字节。响应 SHALL 携带 `cached` / `cache_age_seconds` / `ttl_seconds` / `source`（`cache`\|`device`）。

#### Scenario: 缓存命中
- **WHEN** 距上次读取 < TTL 且未带 `force_refresh`
- **THEN** SHALL 返回缓存数据，`source="cache"`、`cached=true`，不访问设备

#### Scenario: 缓存未命中回读设备
- **WHEN** 缓存过期/无 且设备已在 DeviceStore 注册
- **THEN** SHALL 用库中凭据回读设备，返回 RFC7951 结构，`source="device"`，并回填缓存

### Requirement: BR-02 读取降级（离线/未连接/未注册）

读取路径 SHALL NOT panic（R08）。设备连接建立失败 SHALL 返回 `code=500`；连接存在但未就绪（`IsConnected()=false`）SHALL 返回 `code=503`。设备未在 DeviceStore 注册时以 AUTO/无凭据连接、认证失败 SHALL 归为连接错误返回。

#### Scenario: 设备未连接
- **WHEN** 回读时客户端 `IsConnected()=false`
- **THEN** SHALL 返回 `code=503` "Device is not connected"

#### Scenario: 建连失败
- **WHEN** 连接池建连报错
- **THEN** SHALL 返回 `code=500`，其余请求不受影响

### Requirement: BR-03 读取超时

设备回读 SHALL 受 10s 上下文超时约束；超时 SHALL 返回 `code=500` 且不阻塞。

#### Scenario: get-config 超时
- **WHEN** 设备回读超过 10s
- **THEN** context 取消，SHALL 返回 `code=500`

### Requirement: BR-04 force_refresh 绕缓存回读

`force_refresh=true` 查询参数 SHALL 绕过缓存、强制从设备回读并回填缓存（已实现；取代早期"参数被忽略"的行为）。

#### Scenario: 强制刷新
- **WHEN** 带 `force_refresh=true`
- **THEN** SHALL 跳过缓存直接回读设备，`source="device"`

### Requirement: BR-05 声明式下发

`POST /api/v1/config/:ip/*path` SHALL 将 JSON 配置转为强类型 ygot 结构 → 存入 ConfigStore → 触发对账，返回 `status="ACCEPTED"`。下发即接受语义：配置**存储成功即返回**，实际对齐设备由异步对账完成。

#### Scenario: 下发被接受
- **WHEN** 提交合法 YANG 路径 + JSON 配置
- **THEN** SHALL 存入 ConfigStore、触发对账，返回 `ACCEPTED` + `reconciliation.triggered`

### Requirement: BR-06 类型转换路由

下发 SHALL 按 path 关键字路由到对应转换函数：含 `system:`→System、含 `ifm:ifm`+`interfaces`→Ifm、含 `vlan:`+`vlan/vlans`→Vlan；其余回退原始 map。

#### Scenario: 按路径路由
- **WHEN** path 含 `ifm:ifm/ifm:interfaces`
- **THEN** SHALL 用 `convertMapToHuaweiIfm` 转换为 `HuaweiIfm_Ifm_Interfaces`

### Requirement: BR-07 对账异步触发

`TriggerReconcile(ip, path)` 的返回 SHALL 表示是否命中对应 Controller；无论是否命中，配置 SHALL 已完成存储。

#### Scenario: 无匹配 Controller
- **WHEN** 该 path 无注册 Controller
- **THEN** `reconciliation.triggered=false`，但配置仍已存储、响应 `ACCEPTED`

### Requirement: BR-08 无效请求拒绝

非法 JSON 或类型转换失败 SHALL 返回 `code=400`，SHALL NOT 存储或触发对账。

#### Scenario: 非法 JSON
- **WHEN** 请求 body 非合法 JSON
- **THEN** SHALL 返回 `code=400` "Invalid request"，不写 ConfigStore
