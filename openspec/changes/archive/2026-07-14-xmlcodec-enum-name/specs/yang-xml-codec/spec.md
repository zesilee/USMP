## ADDED Requirements

### Requirement: XC-08 YANG enumeration 值域名编解码

通用 XML 编解码引擎 SHALL 将 YANG `enumeration` 类型的 leaf 与 list key 编码为其 **YANG 值域名**（如 `<admin-status>up</admin-status>`），SHALL NOT 编码为底层整数值（如 `<admin-status>2</admin-status>`）——真实设备按值域名校验，整数会被拒。值域名 SHALL 经 ygot `ΛMap`（`ygot.EnumName`）由枚举 int 值映射取得。`UNSET`（0）SHALL 跳发（不出现在 edit-config）。

引擎 SHALL 按值域名解码：接收的枚举文本 SHALL 经 `ΛMap` 反查值域名→int 还原为 ygot 枚举常量。为不破坏历史报文/数据，解码 SHALL 兼容整数形态（值域名反查未命中时回退按整数解析）。

对 `ΛMap` 中**无对应值域名的枚举整数**（仅合成/异常值命中——真机有效枚举值必在 `ΛMap`），编码 SHALL 退回输出该整数、SHALL NOT 报错或中断整树编码（R08）。

本要求 SHALL NOT 改变 XC-01~07 的既有行为（list/容器根编解码、per-node namespace、YANG empty 类型等）；非枚举 leaf 编解码不变。

#### Scenario: 枚举 leaf 编码为值域名
- **WHEN** 编码一个已设合法枚举值的 leaf（如 `admin-status = up`）
- **THEN** 输出 XML SHALL 为 `<admin-status>up</admin-status>`（值域名），SHALL NOT 为整数

#### Scenario: 枚举 leaf 按值域名解码
- **WHEN** 解码 `<admin-status>up</admin-status>`
- **THEN** 目标 ygot 枚举字段 SHALL 还原为对应常量（`AdminStatus_up`）

#### Scenario: 解码兼容历史整数形态
- **WHEN** 解码 `<admin-status>2</admin-status>`（旧整数报文）
- **THEN** 引擎 SHALL 回退按整数解析、SHALL 还原为等价枚举值（不破坏历史数据）

#### Scenario: UNSET 枚举跳发
- **WHEN** 编码一个未设置（UNSET=0）的枚举 leaf
- **THEN** 该 leaf SHALL NOT 出现在输出 XML

#### Scenario: 未映射枚举值退回整数不中断（R08）
- **WHEN** 编码一个 `ΛMap` 中无对应名的枚举整数（合成/异常值）
- **THEN** 引擎 SHALL 输出该整数、SHALL NOT 报错、SHALL NOT 中断整树编码

#### Scenario: 真正非法枚举文本解码报错
- **WHEN** 解码一个既非合法值域名、又非整数的枚举文本
- **THEN** 引擎 SHALL 返回命名该 leaf 的明确错误（R08，不静默）
