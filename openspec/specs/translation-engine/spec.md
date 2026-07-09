# translation-engine — 意图翻译引擎

## Purpose

把业务意图 Spec（当前为 `bizv1.*` CRD Spec）翻译为厂商 ygot YANG 结构，按厂商注册表派发，供 Stack B crdsource 意图源（与遗留 Actor 路径共用）在下发前调用。仅 Huawei 有注册实现；Cisco/H3C/Juniper 为枚举占位。已知契约缺口：Route 翻译返回裸 map（未完成）、System 翻译不支持、接口以 `interface{}` 传参（R04 张力，输入契约解耦属 P5 声明式化后续）。

> 2026-07-09 由「反向还原」自由格式迁移为 OpenSpec 标准格式（spec-format-standard），行为描述忠实 as-built；TE-05/TE-06 由 change `snd-vendor-registry` 引入（其 delta 中编号为 TE-01/TE-02，因与本文件既有编号冲突在 sync 时重编号）。

## Requirements

### Requirement: TE-01 厂商派发

`TranslateConfig(vendor, configType, spec)` SHALL 经 `GetTranslator(vendor)` 取厂商翻译器后按 configType 分派到 TranslateVlan/Interface/Route/System；未注册厂商 SHALL 返回「未找到厂商」明确错误（R08，不 panic）。

#### Scenario: 按厂商与类型分派
- **WHEN** `TranslateConfig(VendorHuawei, ConfigTypeVlan, spec)`
- **THEN** SHALL 派发到 huawei 翻译器的 `TranslateVlan`

#### Scenario: 未注册厂商
- **WHEN** `TranslateConfig` 传入无注册实现的厂商
- **THEN** SHALL 返回「未找到厂商」错误，SHALL NOT panic

### Requirement: TE-02 VLAN 翻译

`HuaweiTranslator.TranslateVlan` 对 `bizv1.BusinessVlanSpec` SHALL 产出 `*huawei.HuaweiVlan_Vlan_Vlans`：以 uint16 VLAN ID 为键，admin/type/mac-learning 映射为数字枚举。

#### Scenario: VLAN Spec 翻译
- **WHEN** 传入合法 `BusinessVlanSpec`
- **THEN** SHALL 产出以 VLAN ID 为键的 huawei ygot 结构

### Requirement: TE-03 Interface 翻译

`HuaweiTranslator.TranslateInterface` 对 `bizv1.BusinessInterfaceSpec` SHALL 产出 `*huawei.HuaweiIfm_Ifm_Interfaces`，以接口名为键。

#### Scenario: Interface Spec 翻译
- **WHEN** 传入合法 `BusinessInterfaceSpec`
- **THEN** SHALL 产出以接口名为键的 huawei ifm ygot 结构

### Requirement: TE-04 校验

`Validate(configType, spec)` SHALL 校验输入（CIDR/IP 等）；不合法 SHALL 返回 `TranslateError`，不支持项 SHALL 返回 `Unsupported`。

#### Scenario: 非法输入
- **WHEN** spec 含非法 CIDR/IP
- **THEN** SHALL 返回 `TranslateError`，SHALL NOT 下发

### Requirement: TE-05 编译期驱动自注册

厂商 Translator 实现 SHALL 通过其实现文件的 `init()` 调用 `RegisterTranslator` 完成编译期自注册；注册表 SHALL NOT 在 `GetTranslator` 内硬编码注册任何厂商。`RegisterTranslator` 并发调用 SHALL 无数据竞态（R09）。`GetTranslator` 对未注册厂商 SHALL 返回明确错误而非 panic（R08）。

#### Scenario: 编译期注册即可得
- **WHEN** 进程启动后调用 `GetTranslator(VendorHuawei)`
- **THEN** SHALL 返回 huawei Translator（由 init 自注册），无需任何显式注册调用

#### Scenario: 未注册厂商（负路径）
- **WHEN** `GetTranslator(VendorCisco)`（枚举存在但无实现注册）
- **THEN** SHALL 返回明确错误，SHALL NOT panic

### Requirement: TE-06 按设备厂商解析驱动

意图翻译调用方（crdsource）SHALL 按目标设备在 DeviceStore 中的 `Vendor` 解析 Translator，SHALL NOT 硬编码厂商常量；设备未注册或 `Vendor` 为空时 SHALL 降级为 `huawei`（R08，与存量行为等价）。厂商标签匹配 SHALL 大小写无关（`"huawei"` → 枚举 `Huawei`）。

#### Scenario: 按设备厂商取驱动
- **WHEN** DeviceStore 中设备 X 的 `Vendor="huawei"`，对 X 的意图翻译触发
- **THEN** SHALL 使用 huawei Translator（等价于既有行为）

#### Scenario: 设备未注册降级（边界）
- **WHEN** 目标设备不在 DeviceStore
- **THEN** SHALL 按 `huawei` 缺省解析并继续，SHALL NOT 失败

#### Scenario: 无驱动厂商错误透出（负路径）
- **WHEN** 设备 `Vendor` 为无注册驱动的标签（如 `nokia`）
- **THEN** 翻译 SHALL 失败并在错误中含该厂商名
