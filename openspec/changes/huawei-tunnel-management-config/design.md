## Context

BGP 2b（公网邻居 AF 可选策略属性）门控于 4 个 leafref 目标模型的集成。public-config design.md 已核定 2b 依赖 DAG：`tunnel-management`(叶子, 仅依赖 ifm✅) 与 `xpl`(叶子) 无前置，`routing-policy` 依赖 tnlm，`acl` 依赖 time-range+l3vpn+ni。本 change 交付 DAG 第一块地基 `huawei-tunnel-management`。

关键既有事实（决定本 change 体量小、风险低）：
- **结构体已生成**：`huawei-tunnel-management` 的 60 个类型在一期 `huawei-bgp` 全闭包生成时已产出（ygot 因 augment 级联物化整个 import 闭包）。本 change codegen 零新增。
- **容器根路径已铺通**：`/tunnel-management` 与 `/bgp:bgp` 同构（模块顶层容器根，非 list 根）。通用引擎的 plain-container 编解码（XC-05）与 per-node namespace（XC-06）在一期 BGP 已交付并验证。
- **playbook 成熟**：VLAN→IFM→network-instance→BGP 四次「描述符 + 通用引擎 + netconfsim 方言 + 完备矩阵」已跑通，本 change 是第五次复用。

## Goals / Non-Goals

**Goals:**
- 让 `huawei-tunnel-management` 成为 USMP 可配模型：`/tunnel-management/tunnel-policys/tunnel-policy` 按 `name` 存在、`description` 与 `tunnel-down-switch/enable` 可配，读改下发闭环。
- 满足 BGP `tunnel-policy` leafref 前置：目标 list 实例按 `name` 存在即解除 require-instance 阻塞。
- 零 codegen、零 XML 手写代码，纯描述符 + 通用引擎。
- 完备测试矩阵（T02b）绿灯。

**Non-Goals:**
- 深层 `ipv4-set`/`ipv6-set` 子树（choice/presence/ordered-by/nexthops binding/tunnel-name leafref/auto-name）——本波次-follow-up，门控于通用引擎对应能力实测。
- config-false 只读态（`tunnel-infos`/`subscribe-tunnel-policys`）——永不作为下发目标。
- BGP `tunnel-policy` 属性本身的接入——波次⑤，本 change 只交付其前置。
- 前端硬编码表单——经通用模块控制台 YANG 自动渲染（R05）。

## Decisions

**D1：接入边界取「标量层」而非整模型一次接入。**
- 理由：(1) BGP 只按 `name` 引用 tunnel-policy，标量边界（name+description+tunnel-down-switch）已完整解除 leafref 阻塞，无功能缺口；(2) 深层 ipv4/ipv6-set 含 choice/presence/ordered-by/嵌套 list/跨模型 leafref，是通用引擎尚未走过的路径，一次接入会撞 §5.3 体积军规且叠加多个未知 XC 缺口，违背「渐进替换、禁一次性重写」。分期边界显式注册（proposal「明确排除」+ TNLM-01），完备性军规作用于**声明边界**，非简化。
- 备选：整模型一次接入——弃，体量与风险双高，且 MVP 价值（BGP 解锁）标量边界已足。

**D2：复用既有链路，新增仅一条描述符。**
- `backend/internal/drivers/huawei.go` 加 `driver.Descriptor{Vendor:"huawei", Module:"tunnel-management"}`：`MatchRoute`/`MatchDecode`/`MatchEncode` 锚定 `tunnel-management` 根路径 + 显式 `Namespace = "urn:huawei:yang:huawei-tunnel-management"` 常量（不依赖 `Entry.Namespace()` 派生空，同 BGP-02）+ SchemaTree 入口闭包返回 `&HuaweiTunnelManagement_TunnelManagement{}`。编解码全走 `pkg/yang-runtime/xmlcodec`。
- 谓词精确锚定根 token `tunnel-management`，避免子串误匹配（同 BGP-03 教训，虽 tunnel-management 无同族 feature 前缀，仍精确匹配以防未来 `tunnel-management-ext` 混淆）。

**D3：netconfsim 增 tunnel-management 方言，走既有 RFC edit-config 整树替换通道。**
- 复用 config-delete-semantics 已交付的 RFC edit-config（整树替换）+ Decode 锚定容器解歧义。tunnel-management 为容器根，Decode 需锚定 `<tunnel-management>` 顶层容器（同 vlan 同名歧义教训）。

**D4：测试先行（T05/T01），schema 驱动完备性。**
- B1：描述符谓词/查找表格驱动（含 race）、xmlcodec tunnel-policy 往返（含 namespace 真值、边界长度）。
- B2：`*_integration_test.go` 下发→回读→收敛 + 幂等（`testing.Short()` 跳过）。
- schema 驱动枚举标量 config-true leaf 对照 fixture，计数断言防遗漏。

## Risks / Trade-offs

- **[R1] 容器根 Encode 报「no list map field」** → Mitigation：一期 BGP 已由 XC-05 修复容器根 encode/decode 路径并回归，tunnel-management 同构直接复用；apply 期首个往返测试即验证，若仍暴露缺口按 TDD 补 yang-xml-codec delta（回填 proposal Modified Capabilities）。
- **[R2] 生成类型虽在但描述符入口闭包类型名/路径写错** → Mitigation：入口类型 `HuaweiTunnelManagement_TunnelManagement` 已由 `grep` 核实存在于 `all.gen.go`；apply 首步编译 + 注册可达性测试（Lookup ok=true）拦截。
- **[R3] 误判「模型已集成」** → Mitigation：proposal/spec 显式登记本波次仅标量边界功能集成，深层 ipv4/ipv6-set 仍 generated-but-not-integrated。
- **[R4] namespace 派生空致报文无 ns** → Mitigation：显式常量登记（D2），编码测试断言根元素 namespace（TNLM-02）。

## Migration Plan

1. worktree 隔离（✅ 本 change 已在 `huawei-tunnel-management-config` worktree，基线 `go build ./...` 绿）。
2. TDD：先写描述符谓词/xmlcodec 往返/B2 集成测试（红）→ 加描述符 + netconfsim 方言（绿）。
3. `go test ./...`（含 `-race`）全绿 + `go-code-review-check` 通过 → 提交（What/Why/How，≤500 行）。
4. sync delta spec → 主 spec；archive change；PR 合入（≤1000 行）。
5. 回滚：本 change 仅新增一条描述符 + 测试 + sim 方言，无生成物/无架构改动，回退 = 删描述符 remove 分支，零副作用。

## Open Questions

- 深层 ipv4/ipv6-set follow-up 是否单独 change 还是并入波次⑤——待本 change 合入后按 BGP 属性接入实际需要决定（BGP 只需 name，深层选路细节非 BGP 依赖，可能长期推迟）。
