## REMOVED Requirements

### Requirement: TE-01 厂商派发
**Reason**: 唯一调用方 `internal/crdsource`（旧 BusinessVlan/BusinessInterface CRD 桥接）随本 change 删除，`TranslateConfig` 派发链零消费。
**Migration**: 意图→原生配置的展开由 `internal/intent`（BusinessVlanService 控制器）直接产出 huawei-vlan/huawei-ifm 原生配置，不经翻译引擎。

### Requirement: TE-02 VLAN 翻译
**Reason**: `bizv1.BusinessVlanSpec` 类型随 `api/biz/v1` 包删除，翻译目标不复存在。
**Migration**: 跨设备 VLAN 意图见 [[business-vlan-service]] BVS-01（意图 YANG 模型展开为原生配置）。

### Requirement: TE-03 Interface 翻译
**Reason**: 同 TE-02，`bizv1.BusinessInterfaceSpec` 随 `api/biz/v1` 包删除。
**Migration**: 端口归属意图并入 BusinessVlanService 的 devices/access-ports/trunk-ports 模型（[[business-vlan-service]]）。

### Requirement: TE-04 校验
**Reason**: 翻译引擎整包退役，`Validate` 无调用方；意图校验已由 apiserver OpenAPI + 控制器约束引擎分层承担（[[business-intent-crd]] BIC 系列）。
**Migration**: 无——新链路校验契约见 business-intent-crd。

### Requirement: TE-05 编译期驱动自注册
**Reason**: translator 注册表随包删除；「编译期驱动自注册」契约在 device-driver-registry DR-01 中已有等价且在用的承载（`pkg/yang-runtime/driver`，init() 注册、并发安全、未命中降级）。
**Migration**: 驱动注册一律走 [[device-driver-registry]] DR-01。

### Requirement: TE-06 按设备厂商解析驱动
**Reason**: 调用方 crdsource 删除；「厂商是否受支持」的查询职责移交 driver 注册表。
**Migration**: devices-api BR-04 厂商门禁改由 [[device-driver-registry]] DR-04 `VendorSupported`（大小写无关）承载；按设备 Vendor 解析驱动的在用契约为 DR-02/DR-03 查表路由。
