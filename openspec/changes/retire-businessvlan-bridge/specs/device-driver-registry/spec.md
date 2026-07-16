## ADDED Requirements

### Requirement: DR-04 厂商支持性查询

注册表 SHALL 提供 `VendorSupported(vendor string) bool` 查询（含包级 facade）：存在任一描述符的 `Vendor` 与入参**大小写无关**匹配时返回 true，否则返回 false；SHALL NOT panic（R08）。查询与 `Register` 并发调用 SHALL 无数据竞态（R09）。该查询是「厂商是否有已注册驱动」的唯一事实源，供 devices-api BR-04 厂商门禁消费（取代已退役 translation-engine 注册表的 `IsVendorSupported`）。

#### Scenario: 已注册厂商大小写无关命中
- **WHEN** 注册表含 `Vendor="huawei"` 的描述符，查询 `VendorSupported("Huawei")`
- **THEN** SHALL 返回 true

#### Scenario: 未注册厂商（负路径）
- **WHEN** 查询 `VendorSupported("nokia")` 且无任何 nokia 描述符
- **THEN** SHALL 返回 false，SHALL NOT panic

#### Scenario: 与注册并发（race）
- **WHEN** `Register` 与 `VendorSupported` 被多 goroutine 并发调用（`-race`）
- **THEN** SHALL 无数据竞态告警
