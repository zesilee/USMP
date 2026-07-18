# yang-xml-codec — delta（retire-openconfig-models）

## MODIFIED Requirements

### Requirement: XC-04 框架层与厂商生成物解耦

`pkg/yang-runtime/client` 生产代码 SHALL NOT import 任何 `internal/generated/*` 厂商生成包；变更编码/删除编码 SHALL 经驱动描述符注册表按 GoStruct 类型解析编解码数据后调用通用引擎。注册表未命中 SHALL 降级为通用 `xml.Marshal` 兜底链（含 map 反射兜底），SHALL NOT 报「未注册」硬错误（R08）；openconfig 类型特判分支 SHALL NOT 存在。注册表消费方（reconciler、独立测试二进制）SHALL 以空白导入 `internal/drivers` 保证注册可达。

#### Scenario: 注册模块经通用引擎编码
- **WHEN** `Set` 的 Change.NewValue 为已注册描述符的 GoStruct 类型（含其内层 list map 形态）
- **THEN** SHALL 经注册表解析并由通用引擎编码，输出与改造前等价

#### Scenario: 未注册类型降级（负路径）
- **WHEN** Change.NewValue 为无描述符覆盖的类型
- **THEN** SHALL 直接走通用 `xml.Marshal` 兜底链，SHALL NOT 报「未注册」硬错误，SHALL NOT 经任何 openconfig 类型特判
