## ADDED Requirements

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
