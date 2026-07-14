## Context

BGP 2b 的 route-filter 属性门控于 `huawei-xpl` 集成（越序禁令）。xpl 是 2b DAG 的叶子（仅依赖 ifm✅）。波次①（tunnel-management，PR #155）已实证：容器根模型经通用引擎 plain-container（XC-05）+ per-node ns（XC-06）编解码**无新缺口**，netconfsim 模型无关零方言，结构体一期全闭包已生成故零 codegen。本 change 第三次复用该 playbook。

BGP 引用的精确目标（huawei-bgp-common.yang 多处 leafref）= `/xpl:xpl/route-filters/route-filter/name`。route-filter list 结构极简：key=`name`（xpl-filter-name）+ `content`（string 1..16380，mandatory，XPL 策略正文文本 blob）——无嵌套、无 choice/presence。故本波次可完整接入该子树，无推迟子字段。

## Goals / Non-Goals

**Goals:**
- 让 xpl `route-filters/route-filter` 成为 USMP 可配模型：按 `name` + `content` 读改下发闭环。
- 满足 BGP `route-filter` leafref 前置：目标 list 实例按 `name` 存在即解除 require-instance 阻塞。
- 零 codegen、零 XML 手写、纯描述符 + 容器根 reconciler + 通用引擎。
- 完备测试矩阵（T02b）绿灯。

**Non-Goals:**
- xpl 其他策略 list（global/as-path-lists/community-lists/prefix-lists/rd-lists/large-community-lists/route-flow-group-lists/ext-community-*/interface-lists）——非 BGP route-filter leafref 目标，各由自身消费者门控，follow-up。
- config-false 只读态——永不作为下发目标。
- BGP `route-filter` 属性本身的接入——波次⑤。
- 前端硬编码表单——通用模块控制台 YANG 自动渲染（R05）。

## Decisions

**D1：接入边界取 BGP 目标子树 `route-filters/route-filter`，而非整 xpl 模块。**
- 理由：(1) BGP 只按 name 引用 route-filter，该子树已完整解除 leafref 阻塞；(2) xpl 其他 list 是独立策略构造、非本 leafref 目标，一次全接会撞体积军规且叠加各自消费者未定的语义。分期边界显式注册（proposal「明确排除」+ XPL-01）。
- 与波次① tunnel-management 的分期同构：接 BGP 目标、推迟无关子树。

**D2：复用容器根 playbook，新增一条描述符 + 一个 reconciler 包。**
- `huawei.go` 加 `Descriptor{Vendor:"huawei", Module:"xpl"}`：谓词 `HasPrefix "/xpl:xpl"` 精确锚定 + 显式 `HuaweiXplNS = "urn:huawei:yang:huawei-xpl"` + SchemaTree 入口 `HuaweiXpl_Xpl`。
- `internal/controller/xpl` 镜像 tunnelmgmt/bgp 容器根 reconciler（单条整根 MODIFY 收敛）；deviceClient.Get 走 DecoderFor container 模式，JSON 路径 `deviceRoot.Xpl`。

**D3：netconfsim 零改动。**
- 波次① 已实证 netconfsim 模型无关（tree_datastore + RFC edit-config 整树替换），B2 直接复用既有通道。

**D4：测试先行（T05/T01），schema 驱动完备性。**
- B1：描述符谓词对拍（含负路径、race）+ xmlcodec route-filter 往返（namespace 真值、边界长度 content 1..16380）+ schema 驱动断言 route-filter config-true 标量恰好 2。
- B2：`*_integration_test.go` 下发→回读→收敛 + 幂等（`testing.Short()` 跳过）。

## Risks / Trade-offs

- **[R1] 容器根/大字符串 content(16380) 编解码** → Mitigation：容器根路径波次① 已回归；大 string 是普通标量 leaf，往返测试覆盖上界长度。
- **[R2] 描述符入口类型/路径写错** → Mitigation：`HuaweiXpl_Xpl` 已 grep 核实存在；apply 首步编译 + 注册可达性测试拦截。
- **[R3] 误判「xpl 已整体集成」** → Mitigation：proposal/spec 显式登记仅 route-filter 功能集成，其他 list 仍 generated-but-not-integrated。
- **[R4] 与波次①/③④ 同改 huawei.go 冲突** → Mitigation：TM03 串行，本 change 基于 #155 已合入 main，合入后波次③再开。

## Migration Plan

1. worktree 隔离（✅ 本 change 在 `huawei-xpl-config` worktree off 已含 #155 的 main，基线 `go build ./...` 绿）。
2. TDD：先写描述符谓词/xmlcodec 往返/B2 集成测试（红）→ 加描述符 + reconciler（绿）。
3. `go test ./... -race` 全绿 + `go-code-review-check` 通过 → 提交（What/Why/How，≤500 行）。
4. sync delta → 主 spec；archive；PR 合入（≤1000 行）。
5. 回滚：仅新增一条描述符 + reconciler 包 + 测试，无生成物/架构改动，回退零副作用。

## Open Questions

- xpl 其他策略 list 的接入时机——待各自消费者（如 BGP 其他属性、其他协议）实际需要时按 DAG 补，本 change 不预判。
