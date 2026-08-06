## MODIFIED Requirements

### Requirement: XC-02 通用 XML 解码（回读全字段对称）

系统 SHALL 提供通用 NETCONF XML 解码器：对 get-config 回读原文（无论包裹在 `<rpc-reply>`/`<data>`、模块顶层容器或裸容器、无论 namespace 前缀），SHALL 在 list 容器（root，如 `<vlans>`/`<interfaces>`）之内定位 list 条目元素并填充 ygot GoStruct（含 list map、嵌套 list、enum、指针叶）。锚定 list 容器 SHALL 消解「模块顶层容器名与条目名同名」的歧义（华为 `<vlan><vlans><vlan>` 外层模块容器与条目同名 `vlan`，裸扫条目名会误把外层容器当条目）。解码字段覆盖 SHALL 为编码字段的超集（同一份 tag 数据驱动，SHALL NOT 出现「可下发但回读丢失」的字段），呈现叶（config-false，如 class/parent-name）SHALL 照常透出。

条目 key SHALL 经 `ΛListKeyMap()` 获取；**单键列表**以该键值为 map key，**多键（复合键）列表**（map key 为 ygot 生成的复合 key struct，如 devm `physical-entity` 的 class+position+serial-number）SHALL 按 key struct 字段的 `path:` tag 从 `ΛListKeyMap()` 结果填充复合 key struct 作为 map key，SHALL NOT 因键数量 > 1 而报错或丢弃整表。此机制对根级列表与嵌套列表 SHALL 一致适用。key 叶缺失时 SHALL 合成 key 保留条目（宽容语义）：单键列表合成标量 key；多键列表 SHALL 从条目自身键字段复制可得值构造 key struct（缺失字段为零值），条目不丢弃。空输入或无条目 SHALL 返回非 nil 空容器；非法 XML 或非数值 enum 文本 SHALL 返回明确错误。

> 范围注记：键式删除编码（XC-03）的「多 key list SHALL 返回明确不支持错误」契约**保持不变**——多键列表绝大多数为 config-false 状态/统计表，删除支持由真实需求驱动另行变更。

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
- **WHEN** `ΛListKeyMap()` 返回值与 key struct 字段类型不可转换（生成物不一致等异常形态）
- **THEN** SHALL 返回命名该 list 的明确错误（R08，不 panic、不静默错键）
