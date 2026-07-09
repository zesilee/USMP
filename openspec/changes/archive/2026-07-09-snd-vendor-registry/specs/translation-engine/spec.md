# translation-engine delta — snd-vendor-registry

> 主 spec 为遗留「反向还原」自由格式；sync 时已借机迁移为 OpenSpec 标准格式。原 delta 编号 TE-01/TE-02 与主 spec 既有编号冲突，sync 时重编号为 TE-05/TE-06（本文件同步修正）。

## ADDED Requirements

### Requirement: TE-05 编译期驱动自注册

厂商 Translator 实现 SHALL 通过其实现文件的 `init()` 调用 `RegisterTranslator` 完成编译期自注册；注册表 SHALL NOT 在 `GetTranslator` 内硬编码注册任何厂商。`RegisterTranslator` 并发调用 SHALL 无数据竞态（R09）。`GetTranslator` 对未注册厂商 SHALL 返回明确错误而非 panic（R08）。

#### Scenario: 编译期注册即可得
- **WHEN** 进程启动后调用 `GetTranslator(VendorHuawei)`
- **THEN** SHALL 返回 huawei Translator（由 init 自注册），无需任何显式注册调用

#### Scenario: 未注册厂商（负路径）
- **WHEN** `GetTranslator(VendorCisco)`（枚举存在但无实现注册）
- **THEN** SHALL 返回明确错误，SHALL NOT panic

### Requirement: TE-06 按设备厂商解析驱动

意图翻译调用方（crdsource）SHALL 按目标设备在 DeviceStore 中的 `Vendor` 解析 Translator，SHALL NOT 硬编码厂商常量；设备未注册或 `Vendor` 为空时 SHALL 降级为 `huawei`（R08，与存量行为等价）。

#### Scenario: 按设备厂商取驱动
- **WHEN** DeviceStore 中设备 X 的 `Vendor="huawei"`，对 X 的意图翻译触发
- **THEN** SHALL 使用 huawei Translator（等价于既有行为）

#### Scenario: 设备未注册降级（边界）
- **WHEN** 目标设备不在 DeviceStore
- **THEN** SHALL 按 `huawei` 缺省解析并继续，SHALL NOT 失败
