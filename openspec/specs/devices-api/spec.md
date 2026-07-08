# devices-api — 设备注册与连接状态北向接口

## Purpose

devices-api 是 Stack B 北向 REST 接口，提供设备的注册（`POST /api/v1/devices`）、注销（`DELETE /api/v1/devices/:ip`）、列表（`GET /api/v1/devices`）与连接状态查询（`GET /api/v1/devices/:ip/status`）。设备连接信息（IP/端口/凭据）统一经共享 **DeviceStore**（Manager 级内存注册表）存取，供 config-api 等其他能力复用（见 [[device-store]]）。全部端点遵循 BR-12 统一响应格式（HTTP 恒 200，业务成败由 body 的 `code`/`success` 区分）。

## Requirements

### Requirement: BR-01 设备列表返回

`GET /api/v1/devices` SHALL 经共享 DeviceStore 返回全部已注册设备信息及连接池统计（`active_connections`/`total_connections`/`errors`）。设备信息含明文 `password`（属既有行为，保留）。

#### Scenario: 存在已注册设备
- **WHEN** DeviceStore 中存在 N 个已注册设备且调用 `GET /api/v1/devices`
- **THEN** SHALL 返回全部 N 个设备信息（含明文 password）与连接池统计，`code=0`、`success=true`

### Requirement: BR-02 空设备列表

DeviceStore 为空时，`GET /api/v1/devices` SHALL 返回空数组而非 null 或错误。

#### Scenario: 无已注册设备
- **WHEN** DeviceStore 为空且调用 `GET /api/v1/devices`
- **THEN** SHALL 返回 `"devices":[]` 与连接池统计，`code=0`、`success=true`

### Requirement: BR-03 设备注册默认端口

`POST /api/v1/devices` 请求体未提供 `port` 或 `port=0` 时，SHALL 自动将端口置为 830（NETCONF SSH 默认端口，R02）后写入 DeviceStore。

#### Scenario: 缺省端口补齐
- **WHEN** 提交注册请求且 `port` 为 0 或缺省
- **THEN** 写入 DeviceStore 的设备 `port` SHALL 为 830

### Requirement: BR-04 设备注册必填字段校验

`POST /api/v1/devices` 缺少 `ip`/`username`/`password` 任一字段时 SHALL 返回 `code=400`（message 含 "Invalid request"），SHALL NOT 写入 DeviceStore。

#### Scenario: 缺失必填字段
- **WHEN** 请求体缺少 `ip`、`username` 或 `password`
- **THEN** SHALL 返回 `code=400`、`success=false`，不写 DeviceStore

### Requirement: BR-05 设备注册重复 IP 覆盖

提交已存在 IP 的注册请求时，`AddDevice` SHALL 幂等覆盖 DeviceStore 中原有条目（不报错），随后尝试建连。

#### Scenario: 重复 IP 覆盖
- **WHEN** DeviceStore 已存在 IP=X，再次以相同 IP 提交注册
- **THEN** SHALL 覆盖原条目而不报错，并继续尝试 NETCONF 连接

### Requirement: BR-06 设备注册连接失败仍保存

注册流程 SHALL 先经 `AddDevice` 持久化设备信息到共享 DeviceStore，再尝试 NETCONF 连接；建连失败时 SHALL 返回 `code=500`（连接失败），但设备信息 SHALL 已保存在 DeviceStore 中。

#### Scenario: 凭据正确但设备离线
- **WHEN** 设备 IP/凭据合法但设备不在线
- **THEN** 设备信息 SHALL 已写入 DeviceStore，响应 SHALL 返回 `code=500`、`success=false`

### Requirement: BR-07 设备删除

`DELETE /api/v1/devices/:ip` SHALL 经 `RemoveDevice` 从共享 DeviceStore 删除设备条目，并调用 `pool.Release(ip)` 释放连接池资源。

#### Scenario: 删除已注册设备
- **WHEN** DeviceStore 存在 IP=X 且调用 `DELETE /api/v1/devices/:ip`
- **THEN** SHALL 从 DeviceStore 删除该条目、释放连接，返回 `code=0`、`success=true`

### Requirement: BR-08 设备删除幂等（不存在）

删除不存在的设备 SHALL NOT 报错，`RemoveDevice` 对不存在的 key 无操作，仍返回成功（幂等）。

#### Scenario: 删除不存在的设备
- **WHEN** DeviceStore 无 IP=X 且调用 `DELETE /api/v1/devices/:ip`
- **THEN** SHALL 返回 `code=0`、`success=true`，不产生副作用

### Requirement: BR-09 设备状态在线

`GET /api/v1/devices/:ip/status` 对已注册且 NETCONF 连接活跃的设备 SHALL 返回 `running=true`、`connected=true`。

#### Scenario: 连接活跃
- **WHEN** 设备经 DeviceStore 已注册且 NETCONF 连接活跃
- **THEN** SHALL 返回 `{"running":true,"connected":true}`、`code=0`

### Requirement: BR-10 设备状态离线

`GET /api/v1/devices/:ip/status` 对已注册但连接断开的设备 SHALL 返回 `running=true`、`connected=false`（服务运行中，仅设备连接不可用，不 panic，R08）。

#### Scenario: 连接断开
- **WHEN** 设备经 DeviceStore 已注册但 NETCONF 连接断开
- **THEN** SHALL 返回 `{"running":true,"connected":false}`、`code=0`

### Requirement: BR-11 设备状态不存在

`GET /api/v1/devices/:ip/status` 查询未在 DeviceStore 注册的 IP 时 SHALL 返回 `code=404`（"Device not found"）。

#### Scenario: 未注册设备
- **WHEN** DeviceStore 无该 IP 且查询其状态
- **THEN** SHALL 返回 `code=404`、`success=false`

### Requirement: BR-12 统一响应格式

devices-api 全部端点 SHALL 恒以 HTTP 200 返回，业务成功/失败经 JSON body 的 `code` 与 `success` 字段区分，前端 SHALL NOT 依赖 HTTP 状态码判定业务结果。

#### Scenario: 业务错误仍返回 HTTP 200
- **WHEN** 任一请求发生业务错误（如 400/404/500）
- **THEN** HTTP 状态码 SHALL 恒为 200，错误 SHALL 由 body `code`/`success` 表达
