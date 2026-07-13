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
