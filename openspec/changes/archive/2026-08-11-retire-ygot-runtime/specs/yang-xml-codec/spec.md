# yang-xml-codec — delta (retire-ygot-runtime)

## MODIFIED Requirements

### Requirement: XC-01 通用 XML 编码（schema/tag 数据驱动）

系统 SHALL 提供通用 NETCONF XML 编码器（`pkg/yang-runtime/xmlcodec`）：对任意自研 Object（yang-native-runtime YN-01，替代 ygot GoStruct），元素名 SHALL 取自生成器产出的 `path:` struct tag。list 容器 SHALL 嵌套在其 YANG 模块顶层容器（如 `<ifm>`/`<vlan>`，祖先容器名从 Schema IR 的父链派生、止于合成 fake root）之内；模块 namespace SHALL 声明在最外层容器上、内层容器继承（对齐真机与模拟器种子 `DemoSeedConfig` 的嵌套结构——扁平根会在设备数据树里匹配不到既有嵌套条目）。编码 SHALL NOT 依赖任何 per-model 手写字段序列化或元素名字符串替换。编码 SHALL 复刻既有跳发语义：nil 指针叶不发、enum 零值（UNSET）不发、nil 嵌套容器不发、空 list 发自闭合 list 容器（仍包裹在模块容器内）；list 条目 key 叶 SHALL 为首元素（key 叶为 nil 时以 map key 回退）；文本内容 SHALL 做 XML 转义。编码 SHALL NOT 按 schema config-false 过滤（华为模型将在发字段标 config-false，过滤破坏行为等价）。遇到不支持的字段形态 SHALL 返回明确错误（R08，不静默丢字段、不 panic）。类型体系置换（ygot→Object/Schema IR）SHALL NOT 改变任何编码输出（既有 golden 逐字节保持，YN-06 XML 通道对拍）。

#### Scenario: 与既有手写 builder 输出等价（golden 对拍）
- **WHEN** 对 VLAN 全字段（含嵌套 member-ports/suppression/unkown-unicast-discard）与 IFM 全字段（含 damp/error-down/control-flap）fixture 编码
- **THEN** 规范化（同级元素全排序+相同同级去重）后输出 SHALL 与冻结的既有 builder golden 逐字节一致

#### Scenario: 跳发语义
- **WHEN** Object 中某叶为 nil 指针、某 enum 为零值、某嵌套容器为 nil
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

系统 SHALL 提供通用 NETCONF XML 解码器：对 get-config 回读原文（无论包裹在 `<rpc-reply>`/`<data>`、模块顶层容器或裸容器、无论 namespace 前缀），SHALL 在 list 容器（root，如 `<vlans>`/`<interfaces>`）之内定位 list 条目元素并填充自研 Object（含 list map、嵌套 list、enum、指针叶）。锚定 list 容器 SHALL 消解「模块顶层容器名与条目名同名」的歧义（华为 `<vlan><vlans><vlan>` 外层模块容器与条目同名 `vlan`，裸扫条目名会误把外层容器当条目）。解码字段覆盖 SHALL 为编码字段的超集（同一份 tag 数据驱动，SHALL NOT 出现「可下发但回读丢失」的字段），呈现叶（config-false，如 class/parent-name）SHALL 照常透出。

条目 key SHALL 经 `KeyedObject` key 元数据接口（YN-01，等价既有 `ΛListKeyMap()`）获取；**单键列表**以该键值为 map key，**多键（复合键）列表**（map key 为生成的复合 key struct，如 devm `physical-entity` 的 class+position+serial-number）SHALL 按 key struct 字段的 `path:` tag 从 key 元数据填充复合 key struct 作为 map key，SHALL NOT 因键数量 > 1 而报错或丢弃整表。此机制对根级列表与嵌套列表 SHALL 一致适用。key 叶缺失时 SHALL 合成 key 保留条目（宽容语义）：单键列表合成标量 key；多键列表 SHALL 从条目自身键字段复制可得值构造 key struct（缺失字段为零值），条目不丢弃。空输入或无条目 SHALL 返回非 nil 空容器；非法 XML 或非数值 enum 文本 SHALL 返回明确错误。

> 范围注记：键式删除编码（XC-03）的「多 key list SHALL 返回明确不支持错误」契约**保持不变**——多键列表绝大多数为 config-false 状态/统计表，删除支持由真实需求驱动另行变更。

#### Scenario: 编解码往返恒等
- **WHEN** 对全字段 fixture 先 XC-01 编码再 XC-02 解码
- **THEN** 解码结果 SHALL 与原 Object 语义相等——消除字段级永久漂移（IFM 曾下发 35 字段仅回读 10 字段）

#### Scenario: 包裹与前缀鲁棒
- **WHEN** 回读原文分别为 `<rpc-reply><data>…` 包裹、裸容器、带 namespace 前缀三种形态
- **THEN** SHALL 均正确解析出同一结果

#### Scenario: 空回读（边界）
- **WHEN** 回读为空字节或不含目标条目
- **THEN** SHALL 返回非 nil 空容器（map 已初始化），SHALL NOT 返回 nil 或报错

#### Scenario: 全字段端到端收敛（B2）
- **WHEN** 全字段配置经模拟网元下发→回读→二次对账
- **THEN** 第二轮 SHALL 收敛（Changes==0），SHALL NOT 因回读字段缺失产生永久漂移

#### Scenario: 多键列表回读解码（回归：devm/fib 空表）
- **WHEN** 解码含多键列表条目的回读原文（如 devm `physical-entitys/physical-entity`（key: class+position+serial-number）、fib `route-statistics` 下多键统计条目）
- **THEN** 每个条目 SHALL 以按 `path:` tag 填充的复合 key struct 入 map，全部键叶与非键字段正确还原，SHALL NOT 报 `multi-key lists unsupported`、SHALL NOT 整树降级

#### Scenario: 嵌套多键列表解码
- **WHEN** 解码的条目内含嵌套多键 list（`decodeField` Map 分支）
- **THEN** 嵌套条目 SHALL 与根级列表同机制以复合 key struct 入 map，字段正确还原

#### Scenario: 多键列表键叶缺失（宽容语义）
- **WHEN** 多键列表某条目的部分键叶在回读原文中缺失
- **THEN** 条目 SHALL 以「已有键字段 + 缺失字段零值」构造的 key struct 保留入 map，SHALL NOT 丢弃条目或整树报错

#### Scenario: 多键 key struct 字段不可转换（负路径）
- **WHEN** key 元数据返回值与 key struct 字段类型不可转换（生成物不一致等异常形态）
- **THEN** SHALL 返回命名该 list 的明确错误（R08，不 panic、不静默错键）

### Requirement: XC-03 键式删除编码（key 元数据驱动）

系统 SHALL 提供通用删除编码器：对 list 型 Object 的每个条目，主键 SHALL 经 `KeyedObject` key 元数据接口（YN-01，等价既有 `ΛListKeyMap()`）获取（key 叶为 nil 时以 schema Key 语句 + map key 回退），生成嵌套结构——模块顶层容器（带模块 namespace）> list 容器 > 条目元素携带 NETCONF base namespace 的 `operation="delete"` 属性 + 仅 key 叶（key 为首个子元素）。SHALL NOT 按模型类型 switch 硬编码 key 名。空条目集或无 key 信息 SHALL 返回明确错误（R08，不发送裸 delete 元素）。多 key list SHALL 返回明确不支持错误。

#### Scenario: 键式删除嵌套于模块顶层容器
- **WHEN** 对仅含键的 VLAN 条目集 / IFM 接口集做删除编码
- **THEN** 输出 SHALL 为 `<vlan xmlns=NS><vlans><vlan nc:operation="delete"…><id>10</id></vlan></vlans></vlan>` / `<ifm xmlns=NS><interfaces><interface …><name>…</name></interface></interfaces></ifm>`，条目仅携带 key 叶，DP-07 断言全部保持——嵌套结构使设备既有条目的删除能命中（修复「内置接口删不掉」）

#### Scenario: 空目标（负路径）
- **WHEN** 条目集为空
- **THEN** SHALL 返回明确编码错误，SHALL NOT 发送空删除

### Requirement: XC-08 YANG enumeration 值域名编解码

通用 XML 编解码引擎 SHALL 将 YANG `enumeration` 类型的 leaf 与 list key 编码为其 **YANG 值域名**（如 `<admin-status>up</admin-status>`），SHALL NOT 编码为底层整数值（如 `<admin-status>2</admin-status>`）——真实设备按值域名校验，整数会被拒。值域名 SHALL 经自研 `Enum` 接口的值域名映射（YN-01，等价既有 ygot `ΛMap`/`EnumName`）由枚举 int 值映射取得。`UNSET`（0）SHALL 跳发（不出现在 edit-config）。

引擎 SHALL 按值域名解码：接收的枚举文本 SHALL 经值域名映射反查值域名→int 还原为枚举常量。为不破坏历史报文/数据，解码 SHALL 兼容整数形态（值域名反查未命中时回退按整数解析）。

对值域名映射中**无对应值域名的枚举整数**（仅合成/异常值命中——真机有效枚举值必在映射中），编码 SHALL 退回输出该整数、SHALL NOT 报错或中断整树编码（R08）。

本要求 SHALL NOT 改变 XC-01~07 的既有行为（list/容器根编解码、per-node namespace、YANG empty 类型等）；非枚举 leaf 编解码不变。

#### Scenario: 枚举 leaf 编码为值域名
- **WHEN** 编码一个已设合法枚举值的 leaf（如 `admin-status = up`）
- **THEN** 输出 XML SHALL 为 `<admin-status>up</admin-status>`（值域名），SHALL NOT 为整数

#### Scenario: 枚举 leaf 按值域名解码
- **WHEN** 解码 `<admin-status>up</admin-status>`
- **THEN** 目标枚举字段 SHALL 还原为对应常量（`AdminStatus_up`）

#### Scenario: 解码兼容历史整数形态
- **WHEN** 解码 `<admin-status>2</admin-status>`（旧整数报文）
- **THEN** 引擎 SHALL 回退按整数解析、SHALL 还原为等价枚举值（不破坏历史数据）

#### Scenario: UNSET 枚举跳发
- **WHEN** 编码一个未设置（UNSET=0）的枚举 leaf
- **THEN** 该 leaf SHALL NOT 出现在输出 XML

#### Scenario: 未映射枚举值退回整数不中断（R08）
- **WHEN** 编码一个值域名映射中无对应名的枚举整数（合成/异常值）
- **THEN** 引擎 SHALL 输出该整数、SHALL NOT 报错、SHALL NOT 中断整树编码

#### Scenario: 真正非法枚举文本解码报错
- **WHEN** 解码一个既非合法值域名、又非整数的枚举文本
- **THEN** 引擎 SHALL 返回命名该 leaf 的明确错误（R08，不静默）
