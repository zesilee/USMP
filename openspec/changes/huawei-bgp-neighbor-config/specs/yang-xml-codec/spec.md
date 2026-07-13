## ADDED Requirements

### Requirement: XC-07 YANG empty 类型（presence-only）编解码

通用引擎 SHALL 支持 YANG `empty` 类型（ygot 生成为 `YANGEmpty`，即非指针 `bool` 字段）的编解码：这类叶无值、仅以「存在/不存在」表意。编码时 `YANGEmpty(true)` SHALL 产出自闭合元素 `<tag/>`（模块边界则 `<tag xmlns=…/>`，XC-06），`YANGEmpty(false)` SHALL 不发（同 nil 叶跳发语义）。解码时元素存在即置为 `true`（无值可解）。

**触发原因**：BGP 基础邻居 `peer/bfd-parameter/compatible` 是首个走此路径的驱动字段——VLAN/IFM/BGP-public/network-instance 的配置面无 `empty` 类型叶，故既有引擎的 `encodeField`/`decodeField` 未覆盖非指针 `bool`（编码报 `unsupported field form bool`）。这是「通用引擎其实是 XX 中心」缺口序列的又一处，由 peer 深层子树往返真值测试拦下。既有类型（标量指针叶/枚举/嵌套容器/list/leaf-list）行为不变。

#### Scenario: empty 类型 true 编码为自闭合元素
- **WHEN** 编码 `YANGEmpty(true)` 的字段
- **THEN** 输出 SHALL 为自闭合 `<tag/>`（无值）

#### Scenario: empty 类型 false 跳发
- **WHEN** 编码 `YANGEmpty(false)` 的字段
- **THEN** SHALL NOT 发射该元素（同 nil 叶跳发语义）

#### Scenario: empty 类型解码为 present
- **WHEN** 解码含 `<tag/>` 的报文到 `YANGEmpty` 字段
- **THEN** 该字段 SHALL 置为 `true`（元素存在即 present）

#### Scenario: empty 类型往返恒等
- **WHEN** 对含 `YANGEmpty(true)` 字段的 GoStruct 先编码再解码
- **THEN** 该字段 SHALL 恒等为 `true`
