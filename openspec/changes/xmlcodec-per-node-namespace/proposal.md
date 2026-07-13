## Why

BGP 二期 2a 要接入 peers/afs——它们是 `huawei-bgp` 通过 `augment` 挂到 `/ni:network-instance/instances/instance/bgp/…` 下的节点。因此一条 network-instance-with-peers 的 edit-config 报文**跨命名空间**：`<network-instance>`/`<instance>`/`<name>` 属 `huawei-network-instance`（`urn:huawei:yang:huawei-network-instance`），而 `<bgp>` 及其下全部 peers/afs 属 `huawei-bgp`（`urn:huawei:yang:huawei-bgp`）。现有通用引擎 `pkg/yang-runtime/xmlcodec` 只在**根元素**声明单一 namespace，子节点靠 XML 默认命名空间继承——对**单模块**树（vlan/ifm/bgp/system/ni 均是）正确，但对 **augment 跨模块**树会给 bgp 子树打上错误的 ni 命名空间。真实 8.20.10 设备按 namespace 校验会拒绝该报文。这是接任何 augment-under-ni 模型（2a peers、后续 per-VPN feature）的硬前置引擎能力，须先独立交付。

⚠️ **测试测不出、真机才暴露**：netconfsim 按 local 名存取、namespace-宽容，decode 也按 local 名不校验 namespace（见 `bgp-onboarding-deps` 记忆）——错误 namespace 在 sim/单测全绿。故本 change 的正确性防线是**编码输出的 namespace 断言**（encode 侧真值），而非往返/集成。

## What Changes

- **`xmlcodec.Spec` 新增可选 per-node namespace 解析**：新增字段 `Namespaces map[string]string`（module 名 → namespace URI）。缺省 nil/空 → 保持现状（单根 namespace）。
- **编码器 per-node namespace 发射**：编码下降时跟踪「生效 namespace」，根为 `Spec.Namespace`；对每个字段读其 `module` struct tag，若 `Namespaces[module]` 存在且 ≠ 父节点生效 namespace，则在该子元素上发 `xmlns="…"` 并作为其子树的生效 namespace（子节点默认继承直到下一个模块边界）。list 条目与嵌套子容器一致处理。
- **完全向后兼容**：单模块树（所有字段 module 同根）→ 查不到差异 namespace → **零新增 xmlns 发射**、输出逐字节不变（vlan/ifm/bgp/system/ni 的 XC-01/02/05 golden 与往返恒等全保持）。
- **解码不变**：解码本就按 local 名穿透 namespace 前缀（XC-02），跨 namespace 报文天然可解；补一条混合 namespace 往返用例证明。
- **驱动侧登记**：`internal/drivers` 为 network-instance 描述符的 Spec 填 `Namespaces`（含 huawei-network-instance + huawei-bgp，后续按需加 huawei-l3vpn 等），使 ni 根下 bgp 子树编码带正确 namespace。
- **明确排除**：2a 的 peers/afs 字段接入本身（下一个 change）；本 change 只交付引擎的 per-node namespace 能力 + ni 描述符登记多 namespace，不新增任何 peer 配置字段。

## Capabilities

### Modified Capabilities
- `yang-xml-codec`: **ADD XC-06**——通用引擎按 `module` tag 发 per-node namespace，支持 augment 跨模块树（如 ni 根下 bgp 子树）的正确 namespace 编码；单模块树零回归。

## Impact

- **代码**：`pkg/yang-runtime/xmlcodec`（Spec + encode 增量 + moduleTag helper + 单测）、`internal/drivers/huawei.go`（ni Spec 填 Namespaces + 混合 namespace 编码断言测试）。
- **风险（低）**：encode 增量若误判模块边界 → 单模块树多发 xmlns。由 vlan/ifm/bgp golden 逐字节断言拦截（零回归是硬门禁）。
- **合规**：R04（不改 generated/）、TDD 红灯先行、`go-code-review-check`、≤500 行/commit。
