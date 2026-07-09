# snd-xml-codec — XML 编解码声明式化（P5-2）

## Why

`pkg/yang-runtime/client/netconf.go`（1452 行）为每个 YANG 模型手写 XML 三件套：字段逐行 `fmt.Sprintf` 的 build（vlan/ifm 各 150~240 行）、带 `xml:` tag 中间结构的 Parse、`marshalDeleteChange`/`marshalChange` 的 per-model type-switch、4 处 camelCase→kebab 字符串替换器。**加一个 YANG 模块 ≈ 手写 400 行胶水**，且 build/parse 字段不对称已造成字段级永久漂移（IFM 下发 ~35 字段、回读仅 10 字段）。ygot 生成物中通用编解码所需数据齐备（`path:`/`module:` struct tag、内嵌 `SchemaTree`、`ΛListKeyMap()`），这是 SND 声明式化路线（P5-1 #134 已拍板纯 Go 编译期、③注册表→①数据驱动终态）的第二步：把「设备特定 XML 编解码」从手写代码变为数据（schema + 描述符）驱动。

## What Changes

- 新增通用 schema 驱动 XML 编解码引擎 `pkg/yang-runtime/xmlcodec`：
  - **Encode**：reflect 遍历 ygot GoStruct，元素名取 `path:` tag、namespace 取 `SchemaTree` 的 `yang.Entry.Namespace()`，复刻既有「nil 指针不发、enum 零值(=UNSET)不发」语义；
  - **Decode**：token 扫描 + schema 驱动，回读直接填充 ygot GoStruct（退役手写中间结构），**字段覆盖扩展为与 encode 对称**（修字段级永久漂移债）;
  - **EncodeDelete**：经 `ΛListKeyMap()` 通用生成键式 `nc:operation="delete"` 条目（退役 per-model type-switch）。
- `driver.Descriptor` 扩展纯数据字段（root 元素名/schema 入口/GoStruct 工厂），vlan/ifm/system 描述符接线通用引擎；描述符注册机制不变（DR-01）。
- `client.marshalChange`/`marshalDeleteChange` 改查 driver 注册表分发，退役 `buildHuaweiVlanVlansXML`/`buildHuaweiIfmInterfacesXML`/`ParseHuaweiVlanVlansXML`/`ParseHuaweiIfmInterfacesXML`/`mapToHuaweiIfmInterfaces` 及全部 kebab 替换器；消灭 `pkg/yang-runtime/client` 对 `internal/generated/huawei` 的直接依赖（框架层不再 import 厂商生成物）。
- 顺带修复存量 bug：`buildHuaweiVlanVlansXML` 将 `<suppression>` 容器重复发送两遍（netconf.go:1387、netconf.go:1436）。
- **golden test 先行**：先冻结既有 builder 输出为规范化 golden（条目按 key 排序，规避 map 迭代序非确定），通用引擎对拍 golden 后才切换。
- 范围外（明确不动）：`buildOpenConfigInterfacesXML`（Stack A 邻接的 `controller/interfaces` 在用，保留 fallback）；`mergeConfig`/`config_delete` 的 API 层 type-switch（独立 change 顺延）；ygot 生成管线 sed 参数化（P5-3）。

## Capabilities

### New Capabilities
- `yang-xml-codec`: 通用 schema 驱动 NETCONF XML 编解码引擎——任意 ygot GoStruct 的 encode/decode/键式 delete 编码，数据来源仅 ygot 生成物（struct tag + SchemaTree + ΛListKeyMap），新增模块零 XML 代码。

### Modified Capabilities
- `device-driver-registry`: DR-01 描述符新增编解码数据字段（root/schema/工厂）；DR-03 编解码闭包由「每模块手写」收敛为「通用引擎 + 描述符数据」。
- `device-protocol`: DP-07 删除编码从「按 OldValue 模型类型（per-model switch）」改为「经驱动注册表 + ΛListKeyMap schema 驱动」，行为断言（键式条目、仅 key 叶、错误透出）不变。（DP-04 两阶段提交语义不变，写路径编码分发属实现细节不动需求。）

## Impact

- 代码：`backend/pkg/yang-runtime/xmlcodec`（新增）、`backend/pkg/yang-runtime/driver/registry.go`（描述符扩展）、`backend/pkg/yang-runtime/client/netconf.go`（三件套退役，预计净 -600 行）、`backend/internal/drivers/huawei.go`（描述符接线）、`backend/internal/controller/{ifm,vlan}/reconciler.go`（Parse 调用点改经描述符）。
- 测试：新增 B1 golden + xmlcodec 单测（含 -race）；存量 B1/B2/B3 全绿为硬门禁；B2 模拟网元收敛断言兜底 suppression 修复与 decode 全字段扩展两处「语义等价、字节不等价」的行为改善。
- 风险面：下发编码是对账主链路，编码语义偏差=全网漂移；golden 对拍 + 存量 B2 是防线。PR ≤1000 行（净增可控：引擎 +500、退役 -800）。
