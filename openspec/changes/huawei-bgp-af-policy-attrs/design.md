## Context

BGP 2b 的地基波次①②③④（tnlm/xpl/rtp/acl）+ XC-08 枚举修复已全部合入 main。四类 AF 策略属性目标模型均可配，越序禁令解除。本 change 是 2b 收官：接入 2a 排除的 AF 策略属性。

架构事实（决定体量极小，与 2a 同构）：AF 策略属性是 `huawei-bgp` augment 进 `/ni:network-instance` 的 AF 子树上的可选 leaf。network-instance 描述符（2a）覆盖整棵 ni 子树、reconciler 走容器根整根收敛、XC-06 已为 `huawei-bgp` 登记 per-node namespace。sanity 实测：`af/ipv4-unicast/import-filter-policy/{acl-name-or-num,filter-name}` 经 ni 描述符编码为 `<bgp xmlns="urn:huawei:yang:huawei-bgp">…<import-filter-policy><acl-name-or-num>G1</acl-name-or-num><filter-name>RF1</filter-name>…`，namespace 与真值均正确。故本 change 零新描述符/reconciler/codegen。

## Goals / Non-Goals

**Goals:**
- 接入 AF `import-filter-policy` 引用已集成模型的 leafref 属性（acl-name-or-num→acl、filter-name/filter-parameter→xpl）读改下发闭环。
- 证明 2b 全链路：AF 策略属性 → 已集成目标模型 leafref → 经 ni 链路端到端下发。
- 登记 leafref 编排依赖（目标先配）。
- 完备测试矩阵（T02b）绿灯。

**Non-Goals:**
- `ipv4-prefix-filter`（rtp ipv4-prefix-filters 未集成）、深层 route-policy（rtp）/tunnel-policy（tnlm）、export-filter、其他地址族、peer 级策略 —— follow-up。
- 跨模型自动事务（各 reconciler 独立；编排顺序由上层保证）。
- 新描述符/reconciler/codegen/前端硬编码表单。

## Decisions

**D1：接入边界取 AF `import-filter-policy` 引用已集成模型的 leafref 属性。**
- 理由：(1) 这些属性的目标（acl group、xpl route-filter）已集成，leafref 可解析、可端到端验证；(2) `ipv4-prefix-filter` 目标（rtp ipv4-prefix-filters）未集成，接入会致设备侧非法——排除是越序禁令的正确执行，非简化；(3) route-policy/tunnel-policy 目标已集成但位于更深嵌套容器，拆 follow-up 控体量。
- import-filter-policy 是自包含 flat 容器（4 leaf），是证明 AF 策略下发模式的最小干净单元。

**D2：零新描述符/reconciler——复用 ni 链路（同 2a）。**
- AF 策略 leaf 是 huawei-bgp 模块普通 *string leaf，在 ni 生成物 `...Afs_Af_Ipv4Unicast_ImportFilterPolicy` 内。desired 填上即经 ni 描述符 xmlcodec 编码（XC-06 namespace）下发。sanity 已证。

**D3：leafref 编排依赖登记，不改事务模型。**
- require-instance 语义 = 目标先存在。USMP 各模型 reconciler 独立（acl/xpl/ni 各自 reconcile）。本 change 交付 AF 属性下发 + spec 登记"目标先配"编排前置，不引入跨模型自动事务（超本 change 范围）。

**D4：测试先行（T05/T01）。**
- B1：`internal/drivers` af 策略编码断言（值 + huawei-bgp namespace + 零新描述符命中 ni）。
- B2：`internal/controller/networkinstance` 集成含 AF import-filter-policy 属性，下发→回读→收敛 + 幂等。

## Risks / Trade-offs

- **[R1] AF 策略 leaf 编码/namespace 缺口** → Mitigation：sanity 已证编码正确带 namespace；apply 测试固化。
- **[R2] 误接 ipv4-prefix-filter（目标未集成）** → Mitigation：spec/proposal 显式排除，测试只填 acl-name-or-num/filter-name。
- **[R3] leafref 编排被误解为自动事务** → Mitigation：spec AFPOL-02 明确登记"目标先配"、各 reconciler 独立。
- **[R4] 与①②③④ 冲突** → 无：本 change 不改 huawei.go（零新描述符），改的是测试 + 可能的 ni B2——与前四波次不同文件。

## Migration Plan

1. worktree 隔离（✅ `huawei-bgp-af-policy-attrs` off 含全部前置的 main，基线绿）。
2. TDD：写 af 策略编码断言 + B2 集成（红）→ 经既有链路即绿（预期零生产代码）。
3. `go test ./... -race` 全绿 + review → 提交（≤500 行）。
4. sync AFPOL delta → 主 spec；archive；PR 合入 → **BGP 2b 收官**。
5. 回滚：几乎全测试，零生产代码/生成物，回退零副作用。

## Open Questions

- route-policy/tunnel-policy/prefix-filter/其他族的接入时机——按实际配置需求分批 follow-up（prefix-filter 需先集成 rtp ipv4-prefix-filters）。本 change 不预判。
