## ADDED Requirements

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
