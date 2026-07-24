---
name: bgp-2b-wave
description: 做 BGP 2b（公网邻居 AF 策略属性）前必读——多前置 change 的 DAG、序列化约束、波次①已交付
metadata: 
  node_type: memory
  type: project
  originSessionId: 07822cbe-2d47-44b8-bf34-1167f58e17cf
---

BGP 二期-2b = 为公网邻居 AF 接入可选策略属性（route-policy/route-filter/ACL group/tunnel-policy）。因 leafref `require-instance=true`，**每个属性必须先集成其目标模型**（越序禁令，见归档 change huawei-bgp-public-config 的 design.md）。见 [[bgp-onboarding-deps]]。

**DAG（写死排序，禁越序）**：
- 波次① `huawei-tunnel-management`（叶子，仅依赖 ifm✅）→ 解锁 BGP tunnel-policy + 是 routing-policy 前置
- 波次② `huawei-xpl`（叶子，仅依赖 ifm✅，23 类型）→ route-filter
- 波次③ `huawei-routing-policy`（依赖 tnlm，282 类型）→ import/export route-policy
- 波次④ `huawei-acl`（依赖 time-range+l3vpn+ni；**acl 62 类型已生成但 time-range/l3vpn 类型=0 需补 gen.conf codegen**）→ ACL group
- 波次⑤ BGP AF 策略属性增量接入（走既有 ni 描述符，零新描述符，随各前置解锁）

**关键：目标模型的 ygot 结构体一期 huawei-bgp 全闭包已"免费"生成**（tnlm/xpl/rtp/acl），故①②③集成=加描述符+reconciler+完备矩阵，**零 codegen**（不改 gen.conf/generated/）。仅④的 time-range/l3vpn 未生成、需补 codegen。

**序列化约束（重要）**：①②③④都要改**同一文件** `backend/internal/drivers/huawei.go`（加描述符）+ `huawei_test.go`（加对拍/往返）。按 TM03「多 worktree 不可改同一 package」，各波次**必须串行**（前一个 merge 后再基于新 main 开下一个），不能并行开 worktree，否则 huawei.go 冲突。用户已拍板「全部串起来做」。

**集成 playbook（每波次复用，镜像已合入的 bgp 容器根）**：
1. worktree off origin/main（yang-models 是仅构建期 submodule，fresh worktree 未 checkout；探索 YANG 从主仓库 /home/leezesi/USMP/yang-models 读，本波次零 codegen 不需要它）
2. 描述符：huawei.go 加 namespace 常量（取 YANG module namespace，Entry.Namespace() 实测空不可派生）+ Descriptor{谓词 HasPrefix "/{prefix}:{root}"，精确锚定} + SchemaTree 入口闭包
3. reconciler：镜像 `internal/controller/bgp/reconciler.go`（容器根单条整根 MODIFY 收敛防漂移），差异仅 path+GoStruct 类型；deviceClient.Get 走 DecoderFor container 模式
4. netconfsim **模型无关**（tree_datastore+RFC edit-config 整树替换），B2 零方言直接复用
5. 完备矩阵（T02b）：谓词对拍+全链路真值往返+schema 驱动形状锁定+边界+config-false 不入 editcfg+负路径+并发race+B2 端到端
6. ≤500 行/commit（commit-msg 硬门禁，超限拆分：描述符+codec 测试一个 commit、reconciler+B2 一个）

**已交付**：
- 波次① huawei-tunnel-management-config = **PR #155（已 merge，main 6d7b6c7）**。接入 `/tnlm:tunnel-management` 标量边界（tunnel-policy name+description + tunnel-down-switch/enable）。深层 ipv4/ipv6-set 推迟 follow-up。主 spec TNLM-01..05。
- 波次② huawei-xpl-config = **PR #156（CI 全绿，待 merge）**。接入 `/xpl:xpl/route-filters/route-filter`（name+content，BGP route-filter leafref 目标，huawei-bgp-common 多处引用）。xpl 其他策略 list（as-path/community/prefix/rd/large-community 等）非本 leafref 目标，推迟 follow-up。主 spec XPL-01..05。reconciler 在 `internal/controller/xpl`。

- 波次③ huawei-routing-policy-config = **PR #157（CI 跑中，待 merge）**。接入 `/rtp:routing-policy/policy-definitions/policy-definition`（name+address-family-mismatch-deny 标量边界，BGP import/export route-policy leafref 目标 huawei-bgp-common:165/551/575）。深层 `nodes/node`（conditions/actions ~1490 行）+ 其他 rtp filter 推迟 follow-up，schema 形状测试锁死。reconciler 在 `internal/controller/routingpolicy`。主 spec RTP-01..05。

**已建立的稳定 playbook 事实**（4 次跑通 tnlm/xpl/rtp，容器根模型）：容器根编解码复用 XC-05/XC-06 **零新缺口**；netconfsim 模型无关零方言；结构体一期全闭包已生成零 codegen；每次 = 1 描述符(huawei.go)+1 reconciler 包(镜像 bgp/tunnelmgmt/xpl，仅换 path+GoStruct 类型+Device.X 字段名)+完备矩阵(谓词对拍/往返/schema 形状/负路径/race/B2)；≤500 行拆 2 commit(描述符+codec / reconciler+B2)；PR ~780 行；覆盖率稳定 61.9。CI 自托管跑 ~3min，merge 由用户手动(agent 无权 merge，classifier 拦截)。

**⚠️ 波次④ acl 触发了平台级枚举 bug（重要）**：acl `groups/group/type` 是 mandatory 枚举——我的波次首个 enum-on-wire 断言，一击命中通用引擎 **YANG enumeration 编解码发整数而非值域名** 的潜伏 bug。真机按值域名校验→**已交付的 vlan/ifm 枚举字段真机静默失效**（golden 固化整数+sim 宽容+2a af-type 没断言线上名，故长期没暴露）。这正是 T02b「断言 encode 真值而非仅 sim 往返」的价值。用户拍板拆独立 PR 先修：**PR #158 xmlcodec-enum-name（yang-xml-codec ADD XC-08，CI 跑中待 merge）**——encode 用 ygot.EnumName、decode 用 ΛMap 反查名→int(兼容旧整数)、未映射退整数(R08)、重生 vlan/ifm golden、netconfsim/query.go 加 enumInt 名→int 回读(17 集成测试零改)、legacy shim TestBuild* 断言改名。**教训：以后波次边界含枚举 leaf 时，必断言 encode 输出值域名（<x>name</x>），别只信 sim 往返。**

**波次④ acl 已交付 = PR #159（CI 跑中，待 merge）**：接入 `/acl:acl/groups/group`(IPv4)+`group6s/group6`(IPv6) 标量/枚举边界(identity+type mandatory enum+match-order+step+description+number)。**零 codegen**(codegen 顾虑证伪)。首次覆盖枚举 leaf——依赖 #158 XC-08。reconciler 在 `internal/controller/acl`。深层 rule-*(l3vpn must)+ip-pools 推迟 follow-up。主 spec ACL-01..05。**至此波次⑤ 四类策略属性前置(tunnel-policy/route-filter/route-policy/ACL group)全部就位。**

**🎯 BGP 2b 全部交付完成并全部合入 main（2026-07-14，main c3ad79a）**：波次① tnlm #155 / ② xpl #156 / ③ rtp #157 / ④ acl #159 前置 + XC-08 枚举修复 #158 + **⑤ AF 策略属性 #160(收官)** —— 六 PR 全 merged。主 spec 已有 7 个华为面：huawei-{bgp,bgp-neighbor,tunnel-management,xpl,routing-policy,acl,bgp-af-policy}-config + yang-xml-codec XC-08。波次⑤=接 BGP AF `import-filter-policy` 的 acl-name-or-num(→acl)+filter-name/filter-parameter(→xpl) leafref 属性，**零生产代码**——复用 2a 的 ni 描述符+reconciler+XC-06 namespace，只填 AF 策略 leaf 经既有链路下发(B1 编码断言+B2 集成收敛)。主 spec huawei-bgp-af-policy-config(AFPOL-01..03)。

**2b follow-up（未做，registered）**：AF `ipv4-prefix-filter`(需先集成 rtp ipv4-prefix-filters 子树)、深层 route-policy(→rtp policy-definition)/tunnel-policy(→tnlm)、export-filter-policys、ipv6/vpn 其他地址族、peer 级策略；各前置模型的深层子树(tnlm ipv4/ipv6-set、rtp policy-definition nodes、acl rule-*、xpl 其他 list)。跨模型 leafref 编排(目标先配)目前靠各 reconciler 独立+文档登记,无自动事务。

**（历史-已完成）原下一步：#159 merge 后做波次⑤(peer/AF 的 import/export route-policy→rtp policy-definition、route-filter→xpl route-filter、ACL group→acl group、tunnel-policy→tnlm tunnel-policy 的可选 leafref 属性)。走**既有 ni 描述符**(2a 已证 peers/afs 在 ni 根下、单描述符覆盖)，**零新描述符/零新 reconciler**——只在 desired 填 af 的策略属性字段、经既有链路编码下发。完备矩阵重点:leafref require-instance 真值(目标实例存在才合法)、per-node namespace(策略属性带各自模块 ns)。可拆多个小 change/PR。查 huawei-bgp-common 的 af 策略属性字段名(import-policy/export-policy/route-filter/acl-group/tunnel-policy 等)。注意波次⑤改的是 ni 子树(peers/afs)不是 huawei.go,与①②③④不同文件,理论可不串行,但仍建议顺序做。

**（历史备注-已过时）原计划 #158 merge 后波次④ acl 基于新 main 恢复——acl 代码已在 huawei-acl-config worktree（描述符+controller/acl+acl 完备矩阵含枚举往返），codec 修复来自 main 后 acl 完备矩阵即转绿，走完 sync/archive/PR。(2) 再波次⑤ 接 BGP AF 属性。acl 是**零 codegen**（早前"需补 time-range/l3vpn"证伪：l3vpn 仅深层 rule-* 的 must 引用自满足、time-range 已生成；acl group 标量边界不引用二者，深层 rule-* 推迟 follow-up）。acl namespace=urn:huawei:yang:huawei-acl，路径 /acl:acl，Device.Acl，enum type mandatory(Group4Type_basic=3/Group6Type_basic=5)。

**（历史备注）原计划直接 #157 merge 后开波次④**——**最重支**：acl 62 类型已生成，但其依赖 `time-range`(143 行,零依赖)/`l3vpn`(492) 类型**未生成(=0)**，须先加 gen.conf 模块补 codegen(触发 R04 regen-and-diff，非零 codegen！与①②③不同)。BGP 引用目标查 huawei-bgp-common 的 `acl` group leafref 路径。仍改 huawei.go，串行。之后波次⑤ 接 BGP AF 属性(走既有 ni 描述符零新描述符)。
