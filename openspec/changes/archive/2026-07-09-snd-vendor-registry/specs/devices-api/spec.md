# devices-api delta — snd-vendor-registry

## MODIFIED Requirements

### Requirement: BR-04 设备注册必填字段校验

`POST /api/v1/devices` 缺少 `ip`/`username`/`password` 任一字段时 SHALL 返回 `code=400`（message 含 "Invalid request"），SHALL NOT 写入 DeviceStore。请求体 MAY 携带可选 `vendor` 字段：缺省 SHALL 置为 `huawei`；携带无已注册驱动的厂商值时 SHALL 返回 `code=400` 并 SHALL NOT 写入 DeviceStore（早失败，优于下发时报错）。

#### Scenario: 缺失必填字段
- **WHEN** 请求体缺少 `ip`、`username` 或 `password`
- **THEN** SHALL 返回 `code=400`、`success=false`，不写 DeviceStore

#### Scenario: vendor 缺省补齐
- **WHEN** 注册请求未携带 `vendor`
- **THEN** 写入 DeviceStore 的设备 `Vendor` SHALL 为 `huawei`，注册行为与既有语义完全一致

#### Scenario: 未知厂商拒绝（负路径）
- **WHEN** 注册请求携带 `vendor="nokia"`（无已注册驱动）
- **THEN** SHALL 返回 `code=400` 并明确提示不支持的厂商，不写 DeviceStore
