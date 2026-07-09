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

### Requirement: BR-09 行删除通道（DELETE，命令语义）

`DELETE /api/v1/config/:ip/*path?key=<条目主键>` SHALL 以命令语义删除列表单条目：按 path 路由模型分支解析 key → 从 ConfigStore desired 移除该键（与合并写同临界区，幂等）→ 构造仅含 key 叶、带 `operation="delete"` 的 edit-config **同步下发**（candidate→commit）→ 成功后失效该设备运行缓存、记录审计、触发对账验证收敛。下发失败（含设备 data-missing）SHALL 原样透出错误、SHALL NOT 失效缓存或写审计。POST 合并语义与 `walkMap` merge/subset 收敛语义 SHALL NOT 因删除通道改变。

#### Scenario: 删除成功端到端

- **WHEN** 设备存在 vlan 10，请求 `DELETE /config/<ip>/vlan:vlan/vlan:vlans?key=10`
- **THEN** SHALL 返回成功，回读 SHALL 不再含 vlan 10（删除元素仅含 key 叶 `<id>`）
- **AND** desired SHALL 已移除键 10，二轮对账 SHALL 0 change（不复活、不漂移）

#### Scenario: 删除不存在条目（负路径）

- **WHEN** 请求删除设备上不存在的条目键
- **THEN** SHALL 透出设备错误（如 data-missing），SHALL NOT 失效缓存

#### Scenario: 非法 key / 未知路径（负路径）

- **WHEN** key 缺失、类型不合模型（如 vlan key 非整数）、或 path 无模型分支
- **THEN** SHALL 返回 `code=400`，SHALL NOT 触达设备

#### Scenario: 并发删除（R09）

- **WHEN** 并发对同一 path 发起删除与合并下发
- **THEN** desired 读改写 SHALL 串行化，SHALL NOT 丢更新或竞态（-race 全绿）

### Requirement: BR-10 删除的模型驱动门禁

DELETE SHALL 在触达设备前按 schema 校验：目标 list 节点 `operation-exclude` 含 `delete`、或目标节点为 readonly（config false）时 SHALL 拒绝（400 + 明确错误），与前端按钮门禁互为防御。schema 查不到该路径时 SHALL 放行（降级，R08——设备/转换层为最终权威）。

#### Scenario: operation-exclude 拒绝

- **WHEN** 目标 list 在模型中标注 `ext:operation-exclude` 含 delete
- **THEN** SHALL 返回 400 且 SHALL NOT 触达设备

#### Scenario: readonly 路径拒绝

- **WHEN** 目标路径为 `config false` state 子树
- **THEN** SHALL 返回 400 且 SHALL NOT 触达设备

#### Scenario: schema 未覆盖路径降级放行（边界）

- **WHEN** 路径在 schema 中查不到对应节点
- **THEN** 门禁 SHALL 放行，由模型分支路由/设备决定成败
