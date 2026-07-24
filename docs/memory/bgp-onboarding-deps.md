---
name: bgp-onboarding-deps
description: 接入华为 BGP 或规划其依赖模型(routing-policy/acl/xpl/tunnel-management/network-instance 等)前必读：分期依赖 DAG、peers 藏在 network-instance augment 下的结构真相、routing 软 must 可摘除、change huawei-bgp-public-config 已就绪
metadata: 
  node_type: memory
  type: project
  originSessionId: 2bf9ef6e-6e89-4e37-8213-e80f8f902802
---

华为 BGP 接入的依赖分析(2026-07-13 完成，针对 `yang-models/network-router/8.20.10/ne40e-x8x16/`，注意 gen.conf 目标是 **8.20.10** 非 8.20.0，两目录文件集不同)。change `huawei-bgp-public-config`(openspec/changes/)四件制品已就绪校验通过，本期=公网 BGP 进程标量层，依赖=0，可 `/opsx:apply` 落地。接入沿用 [[snd-driver-registry]] playbook(gen.conf 加模块名 + 注册一条 driver.Descriptor + netconfsim 方言 + [[test-governance-military-rules]] 完备矩阵)。

**结构真相(逐字段全 augment 树核验，颠覆直觉)**：
- `huawei-bgp` 有**两个独立配置根**：① `/bgp:bgp`(模块顶层独立容器，公网进程标量，与 VLAN/IFM 同构，SchemaTree 入口预期 `HuaweiBgp_Bgp`)；② `augment /ni:network-instance/instances/instance/bgp`(bgp.yang:748)。
- **peers/afs/peer-groups 全在根②的 augment 下**(peers@3207, afs@797, peer-groups@2059)，`when name='_public_'` 区分公网 vs VPN。独立的 `/bgp:bgp/instance-processs`(658-747)**只有进程标量、无 peers**。故**公网邻居即 `network-instance/instance[_public_]/.../peers`，不独立存在** → 任何 peering(哪怕只做公网)**强制先集成 `huawei-network-instance`**(103 行，零未集成依赖，极易)。"二期=peers、三期=per-VPN"的切分作废，二者本是同一 augment 树。

**依赖强度核验**：
- **routing(rt) 是软耦合可摘除**：全模块仅 bgp.yang:1073 一处 `must "not(../route-relay-tunnel='true' and /rt:routing/.../relay-tunnel/enable='true')"`；routing 不配时 xpath 求空→约束自满足→**仅需 codegen-present，不需完整可配** → `routing`(2828)/`bfd`(4342)/`ethernet` 整条最重分支从硬路径摘除。
- **强制字段零跨模型依赖**：整 augment 树仅 2 处 `mandatory true`，无一是跨模型 leafref(peer `remote-as` 是 BGP 自有 `as-number-validate`)。所有 route-policy/route-filter/acl/tunnel 引用**均可选 leaf**。
- **本期公网 base-process 依赖=0**：全 rw 字段自包含于 huawei-bgp 主+`huawei-bgp-type`+`huawei-bgp-common` 子模块；唯一跨模型引用是 config-false 回读态(`vpn-brief-infos`→ni)。

**分期依赖 DAG(写死排序，禁越序简化——leafref require-instance 默认真，越序=设备侧非法配置)**：
```
本期 公网 BGP (依赖0) → network-instance(103,唯一硬前置) → 2a 基础邻居(依赖仅ni)
2b 可选策略属性 门控于各 leafref 目标: tunnel-management(684,仅ifm)/xpl(355,仅ifm)
   → routing-policy(2830,依赖tnlm) / acl(2248,依赖 time-range(143)+l3vpn(492)+ni)
      l3vpn 依赖 ni+rtp+tnlm+xpl
摘除: routing/bfd/ethernet (软must,仅codegen-present)
```
模型根/体量：routing-policy/xpl/acl/tunnel-management 均独立 container 根；acl 是最重分支(拉 l3vpn+ni+time-range)。已集成：ifm、pub-type、extension、system、vlan。

**分期不是简化**：2a 完整覆盖强制+非策略字段=该子能力完整交付；2b 策略属性是 YANG 可选 leaf，其延后由 leafref 正确性强制(依赖未集成时越序即非法)，非遗漏。

**Why:** 防止后续规划 BGP peers/VPN/策略时误判依赖(以为要先集成 7-10 个含 bfd 的重模型)、或漏掉"公网 peers 也在 network-instance 下"这个反直觉结构而把 network-instance 排到三期。

**genfix 确定性坑（已修，任何多-augment 闭包厂商必踩）**：加 huawei-bgp 因 augment 级联生成整个 import 闭包（~73k 行，含 network-instance/acl/bfd/routing/l3vpn/routing-policy/time-range/tunnel-management 惰性 struct，ygot 无法剪枝——排除 augment 目标会报错）。闭包里多模块 augment 同一目标（bfd/ethernet/tunnel-management→ifm）→ goyang 非确定 map 序应用 → ygot 内嵌 gzip schema blob 的 `Augmented` 数组顺序逐次漂移 → 违反 CG-01"重复生成字节一致"、R04/CG-03 regen-and-diff 永失败。基线 vlan/ifm 无此闭包故一直确定。**已修**：genfix（`backend/tools/genfix`）新增确定性 schema 规范化（gunzip→UseNumber 解析→排序 Augmented 等无序数组+对象键→固定参数重压→回填），MODIFY CG-02。**教训**：`Augmented` 是唯一非确定数组；decorate-sort 必须 key+val 绑定同排（独立 keys 切片作比较器有错位 bug，2 元素单测假性通过，被端到端 regen 揪出——弱单测是陷阱）；基线 schema blob 会一次性重规范化（合法可复现，非"字节不变"，语义等价才是保证）。

**「通用引擎其实不够通用」——BGP 揪出两处缺口（VLAN/IFM 从未触发，均被测试而非真机拦下）**：
1. **genfix schema 确定性**（组 1B，已修 commit cbf3c8f）——多-augment 闭包，见上。
2. **xmlcodec 容器根**（组 3.4，已修 commit 6113fe1，加 XC-05）——`pkg/yang-runtime/xmlcodec` 原是 **list 中心**设计（VLAN/IFM 皆 list 根 `/vlan/vlans`、`/ifm/interfaces`，根有 map 字段）。BGP 公网 `/bgp:bgp` 是**容器根**（仅标量+子容器，无根 list），`Encode` 直接报 `container HuaweiBgp_Bgp has no list map field`。已加 `findContainerMap` 择模式 + `encodeContainer`/`decodeContainer`（复用 encodeFields/decodeStruct）。**教训**：接容器根模块（BGP/system 类）前知悉引擎已支持；但**容器根 delete 仍未支持**（EncodeDelete 对容器根返干净错误、待 group 4 补 BGP 删除语义）。**质量方法**：往返真值测试（RFC7951→XML 带 namespace→回读等价，断言真值非仅非空）是揪出这类静默缺陷的关键——只测"非空"会漏。

**namespace 坑**：BGP 用 `urn:huawei:yang:huawei-bgp`（8.20.10 YANG 声明的权威 module namespace，用户拍板）。⚠️ VLAN/IFM 驱动用的是旧式 `urn:huawei:params:xml:ns:yang:huawei-<mod>`（与 8.20.10 YANG 不符，潜在真机隐患）；且 **namespace 正确性全测试套件测不出**（sim namespace-宽容、decode 按 local 名不校验 namespace）——只 encode 发给真机才校验。谓词用 `HasPrefix("/bgp:bgp")`（比 Contains 精确，排除 bgp-flow:/bgp-evpn:/per-VPN）。

**schema 驱动完备性测试（组 4 核心方法，防"全属性可配"漏字段）**：`pkg/yang-runtime/xmlcodec/bgp_roundtrip_test.go` 反射+schema 继承枚举 `/bgp:bgp` 下**每个 config-true 标量 leaf**自动赋值→编码→解码→整体 DeepEqual + 计数断言（本期恰 **29** 个）。当场揪出两处"简化=遗漏"：(1) `default-parameter` 我 spec 误列 rw、实为 config-false（schema config 继承为准，别信 YANG 源直觉）；(2) `confederation/as` 是 **leaf-list `[]string`**（多子 AS），手列会漏。教训：手写 fixture 必漏，**schema 驱动枚举 + 计数断言**才是完备性防线，模型加字段即触发复审。config-true 标量集=global 2+base-process 直属 13+confederation 3(含 as leaf-list)+graceful-restart 4+reference-period 3+timer 4。`paf-controls` 是 `global` 同级 **list**（非其下标量）、`instance-processs` 多进程 list——本期均不接。**容器根 delete 债**：EncodeDelete/parseDeleteTarget 对 BGP 均干净报错非 panic；MVP 禁用走 enable=false modify，node-delete 后续单列（group 4b）。

**第 5 处 list 中心缺口 = reconcile 写路径（组 5，已修 D8，commit f632d84）**：BGP 配置光有描述符+编解码**还到不了设备**，需 reconciler（`internal/controller/bgp`，copy ifm 模式）+ main.go 注册控制器。B2 集成（下发→回读→二次收敛）当场揪出：`diff.walkStruct` 递归进容器根、对每个顶层子容器各发一条 change（value=子容器指针），而 `client.XMLEncoderForValue` 只登记根类型 `*HuaweiBgp_Bgp` → 匹配不到子容器 → 落 `xml.Marshal` 兜底发 **Go 类型名** `<HuaweiBgp_Bgp_BaseProcess>`（非 `<bgp>`），回读解不出→永久漂移。**D8 修复**：BGP `diffEngineAdapter.Diff` 检出任一漂移即**收敛为单条整根 change**（下发整个 desired `/bgp:bgp`，经描述符 xmlcodec container 编码为单条 `<bgp>…`，edit-config merge 收敛），最小侵入放 BGP adapter、不动共享 diff/client（VLAN/IFM list 路径零回归）。netconfsim 无需 BGP 方言（通用 tree datastore 按 local 名存取）。

**BGP 至此暴露 5 处「通用引擎其实是 list 中心」缺口**（全 VLAN/IFM 从未触发、容器根+多-augment 才暴露、均被测试拦下）：①genfix schema 确定性 ②xmlcodec 容器根编解码 ③完备性 config 判定 ④完备性 leaf-list ⑤reconcile 写路径容器根收敛。接下一个容器根模块（system 类）前预期还会有类似缺口，走 B2 集成能拦。

**状态（2026-07-13）——公网 BGP MVP 已交付**：feature **PR #146 已合入 main**（sha 2ae4ec1；gen+描述符+xmlcodec容器根 XC-05+reconciler+B2集成+完备矩阵，`go test -race ./...` 30 包全绿、覆盖率 60.6→61.0，公网 BGP 读写闭环真在模拟网元跑通）。收尾 **PR #147（sync+archive）已开、CI 绿待 merge**：三 delta 合主 spec（主 spec 落 `openspec/specs/huawei-bgp-config`(新)/`yang-xml-codec` XC-05/`yang-codegen-pipeline` CG-02），change 已归档 `openspec/changes/archive/2026-07-13-huawei-bgp-public-config`。**推迟债（文档登记，非阻塞）**：容器根 node-delete(MVP 禁用走 enable=false modify)、sim namespace 严格校验、diffEngineAdapter 容器根收敛通用化。**下一步分期**：二期 peers/AF 前先集成 network-instance（peering 唯一硬前置），策略属性再门控 routing-policy/xpl/acl/tunnel-management——依赖 DAG 见归档 change 的 design.md。

**状态（2026-07-13）——network-instance 硬前置已交付，解锁 2a 基础邻居**：change `huawei-network-instance-config` 两 PR **已合入 main**（#148 spec-first 制品 + #149 实现 sha dcade60；gen.conf+单描述符+networkinstance reconciler 容器根收敛+B2 集成+完备矩阵，覆盖率 61.0→61.6），sync+archive 收尾另一 PR（主 spec 落 `openspec/specs/huawei-network-instance-config`(新)，change 归档 `archive/2026-07-13-huawei-network-instance-config`）。接入的是 `/ni:network-instance` **原生 config-true 面**（global 3 标量 + instance name/description，共 5 个，schema 枚举计数断言）。**四个实证结论（都靠往返真值/集成测试非臆断，写进 design D1/D2/D5b/D6）**：① `huawei-network-instance` 的 Go 类型早在 BGP augment 闭包内生成，加 gen.conf 仅头注释漂移；② 生成的 `instance` struct 是**多模块 augment 共享合并点**（原生 Name/Description + `Bgp`(huawei-bgp) + `Afs`/`Parameter`/`TrafficStatisticEnable`(huawei-l3vpn)）→ `/ni:network-instance` 是**单一 ygot 根**，2a peers 是往 `Instance.Bgp` 填、**扩展同一描述符**（谓词 `HasPrefix("/ni:network-instance")` 已覆盖 per-VPN bgp 路径），非另立描述符；③ **嵌套 list（instances/instance 挂容器根子容器下）编解码零缺口**，XC-05 无需扩展、无第 6 处 list-中心缺口；④ 「只做 config-true」收窄的是**驱动/测试面非生成/解码面**——config-false 字段仍生成仍可 decode，且 YANG 禁 config-true leafref→config-false，故不影响任何跨模块依赖。**推迟债**：config-false 回读态（sys-router-id/vrf-id）、设备侧 instance 删除（声明式 subset 语义天然不删含 `_public_`，删除须走 DELETE 命令通道，沿用 BGP node-delete 债）、per-node namespace（2a 往 Instance.Bgp 填 peers 时 augment 节点须带 huawei-bgp namespace）。**2a 现可开工**：peers/remote-as/af-type，依赖仅 network-instance（已交付），所有跨模型引用均可选 leaf、唯一 mandatory 是 BGP 自有类型 remote-as。

**状态（2026-07-13）——per-node namespace XC-06 + 2a 公网基础邻居均已交付，2b 可规划**：
- **XC-06 per-node namespace**（change `xmlcodec-per-node-namespace`，PR #151 实现 + #152 sync/archive **已合入 main**）：通用引擎按字段 `module` tag 在模块边界发 `xmlns`（`Spec.Namespaces` map），支持 augment 跨模块树。**硬前置原因**：ni 根下 huawei-bgp 子树须带 huawei-bgp namespace，否则真机拒绝；**sim/decode namespace-宽容测不出**，正确性靠 **encode 输出 namespace 真值断言**。单模块树（vlan/ifm/bgp/system/ni）`nsResolver.at()` 返"" → golden 逐字节零回归。
- **2a 公网基础邻居**（change `huawei-bgp-neighbor-config`，PR #153 实现 + #154 sync/archive）：接入 `instance[_public_]/bgp/base-process/peers/peer`（key=address）**41 config-true 标量**（remote-as/description/connect-mode/tcp-mss/password/tracking… + timer/graceful-restart/bfd-parameter 子容器）+ `afs/af` 的 af-type 枚举 key。**核心洞察成立=复用 ni 描述符/reconciler 零新描述符/控制器/main.go 改动**（D1，peer 填上即经既有链路+XC-06 下发；B2 实测下发报文混合 namespace 正确）。
- **apply 期三实测发现（往返真值/B2，非臆断）**：① **YANGEmpty（YANG empty 类型，非指针 bool）引擎缺口→修 XC-07**：peer/bfd-parameter/compatible 首个 empty 类型驱动字段，既有引擎报 `unsupported field form bool`（VLAN/IFM/BGP/ni 无 empty 故从未触发；「通用引擎其实是 XX 中心」缺口序列又一处）——encode 发 `<tag/>`(true)/不发(false)、decode 存在即 true，零回归；② **af-type 枚举 key + 深层嵌套无缺口**（list-under-list 直接跑通）；③ **remote-as 事实校准**：YANG 声明 `mandatory true` 但 **ygot ΛValidate 不强制** list-entry mandatory leaf → 设备/API 层须兜底（测试锁定防臆断）。
- **推迟债**：peer/af 删除沿用声明式 subset（天然不删）+ DELETE 命令通道；choice（peer 有 `load-balancing-ignore-as-type` 等 choice，[[yang-constraint-engine]] 记需构建期 codegen）。
- **2b 可规划**（af 内策略属性：import/export route-policy、route-filter、ACL group、tunnel-policy）：各门控于其 leafref 目标模型集成——`tunnel-management`/`xpl`(仅 ifm) → `routing-policy`(依赖 tnlm) → `acl`(依赖 time-range+l3vpn+ni)，各自独立 change 可并行；未集成前禁接对应属性（leafref require-instance 致设备侧非法）。

**Why:** 防止后续规划 BGP peers/VPN/策略时误判依赖(以为要先集成 7-10 个含 bfd 的重模型)、或漏掉"公网 peers 也在 network-instance 下"这个反直觉结构而把 network-instance 排到三期；并防再踩多-augment 闭包 schema 非确定性坑。

**How to apply:** 动 BGP peers/AF 或规划 routing-policy/acl/xpl/tunnel-management/network-instance 接入前，先按此 DAG 定序；先集成 network-instance 才能做任何 peering；routing 只需可解析别当可配面做。接任何多-augment 闭包厂商前知悉 genfix 已处理 schema 确定性。

**⏭️ 当前进度与下次入口（2026-07-13 收尾）**：一期公网 BGP + 2a 前置(ni + XC-06) + 2a 公网基础邻居 peers/AF **全部已合入 main 并归档**（见上两个状态块）。**下一步 = 2b af 内策略属性**（import/export route-policy、route-filter、ACL group、tunnel-policy）。**2b 最轻入口 = 先接 `tunnel-management` 或 `xpl`**（只依赖已集成的 ifm），再逐级解锁 `routing-policy`(依赖 tnlm) → `acl`(依赖 time-range+l3vpn+ni)；各策略属性门控于其 leafref 目标模型已集成，未集成前禁接（require-instance 致设备侧非法）。**恢复方式**：新会话说「继续 BGP 2b」即可，我读本 memory + 归档 `openspec/changes/archive/2026-07-13-huawei-bgp-neighbor-config/design.md` 接上；沿用 [[snd-driver-registry]] playbook（gen.conf 加模块 + 复用 ni 描述符驱动面扩展 + XC-06 namespace + XC-07 empty 类型 + 完备矩阵）。新起 worktree（§6），别在 main 开发。
