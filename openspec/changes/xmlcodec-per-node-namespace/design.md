## Context

`xmlcodec` 的 `Spec` 现有单一 `Namespace`，`Encode` 只在根元素（及外包祖先容器的最外层）声明它，子节点靠 XML 默认命名空间继承。这对单模块树正确。BGP 二期 2a 的 peers/afs 是 `huawei-bgp` augment 到 `huawei-network-instance` 根下——生成物字段的 `module:"…"` tag 忠实记录了每个节点的所属模块（`bgp`/`afs`/`peers`/`peer` 均 `module:"huawei-bgp"`，而 `name`/`description` 为 `module:"huawei-network-instance"`）。故 per-node namespace 的信息已在生成物内，缺的只是编码器据此发射。

约束：R04（禁改 generated/）、向后零回归（vlan/ifm/bgp golden 逐字节）、TDD 红灯先行、engine 通用（device-specific 全走数据）。

## Goals / Non-Goals

**Goals:**
- 编码器按 `module` tag 在模块边界发 `xmlns`，支持 augment 跨模块树正确 namespace。
- 完全向后兼容：单模块树零新增发射、golden 逐字节不变。
- ni 描述符登记多 namespace，为 2a peers 铺路（本 change 不接 peer 字段）。

**Non-Goals:**
- peers/afs 字段接入（2a）。
- 解码侧改动（已 namespace-宽容）。
- namespace 派生规则（各模块 namespace 仍是显式数据，无统一派生——vlan/ifm 旧式 `params:xml:ns` vs bgp/ni 新式 `urn:huawei:yang:`）。

## Decisions

**D1：`Spec` 加 `Namespaces map[string]string`（module→URI），可选。**
缺省 nil → 现状单根行为。有值时作为 per-node 解析表。不改 `Namespace`（根仍用它，向后兼容）。
- 备选：用 resolver func `func(module) string`——否决，map 更简单、可测、无闭包状态；未来需动态再演进。
- 备选：从生成物内嵌 schema 派生 namespace——否决，实测 `Entry.Namespace()` 返空（既有 design D3b 结论），且 vlan/ifm 旧式 namespace 与 module 名无派生关系。

**D2：编码下降跟踪「父生效 namespace」，模块边界发 xmlns。**
`encodeFields`/容器编码携带一个 `parentNS` 参数（根为 `Spec.Namespace`）。对每个字段：取其 `module` tag → `ns := Namespaces[module]`（查不到则 ns=parentNS，不发）；若 `ns != "" && ns != parentNS` → 该元素开标签发 `xmlns=ns`，其子树以 ns 为 parentNS 递归；否则不发、沿用 parentNS。list 条目继承其 list 字段的 ns 判定。
- 关键不变量：单模块树中所有字段 module 相同 → Namespaces 要么无该 module 项、要么值==根 ns → 永不发新 xmlns → **golden 逐字节不变**。

**D3：moduleTag helper 与 pathTag 并置。**
复用现有反射字段遍历，加 `moduleTag(f) = f.Tag.Get("module")`（已在 ni 完备测试用过同款）。不引入新遍历。

**D4：正确性防线 = 编码 namespace 真值断言（非往返/sim）。**
因 sim/decode namespace-宽容，往返/集成测**证不出** namespace 错。故 XC-06 用例断言 **encode 输出**：ni 根下 bgp 子树的 `<bgp>` 元素**必须**带 `xmlns="urn:huawei:yang:huawei-bgp"`，而 `<name>` 不另发（继承 ni 根）。并保留混合 namespace 往返用例证明解码鲁棒。

## Risks / Trade-offs

- **[单模块树误发 xmlns]** encode 增量若模块边界判断有 bug → vlan/ifm/bgp 多发 xmlns、golden 变。→ **Mitigation**：现有 XC-01/02/05 golden 逐字节断言 + 往返恒等即硬门禁；红灯先行先写「单模块零新增发射」断言。
- **[Namespaces 未登记致漏发]** 若 ni Spec 未填 huawei-bgp → bgp 子树仍继承 ni ns（真机错，但测不出）。→ **Mitigation**：本 change 在 ni 描述符显式填 huawei-bgp namespace，并加 encode 断言用例锁定；未来加新 augment 模块时同步登记（登记为 2a 前置检查项）。
- **[deep 嵌套多重模块边界]** peer→afs→af 若再跨模块 → 多层 xmlns。→ 递归 parentNS 天然支持；2a 集成时验证。

## Migration Plan

worktree 内 TDD：先写「单模块零新增发射（golden 不变）」+「ni 根下 bgp 子树带 huawei-bgp namespace」红灯 → 加 Spec.Namespaces + encode per-node 发射 → 绿 → ni 描述符登记多 namespace → review → commit。回滚=移除 Namespaces 字段与发射分支（纯增量、无状态迁移）。

## Open Questions

- 无。能力边界清晰，2a 接 peer 字段时复用本能力。
