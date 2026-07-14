## Context

BGP 2b 的 import/export route-policy 属性门控于 `huawei-routing-policy`（rtp）集成（越序禁令）。rtp 在 2b DAG 中依赖 tunnel-management（波次①，#155 已合入 main）。波次①② 已实证容器根经通用引擎 plain-container（XC-05）+ per-node ns（XC-06）编解码无新缺口、netconfsim 模型无关零方言、结构体一期全闭包已生成零 codegen。本 change 第四次复用该 playbook。

BGP 引用的精确目标（huawei-bgp-common.yang:165/551/575 等）= `/rtp:routing-policy/policy-definitions/policy-definition/name`。policy-definition 生成物 = {Name, AddressFamilyMismatchDeny(*bool), Nodes(*struct)}——直属 config-true 标量 2 个 + 深层 Nodes 子树。Nodes 子树在 YANG 中约 1490 行（node 的 conditions/actions 深层嵌套 list + choice），是 route-policy 的实际逻辑。

## Goals / Non-Goals

**Goals:**
- 让 rtp `policy-definitions/policy-definition` 成为 USMP 可配模型：按 name + address-family-mismatch-deny 读改下发闭环。
- 满足 BGP import/export route-policy leafref 前置：目标 list 实例按 name 存在即解除 require-instance 阻塞。
- 零 codegen、零 XML 手写、纯描述符 + 容器根 reconciler + 通用引擎。
- 完备测试矩阵（T02b）绿灯。

**Non-Goals:**
- policy-definition 深层 `nodes/node`（conditions/actions 匹配动作子句）——route-policy 实际逻辑，follow-up。
- rtp 其他 filter/list（community/ext-community/prefix/as-path/rd/large-community 等）——各由自身 BGP 消费属性门控。
- config-false 只读态、BGP route-policy 属性本身（波次⑤）、前端硬编码表单（R05 自动渲染）。

## Decisions

**D1：接入边界取 `policy-definition` 标量层，深层 nodes 推迟。**
- 理由：(1) BGP 只按 name 引用 route-policy，标量边界（name + address-family-mismatch-deny）已完整解除 leafref 阻塞；(2) nodes 子树 ~1490 行含 conditions/actions 深层嵌套 list + choice，一次接入撞体积军规且叠加多个未知 XC 缺口，违背「渐进替换、禁一次性重写」。分期边界显式注册（proposal + RTP-01），完备性军规作用于声明边界。
- 与波次①（tunnel-policy 标量层，推迟 ipv4/ipv6-set）同构。

**D2：复用容器根 playbook，新增一条描述符 + 一个 reconciler 包。**
- `huawei.go` 加 `Descriptor{Vendor:"huawei", Module:"routing-policy"}`：谓词 `HasPrefix "/rtp:routing-policy"` + 显式 `HuaweiRoutingPolicyNS` + SchemaTree 入口 `HuaweiRoutingPolicy_RoutingPolicy`。
- `internal/controller/routingpolicy` 镜像 xpl/tunnelmgmt 容器根 reconciler；deviceClient.Get 走 DecoderFor container 模式，JSON 路径 `deviceRoot.RoutingPolicy`。

**D3：netconfsim 零改动**（波次①② 已实证模型无关）。

**D4：测试先行（T05/T01），schema 驱动完备性。**
- B1：描述符谓词对拍（含负路径、race）+ xmlcodec policy-definition 往返（namespace 真值）+ schema 驱动形状锁定（直属 config-true 标量恰好 2、深层 nodes 为推迟容器）。
- B2：`*_integration_test.go` 下发→回读→收敛 + 幂等。

## Risks / Trade-offs

- **[R1] 容器根 Encode/Decode** → Mitigation：波次①②③ 已回归容器根路径，直接复用；apply 首个往返测试验证。
- **[R2] 描述符入口类型/路径写错** → Mitigation：`HuaweiRoutingPolicy_RoutingPolicy` 已 grep 核实存在；编译 + 注册可达性测试拦截。
- **[R3] 误判「rtp 已整体集成」** → Mitigation：proposal/spec 显式登记仅 policy-definition 标量边界功能集成，深层 nodes + 其他 filter 仍 generated-but-not-integrated；schema 形状测试锁死推迟边界。
- **[R4] 与波次①②④ 同改 huawei.go 冲突** → Mitigation：TM03 串行，基于 #155/#156 已合入 main，合入后波次④再开。

## Migration Plan

1. worktree 隔离（✅ 本 change 在 `huawei-routing-policy-config` worktree off 含 #155/#156 的 main，基线 `go build ./...` 绿）。
2. TDD：先写描述符谓词/xmlcodec 往返/B2 集成测试（红）→ 加描述符 + reconciler（绿）。
3. `go test ./... -race` 全绿 + `go-code-review-check` 通过 → 提交（≤500 行，拆描述符+codec / reconciler+B2）。
4. sync delta → 主 spec；archive；PR 合入（≤1000 行）。
5. 回滚：仅新增一条描述符 + reconciler 包 + 测试，无生成物/架构改动，回退零副作用。

## Open Questions

- policy-definition 深层 nodes 的接入时机——待 BGP route-policy 属性（波次⑤）落地后，若需真正配置 route-policy 逻辑再按 TDD 分批接入（choice/嵌套 list 逐个补 XC delta），本 change 不预判。
