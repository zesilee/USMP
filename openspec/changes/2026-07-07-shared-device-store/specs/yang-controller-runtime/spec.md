## MODIFIED Requirements

### Requirement: Reconciler 从 DeviceStore 解析建连参数

各 Reconciler 的 `deviceClient` SHALL 用 `req.DeviceID` 查共享 `DeviceStore` 取完整连接信息建连，SHALL NOT 再从 DeviceID 字符串解析 `user:pass@ip:port`（该形式仅测试用、掩盖了生产缺凭据的缺陷，SHALL 删除）。库未命中 SHALL 以 `{IP:id, Protocol:AUTO}` 兜底一次并记 warning（R08，不崩），SHALL NOT 再硬编码 `admin/admin` 凭据（删除 #100 兜底）。

#### Scenario: 已注册设备对账建连带凭据
- **WHEN** 以纯 DeviceID 触发对账、设备已在 DeviceStore 注册
- **THEN** 建 NETCONF 连接 SHALL 携带库中凭据，SSH SHALL 以 password 认证成功（不再退化为 `none`）

#### Scenario: 未注册设备降级不崩
- **WHEN** 对账一个未在库中的 DeviceID
- **THEN** SHALL 以 AUTO 兜底建连并记 warning，认证失败 SHALL 返回明确错误（R08），SHALL NOT panic

### Requirement: 周期源以 DeviceStore 的设备列表驱动漂移检测

周期 EventSource SHALL 从 `DeviceStore.List()` 取设备列表逐个入队对账（每 tick 动态取，新增设备无需重启）。SHALL NOT 再以 `nil` 设备列表空转（回归 #101：nil→零 enqueue→无持续 out-of-band 漂移检测）。

#### Scenario: 周期发出库中设备的对账事件
- **WHEN** 周期 tick 触发、库中有 N 个设备
- **THEN** SHALL 为每个设备就配置路径入队一个对账事件

#### Scenario: 空库不空转报错
- **WHEN** 库中无设备
- **THEN** SHALL 不入队任何事件，SHALL NOT panic
