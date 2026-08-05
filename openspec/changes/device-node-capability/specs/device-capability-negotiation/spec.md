## ADDED Requirements

### Requirement: CN-04 节点级不支持集学习

系统 SHALL 在设备对配置读/写返回 `unknown-element` 类 rpc-error（error-tag=unknown-element 或 bad-element，severity=error）且 bad-element 与请求 YANG 路径段名匹配时，将该请求路径记入该设备的**节点级不支持集**。不支持集 SHALL 存于连接层内存（与 hello capabilities 同生命周期），断线重连 SHALL 清空重学；SHALL NOT 持久化到 CRD/磁盘（对齐 CN-01/R03）。bad-element 与请求路径不匹配的 rpc-error SHALL NOT 入集（按普通设备错误透传）。

#### Scenario: unknown-element 学习入集
- **WHEN** 对设备 D 读取路径 `devm:devm/devm:cards`，设备回 error-tag=unknown-element、bad-element=cards
- **THEN** (D, 该路径) SHALL 入不支持集，且后续查询可见

#### Scenario: 归因不匹配不入集（负路径）
- **WHEN** 设备回 unknown-element 但 bad-element 与请求路径任一段名均不匹配
- **THEN** SHALL NOT 入集，错误 SHALL 按普通设备错误返回

#### Scenario: 重连清空
- **WHEN** 设备 D 断线重连
- **THEN** D 的不支持集 SHALL 为空（重新学习）

#### Scenario: 并发安全
- **WHEN** 多协程并发读写同一设备的不支持集
- **THEN** SHALL 无数据竞态（race 检测通过）

### Requirement: CN-05 不支持集按设备透出

`GET /api/v1/yang/schema/:module?device=<ip>` 携带 `device` 参数时，响应 SHALL 附 `unsupported` 数组：该模块下已学习的不支持子路径（相对模块根的首段名）；空集 SHALL 省略该键。无 `device` 参数 SHALL 保持既有响应不变（向后兼容）。`device` 指向未注册设备 SHALL 沿用 CN-02 的 404 语义。

#### Scenario: 已学习页签透出
- **WHEN** 设备 D 已学习 `devm:devm/devm:cards` 不支持，请求 `GET /yang/schema/devm?device=D`
- **THEN** 响应 SHALL 含 `unsupported:["cards"]`

#### Scenario: 无参数向后兼容
- **WHEN** 请求 `GET /yang/schema/devm`（无 device）
- **THEN** 响应 SHALL NOT 含 `unsupported` 键，其余内容与既有契约一致

### Requirement: CN-06 hello capabilities 原文透出

`GET /api/v1/devices/:ip/capabilities` SHALL 返回该设备 hello 报文自报的完整 capabilities 列表（原文字符串数组，不加工），供诊断与 deviations 侦察（二期捷径评估）。设备未注册 SHALL 404；已注册但能力不可得（离线且建连失败）SHALL 返回空列表 + `negotiated:false`（对齐 CN-02 降级口径，SHALL NOT 5xx）。

#### Scenario: 在线设备透出原文
- **WHEN** 设备已连接，请求 `GET /devices/:ip/capabilities`
- **THEN** SHALL 返回 hello capabilities 原文数组（含 base 能力与模块能力 URI）

#### Scenario: 离线降级（负路径）
- **WHEN** 设备已注册但离线且建连失败
- **THEN** SHALL 返回空列表 + `negotiated:false`，SHALL NOT 5xx
