# yang-xml-codec — 通用 schema 驱动 NETCONF XML 编解码引擎

## Purpose

通用 NETCONF XML 编解码引擎（`pkg/yang-runtime/xmlcodec`）：任意 ygot GoStruct 的 encode / decode / 键式 delete 编码，数据来源仅 ygot 生成物（`path:` struct tag、内嵌 SchemaTree、`ΛListKeyMap()`）与驱动描述符登记的编解码数据（模块 namespace 为显式数据——内嵌 gzip schema 的 `Entry.Namespace()` 实测为空，无法派生）。新增 YANG 模块零 XML 代码，是 SND 声明式化（P5）的编解码承载层。由 change `snd-xml-codec` 引入（PR #136/#137/#138，2026-07-09）。

## Requirements

### Requirement: XC-01 通用 XML 编码（schema/tag 数据驱动）

系统 SHALL 提供通用 NETCONF XML 编码器（`pkg/yang-runtime/xmlcodec`）：对任意 ygot GoStruct，元素名 SHALL 取自 ygot 生成的 `path:` struct tag。list 容器 SHALL 嵌套在其 YANG 模块顶层容器（如 `<ifm>`/`<vlan>`，祖先容器名从 schema `Entry.Parent` 链派生、止于合成 fake root）之内；模块 namespace SHALL 声明在最外层容器上、内层容器继承（对齐真机与模拟器种子 `DemoSeedConfig` 的嵌套结构——扁平根会在设备数据树里匹配不到既有嵌套条目）。编码 SHALL NOT 依赖任何 per-model 手写字段序列化或元素名字符串替换。编码 SHALL 复刻既有跳发语义：nil 指针叶不发、enum 零值（UNSET）不发、nil 嵌套容器不发、空 list 发自闭合 list 容器（仍包裹在模块容器内）；list 条目 key 叶 SHALL 为首元素（key 叶为 nil 时以 map key 回退）；文本内容 SHALL 做 XML 转义。编码 SHALL NOT 按 schema config-false 过滤（华为模型将在发字段标 config-false，过滤破坏行为等价）。遇到不支持的字段形态 SHALL 返回明确错误（R08，不静默丢字段、不 panic）。

#### Scenario: 与既有手写 builder 输出等价（golden 对拍）
- **WHEN** 对 VLAN 全字段（含嵌套 member-ports/suppression/unkown-unicast-discard）与 IFM 全字段（含 damp/error-down/control-flap）fixture 编码
- **THEN** 规范化（同级元素全排序+相同同级去重）后输出 SHALL 与冻结的既有 builder golden 逐字节一致

#### Scenario: 跳发语义
- **WHEN** GoStruct 中某叶为 nil 指针、某 enum 为零值、某嵌套容器为 nil
- **THEN** 输出 SHALL NOT 含对应元素；空 list SHALL 编码为包裹在模块容器内的自闭合 list 容器（`<ifm xmlns=NS><interfaces/></ifm>`）

#### Scenario: 嵌套于模块顶层容器（与真机数据树对齐）
- **WHEN** 对 IFM 接口 / VLAN 条目编码
- **THEN** list 容器 SHALL 包裹在模块顶层容器内、namespace 声明在最外层（`<ifm xmlns=NS><interfaces><interface>…` / `<vlan xmlns=NS><vlans><vlan>…`），使针对设备既有嵌套条目的 edit-config（含删除）能在设备数据树里命中——扁平根元素 SHALL NOT 发送

#### Scenario: 特殊字符转义
- **WHEN** 字符串叶含 `<`、`&`、引号
- **THEN** 输出 SHALL 为合法 XML（实体转义），设备端解析不损坏

#### Scenario: 不支持形态（负路径）
- **WHEN** 编码遇到引擎未支持的字段形态
- **THEN** SHALL 返回明确错误，SHALL NOT 静默跳过该字段或 panic

### Requirement: XC-02 通用 XML 解码（回读全字段对称）

系统 SHALL 提供通用 NETCONF XML 解码器：对 get-config 回读原文（无论包裹在 `<rpc-reply>`/`<data>`、模块顶层容器或裸容器、无论 namespace 前缀），SHALL 在 list 容器（root，如 `<vlans>`/`<interfaces>`）之内定位 list 条目元素并填充 ygot GoStruct（含 list map、嵌套 list、enum、指针叶）。锚定 list 容器 SHALL 消解「模块顶层容器名与条目名同名」的歧义（华为 `<vlan><vlans><vlan>` 外层模块容器与条目同名 `vlan`，裸扫条目名会误把外层容器当条目）。解码字段覆盖 SHALL 为编码字段的超集（同一份 tag 数据驱动，SHALL NOT 出现「可下发但回读丢失」的字段），呈现叶（config-false，如 class/parent-name）SHALL 照常透出。条目 key SHALL 经 `ΛListKeyMap()` 获取，key 叶缺失时 SHALL 合成 key 保留条目（宽容语义）。空输入或无条目 SHALL 返回非 nil 空容器；非法 XML 或非数值 enum 文本 SHALL 返回明确错误。

#### Scenario: 编解码往返恒等
- **WHEN** 对全字段 fixture 先 XC-01 编码再 XC-02 解码
- **THEN** 解码结果 SHALL 与原 GoStruct 语义相等——消除字段级永久漂移（IFM 曾下发 35 字段仅回读 10 字段）

#### Scenario: 包裹与前缀鲁棒
- **WHEN** 回读原文分别为 `<rpc-reply><data>…` 包裹、裸容器、带 namespace 前缀三种形态
- **THEN** SHALL 均正确解析出同一结果

#### Scenario: 空回读（边界）
- **WHEN** 回读为空字节或不含目标条目
- **THEN** SHALL 返回非 nil 空容器（map 已初始化），SHALL NOT 返回 nil 或报错

#### Scenario: 全字段端到端收敛（B2）
- **WHEN** 全字段配置经模拟网元下发→回读→二次对账
- **THEN** 第二轮 SHALL 收敛（Changes==0），SHALL NOT 因回读字段缺失产生永久漂移

### Requirement: XC-03 键式删除编码（ΛListKeyMap 驱动）

系统 SHALL 提供通用删除编码器：对 list 型 GoStruct 的每个条目，主键 SHALL 经 ygot `ΛListKeyMap()` 获取（key 叶为 nil 时以 schema Key 语句 + map key 回退），生成嵌套结构——模块顶层容器（带模块 namespace）> list 容器 > 条目元素携带 NETCONF base namespace 的 `operation="delete"` 属性 + 仅 key 叶（key 为首个子元素）。SHALL NOT 按模型类型 switch 硬编码 key 名。空条目集或无 key 信息 SHALL 返回明确错误（R08，不发送裸 delete 元素）。多 key list SHALL 返回明确不支持错误。

#### Scenario: 键式删除嵌套于模块顶层容器
- **WHEN** 对仅含键的 VLAN 条目集 / IFM 接口集做删除编码
- **THEN** 输出 SHALL 为 `<vlan xmlns=NS><vlans><vlan nc:operation="delete"…><id>10</id></vlan></vlans></vlan>` / `<ifm xmlns=NS><interfaces><interface …><name>…</name></interface></interfaces></ifm>`，条目仅携带 key 叶，DP-07 断言全部保持——嵌套结构使设备既有条目的删除能命中（修复「内置接口删不掉」）

#### Scenario: 空目标（负路径）
- **WHEN** 条目集为空
- **THEN** SHALL 返回明确编码错误，SHALL NOT 发送空删除

### Requirement: XC-04 框架层与厂商生成物解耦

`pkg/yang-runtime/client` 生产代码 SHALL NOT import 任何 `internal/generated/*` 厂商生成包获取编解码逻辑；变更编码/删除编码 SHALL 经驱动描述符注册表按 GoStruct 类型解析编解码数据后调用通用引擎。注册表未命中 SHALL 保持既有降级链（openconfig 遗留分支、`xml.Marshal` 兜底）不变（R08）。注册表消费方（reconciler、独立测试二进制）SHALL 以空白导入 `internal/drivers` 保证注册可达。

#### Scenario: 注册模块经通用引擎编码
- **WHEN** `Set` 的 Change.NewValue 为已注册描述符的 GoStruct 类型（含其内层 list map 形态）
- **THEN** SHALL 经注册表解析并由通用引擎编码，输出与改造前等价

#### Scenario: 未注册类型降级（负路径）
- **WHEN** Change.NewValue 为无描述符覆盖的类型
- **THEN** SHALL 走既有 fallback 链，行为与改造前一致，SHALL NOT 报「未注册」硬错误


### Requirement: XC-05 plain-container 根模块编解码（容器根，非 list 根）

通用引擎 SHALL 支持 **plain-container 根**模块——根节点是仅含标量、子容器（可含更深层嵌套 list）而**无根级 YANG list** 的容器，如 `/bgp:bgp`（`HuaweiBgp_Bgp` 持有 base-process/global 等子容器，根无 list map 字段）。引擎 SHALL 按 GoStruct 是否含根级 YANG-list map 字段自动择 list 模式（vlan/ifm）或 container 模式（bgp），二者 SHALL 复用同一字段级机制（叶 / 嵌套容器 / 更深嵌套 list / leaf-list 的编解码），SHALL NOT 为容器根另写平行的字段序列化逻辑（避免与 list 路径漂移）。

编码 SHALL 产出 `<root xmlns=NS>{fields}</root>`（root 取自 schema 根容器名，namespace 声明在根上；根容器有祖先时按 XC-01 同规则外包并把 namespace 移至最外层），全字段为空时 SHALL 产出自闭合 `<root xmlns=NS/>`；跳发语义（nil 叶/UNSET enum/nil 子容器不发）SHALL 与 list 路径一致。解码 SHALL 按 local 名穿透 `<rpc-reply>`/`<data>` 包裹与 namespace 前缀定位根容器元素、填充根 GoStruct，根元素缺失或空输入 SHALL 返回未填充的容器（宽容，不报错）。container 模式引入 SHALL NOT 改变既有 list 根（vlan/ifm）行为（XC-01/XC-02 golden 与往返恒等保持）。

#### Scenario: 容器根编码（标量 + 多层嵌套子容器）
- **WHEN** 对 `/bgp:bgp`（base-process 含 enable/as/as-path-limit 及嵌套 graceful-restart 子容器）编码
- **THEN** 输出 SHALL 为 `<bgp xmlns="urn:huawei:yang:huawei-bgp"><base-process>…<graceful-restart>…</graceful-restart></base-process></bgp>`，标量真值正确、无 list 元素包裹形态

#### Scenario: 空容器根自闭合
- **WHEN** 对全字段为 nil 的容器根编码
- **THEN** 输出 SHALL 为自闭合 `<bgp xmlns="urn:huawei:yang:huawei-bgp"/>`

#### Scenario: 容器根编解码往返恒等
- **WHEN** 对容器根 GoStruct 先编码再解码
- **THEN** 解码结果 SHALL 与原 GoStruct 语义相等（标量与嵌套子容器字段一致）

#### Scenario: 容器根回读包裹/前缀鲁棒
- **WHEN** get-config 回读原文为 `<rpc-reply><data><bgp:bgp xmlns:bgp=…>…</bgp:bgp></data></rpc-reply>`（含 namespace 前缀）
- **THEN** SHALL 按 local 名穿透包裹与前缀正确解码出根字段

#### Scenario: 择模式不回归 list 根
- **WHEN** 引入 container 模式后对 VLAN/IFM（list 根）编解码
- **THEN** 输出 SHALL 与既有 golden 逐字节一致、往返恒等保持（list 模式零回归）

### Requirement: XC-06 per-node namespace（module tag 驱动，支持 augment 跨模块树）

通用引擎 SHALL 支持按生成物字段的 `module` struct tag 在**模块边界**发射 per-node XML namespace，使 augment 跨模块树（如 `huawei-bgp` peers/afs augment 到 `huawei-network-instance` 根下）的编码携带每节点正确的 namespace。`Spec` SHALL 提供可选的 module→namespace 解析表（`Namespaces map[string]string`）；缺省（nil/空）时 SHALL 保持既有单根 namespace 行为。

编码时引擎 SHALL 跟踪父节点生效 namespace（根为 `Spec.Namespace`）：对每个字段取其 `module` tag 解析 namespace，当解析到的 namespace 非空且**不等于**父生效 namespace 时，SHALL 在该元素上声明 `xmlns="…"` 并以之为其子树的生效 namespace（子节点默认继承直到下一个模块边界）；否则 SHALL NOT 发射多余 xmlns、沿用父 namespace。此机制 SHALL 对 list 条目与嵌套子容器一致适用，SHALL NOT 为 per-node namespace 另写平行序列化逻辑（复用 XC-01/XC-05 字段级机制）。

per-node namespace 引入 SHALL NOT 改变**单模块树**（vlan/ifm/bgp/system/network-instance——所有字段 module 同根）的编码输出：SHALL 与既有 XC-01/XC-02/XC-05 golden 逐字节一致、往返恒等保持（零回归为硬门禁）。解码 SHALL 无需改动（既有按 local 名穿透 namespace 前缀，跨 namespace 报文天然可解）。

> 正确性说明：netconfsim 与解码均 namespace-宽容（按 local 名存取/定位），故 per-node namespace 的正确性 SHALL 由**编码输出的 namespace 真值断言**保证，而非往返或集成（后者证不出 namespace 错）。

#### Scenario: augment 跨模块树在模块边界发正确 namespace
- **WHEN** 对 `network-instance`（根 namespace `urn:huawei:yang:huawei-network-instance`）下含 `huawei-bgp` augment 子树（`bgp/base-process`）的 GoStruct 编码，且 Spec.Namespaces 登记了 `huawei-bgp → urn:huawei:yang:huawei-bgp`
- **THEN** 输出中 `<bgp>` 元素 SHALL 携带 `xmlns="urn:huawei:yang:huawei-bgp"`，其下 base-process 等子节点继承之而不重复发；同级的 `<name>`/`<description>`（huawei-network-instance）SHALL NOT 另发 xmlns（继承 ni 根）

#### Scenario: 单模块树零新增 xmlns（golden 逐字节不变）
- **WHEN** 引入 per-node namespace 后对 VLAN/IFM/BGP（单模块树，Namespaces 未登记差异模块或值同根）编码
- **THEN** 输出 SHALL 与既有 golden 逐字节一致——除根（及外包祖先）外无任何新增 xmlns 发射，往返恒等保持

#### Scenario: 缺省 Namespaces 保持现状
- **WHEN** Spec.Namespaces 为 nil/空对某模块编码
- **THEN** SHALL 退回既有单根 namespace 行为，输出不变

#### Scenario: 混合 namespace 报文解码鲁棒
- **WHEN** 解码含混合 namespace 前缀的回读报文（ni 根 + bgp 子树各自 namespace 前缀）
- **THEN** SHALL 按 local 名穿透前缀正确填充 GoStruct（解码不因 per-node namespace 而改变）

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
