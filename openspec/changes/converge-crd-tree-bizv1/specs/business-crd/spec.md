## MODIFIED Requirements

### Requirement: 统一 CRD 树到 api/biz/v1（退役 api/v1）

business CRD 类型 SHALL 统一到 `api/biz/v1`（+`api/core/v1` 的 NativeDeviceConfig）；退役 `api/v1`，消除 `biz.usmp.io/v1` group 双注册与 schema 不兼容（迁移债 D1）。

#### Scenario: 单一 CRD 树被引用
- **WHEN** translator/crdsource 引用 business CRD 类型
- **THEN** SHALL 仅 `api/biz/v1`；`api/v1` 已删除、无引用

#### Scenario: NativeDeviceConfig 归位 core/v1
- **WHEN** 引用 NativeDeviceConfig 类型
- **THEN** SHALL 为 `api/core/v1`（`api/v1` 内的同名类型为死代码，随 api/v1 删除）
