## Context

BGP 2b 的 ACL group 属性门控于 `huawei-acl` 集成（越序禁令）。BGP 引用 `/acl:acl/groups/group/identity`（IPv4）与 `/acl:acl/group6s/group6/identity`（IPv6）（huawei-bgp-common.yang:1089/1160/1572…）。波次①②③ 已实证容器根 playbook 稳定（复用 XC-05/XC-06 零缺口、netconfsim 零方言、结构体已生成零 codegen）。本 change 第五次复用。

**关键范围核实（纠正早前 codegen 顾虑）**：早前 DAG 登记「acl 需先补 time-range/l3vpn codegen」。实测：
- `time-range` 类型一期全闭包已生成（`HuaweiTimeRange_*` 14 类型）。
- `l3vpn` 类型未生成，但 acl 对 l3vpn 的引用**全部是深层 `rule-*` 子树内的 `must` 约束**（huawei-acl.yang:537/793/1523/1840/2155，`must "/ni:.../l3vpn:afs/..."`），非 require-instance leafref；且 `must` 在 l3vpn 未配时 xpath 求空自满足。
- acl group **标量/枚举边界**（identity/type/match-order/step/description/number）**不引用 l3vpn/time-range**。
- 结论：本波次（group 标量边界）**零 codegen**，与①②③同构。l3vpn 仅在接入深层 rule-* 时才需，属 follow-up。

## Goals / Non-Goals

**Goals:**
- 让 acl `groups/group` + `group6s/group6` 成为 USMP 可配模型：按 identity + type(enum) + 标量读改下发闭环。
- 满足 BGP ACL group leafref 前置：目标 list 实例按 identity 存在即解除 require-instance 阻塞。
- 零 codegen、零 XML 手写、纯描述符 + 容器根 reconciler + 通用引擎；首次覆盖枚举 leaf 编解码。
- 完备测试矩阵（T02b）绿灯。

**Non-Goals:**
- group/group6 深层 `rule-*`（实际 ACL 规则，含 l3vpn/time-range/ni must 与深层嵌套 list）——follow-up，门控于 l3vpn 集成 + 深层嵌套引擎支持。
- acl `ip-pools`/`ip-pool6s`/`port-pools`——非 BGP leafref 目标，follow-up。
- config-false 只读态、BGP ACL group 属性本身（波次⑤）、l3vpn 模型集成、前端硬编码表单（R05 自动渲染）。

## Decisions

**D1：接入边界取 group/group6 标量+枚举层，深层 rule-* 推迟。**
- 理由：(1) BGP 只按 identity 引用 ACL group，标量/枚举边界已解除 leafref 阻塞；(2) rule-* 是实际规则条目，含 l3vpn must（须 l3vpn 集成）+ 深层嵌套 list，一次接入撞体积军规 + 引入未集成模型依赖。分期边界显式注册（proposal + ACL-01）。
- 与波次①③（标量层，推迟深层）同构。

**D2：group 与 group6 同 change 一并接入。**
- BGP 同时引用两者（groups/group + group6s/group6），且结构近同（group6 少 rule-ethernets/rule-mplss）。一并接入才完整解锁 BGP ACL group（v4+v6）；二者标量边界都小，合并不超体积。

**D3：复用容器根 playbook，新增一条描述符 + 一个 reconciler 包。**
- `huawei.go` 加 `Descriptor{Vendor:"huawei", Module:"acl"}`：谓词 `HasPrefix "/acl:acl"` + 显式 `HuaweiAclNS` + SchemaTree 入口 `HuaweiAcl_Acl`。
- `internal/controller/acl` 镜像 routingpolicy/xpl 容器根 reconciler；JSON 路径 `deviceRoot.Acl`。

**D4：枚举 leaf 首次进入波次边界——重点测枚举往返。**
- type（mandatory group4-type/group6-type）、match-order 是枚举。BGP 2a af-type 已实证引擎支持枚举 key；本波次断言枚举**值**编码为值域名、回读还原为枚举常量（往返等价）。

**D5：netconfsim 零改动**（①②③ 已实证模型无关）。

## Risks / Trade-offs

- **[R1] 枚举 leaf 编解码缺口** → Mitigation：BGP 2a af-type 枚举已过引擎；apply 首个含 enum 的往返测试即验证，若暴露缺口按 TDD 补 xmlcodec delta。
- **[R2] mandatory type 缺失致校验失败** → Mitigation：所有 fixture 均设 type（basic 等有效值）；负路径测缺 type 由校验拦截。
- **[R3] 误判「acl 已整体集成」** → Mitigation：proposal/spec 显式登记仅 group/group6 标量边界功能集成；深层 rule-*、ip-pools、l3vpn 仍未集成。
- **[R4] 与①②③ 同改 huawei.go 冲突** → Mitigation：TM03 串行，基于 #155/#156/#157 已合入 main。

## Migration Plan

1. worktree 隔离（✅ 本 change 在 `huawei-acl-config` worktree off 含 #155/#156/#157 的 main，基线 `go build ./...` 绿）。
2. TDD：先写描述符谓词/xmlcodec 往返（含 enum）/B2 集成测试（红）→ 加描述符 + reconciler（绿）。
3. `go test ./... -race` 全绿 + `go-code-review-check` 通过 → 提交（≤500 行，拆描述符+codec / reconciler+B2）。
4. sync delta → 主 spec；archive；PR 合入（≤1000 行）。
5. 回滚：仅新增一条描述符 + reconciler 包 + 测试，无生成物/架构改动，回退零副作用。

## Open Questions

- 深层 rule-* 的接入时机——须先集成 l3vpn（其 must 目标）+ 深层嵌套引擎支持，待 ACL 规则实际配置需求出现时按 DAG 补，本 change 不预判。
