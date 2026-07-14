## Why

BGP 二期-2b 的地基已全部就位：波次①②③④（tunnel-management #155 / xpl #156 / routing-policy #157 / acl #159）+ 平台级枚举修复（XC-08 #158）依次合入，使 BGP AF 策略属性引用的四类目标模型（tnlm/xpl/rtp/acl）全部成为 USMP 可配模型——越序禁令解除。

2a（`huawei-bgp-neighbor-config`）曾显式排除"af 内策略属性（import/export route-policy、route-filter、ACL group、tunnel-policy）——门控于 2b"。本 change 是 2b 的**收官波次⑤**：接入这些可选 leafref 策略属性本身，证明其经既有 network-instance 链路端到端下发、且 leafref 目标（现已可配）可解析。

**关键架构事实（决定本 change 体量极小）**：这些属性是 `huawei-bgp` augment 进 `/ni:network-instance` 的 AF 子树上的可选 leaf。2a 已证 network-instance 描述符 + reconciler **覆盖整棵 ni 子树（含 bgp augment）**、且 XC-06 per-node namespace 已为 `huawei-bgp` 登记。故本 change **零新描述符、零新 reconciler、零 codegen**——只在 desired 填 AF 策略属性字段，经既有链路编码（带 huawei-bgp namespace）下发。主体是**完备测试矩阵**（sanity 已证 `af/ipv4-unicast/import-filter-policy/{acl-name-or-num,filter-name}` 经 ni 描述符正确编码带 namespace）。

## What Changes

- **接入 AF `import-filter-policy` 策略属性面**：`/ni:.../instance[_public_]/bgp/base-process/afs/af[type]/ipv4-unicast/import-filter-policy` 下引用**已集成**目标模型的 leafref 可选属性：
  - `acl-name-or-num`（leafref → `/acl:acl/groups/group/identity`，acl ✅ #159）
  - `filter-name`（leafref → `/xpl:xpl/route-filters/route-filter/name`，xpl ✅ #156）+ `filter-parameter`（xpl filter 参数）
- **复用既有链路，零新描述符/控制器/codegen**：经 network-instance 描述符（谓词已覆盖 af 路径）+ ni reconciler 容器根收敛 + XC-06 per-node namespace（`<bgp>`↓ 带 huawei-bgp namespace，AF 策略 leaf 归 huawei-bgp 模块自动继承）。
- **完备测试矩阵**（`yang-config-test-design` / T02b）：编码真值（leaf 值 + huawei-bgp namespace）/ B2 下发→回读→收敛（经 ni reconciler）/ leafref 编排依赖登记（目标实例须先由对应模型 reconciler 配置）/ 并发-race / 负路径 / 幂等。
- **明确排除（分期，注册为 follow-up，非简化）**：
  - `import-filter-policy/ipv4-prefix-filter`（leafref → rtp **`ipv4-prefix-filters`**——该 rtp 子树**未集成**，波次③ 只接了 `policy-definitions`；接入需先补 rtp ipv4-prefix-filters 集成）。
  - **route-policy**（→ rtp policy-definition ✅）与 **tunnel-policy**（→ tnlm ✅）——位于更深嵌套容器（import-routes/export-filter-policys 等），目标已集成但结构更深，follow-up。
  - `export-filter-policys`、ipv6/vpn 等其他地址族、peer 级策略属性 —— follow-up。

## Capabilities

### New Capabilities
- `huawei-bgp-af-policy-config`: 华为 BGP AF `import-filter-policy` 策略属性（`acl-name-or-num`→acl、`filter-name`/`filter-parameter`→xpl）的模型驱动配置管理——覆盖字段清单、复用 ni 描述符与 XC-06 namespace、leafref 编排依赖（目标模型 tnlm/xpl/rtp/acl 已集成）、分期边界（prefix-filter/route-policy/tunnel-policy/其他族 follow-up）、以及完备测试矩阵。是 BGP 2b 的收官（2a 排除的策略属性由此兑现）。

### Modified Capabilities
<!-- 预期无：ni 描述符/reconciler、XC-06 namespace 均已就位（2a/#151）。AF 策略 leaf 是 huawei-bgp 模块普通 *string leaf，经既有链路编码（sanity 已证）。若 apply 期暴露缺口再补。 -->

## Impact

- **代码**：预期**几乎全是测试**（`internal/drivers` af 策略编码断言 + `internal/controller/networkinstance` B2 集成含 AF 策略属性）；生产代码预期零（描述符/reconciler/namespace 均已就位）。若暴露缺口按 TDD 补。**不动 `generated/`、不动 `gen.conf`、不动 `huawei.go`**（零新描述符——与①②③④不同文件，理论不与其串行）。
- **依赖**：acl（✅ #159）、xpl（✅ #156）——本 change 属性的 leafref 目标；network-instance 描述符（✅）+ XC-06（✅ #151）。
- **leafref 编排语义登记**：AF 策略属性携带目标实例名（string）；真机 require-instance 要求目标先存在——即编排顺序上"先由 acl/xpl reconciler 配置 group/route-filter，再下发引用它的 BGP AF"。本 change 交付属性下发能力 + 登记该编排依赖，不改跨模型事务模型。
- **前端**：AF 策略属性经通用模块控制台 YANG 自动渲染（R05）。
- **合规**：R04、T02b、B2 集成、worktree 隔离、≤500 行/commit、PR ≤1000 行。
