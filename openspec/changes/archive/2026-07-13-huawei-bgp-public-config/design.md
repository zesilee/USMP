## Context

USMP 的 Stack B（yang-controller-runtime）已通过 P5「驱动注册表 + 通用 XML 编解码引擎」把「加设备模型」收敛为**注册一条 `driver.Descriptor` + gen.conf 加模块名**的数据操作，VLAN、IFM 是两个已交付范例。本 change 用同一 playbook 接入 BGP，但 BGP 模型结构与 VLAN/IFM 有本质差异，须先厘清：

- `huawei-bgp.yang`（8.20.10 版 5806 行）有**两个独立配置根**：
  - **ROOT #1 `/bgp:bgp`** —— 模块顶层**独立**容器（公网 BGP），结构上与 `huawei-vlan → /vlan/vlans`、`huawei-ifm → /ifm/interfaces` 同构，**不依赖 network-instance**。
  - **ROOT #2 `/ni:network-instance/instances/instance[name]/bgp`** —— 由 `huawei-bgp` 通过 `augment` 挂到 `huawei-network-instance` 下的 per-VPN BGP，且带 `when "/bgp:bgp/bgp:base-process/bgp:enable='true'"` —— **公网未 enable 时 per-VPN 分支不生效**。
- 所有 feature 模块（evpn/flow/l2vpnad/link-state…）都 `augment` 到 ROOT #2 的地址族层。
- `huawei-network-instance.yang` 本体仅 ~100 行（`instance[name]` + name/description/router-id），per-VPN `bgp` 容器并不在该文件内，而是 `huawei-bgp` augment 进去的。

因此公网 `/bgp:bgp` 是所有后续分期（peers/AF → per-VPN → feature）的**共同地基**，且自包含、零 network-instance 依赖，是最小可闭环 MVP。

## Goals / Non-Goals

**Goals:**
- 接入 `/bgp:bgp/global` + `/bgp:bgp/base-process` **标量层**（含 confederation/graceful-restart/timer 小容器）的读改下发闭环，SchemaTree 入口 `HuaweiBgp_Bgp`。
- 沿用 VLAN/IFM playbook：gen.conf 加 `huawei-bgp` → `make gen-yang` 生成 → 注册一条 `driver.Descriptor` → netconfsim BGP 方言 → `yang-config-test-design` 完备矩阵。
- **零 XML 胶水代码**：encode/decode/键式 delete 全部由 `pkg/yang-runtime/xmlcodec` 按 ygot 数据驱动。
- 证明 P5 架构「加模块=注册数据、通用引擎/管线 spec 不变」这一论断在 BGP（体量最大、结构最复杂的模型）上成立。

**Non-Goals（分期，明确排除）:**
- `base-process/instance-process`（peers、地址族 af）与 `instance-processs` list —— 二期。
- per-VPN BGP（ROOT #2，`augment /ni:.../instance/bgp`）与 `huawei-network-instance` 配置面 —— 三期。
- 全部 feature 模块（evpn/flow/l2vpnad/link-state/lsp/mdt/mvpn/srpolicy/srv6/vpntarget…）—— 四期+，各自独立 change。
- BGP 只读状态/遥测（`huawei-bgp-routing-table` config-false、`huawei-bgp-notification`）—— 不在配置面范围。
- 前端新增硬编码 BGP 表单 —— 由通用「模块控制台」按 YANG 自动渲染（R05）。

## 依赖分析（商用交付关键：不简化，依赖先交付）

逐字段核对 MVP 区（`/bgp:bgp/global` 442–457 + 整个 `base-process` 472–657，8.20.10）的每个类型、leafref、when/must，及所 `uses` 的 7 个 grouping 体内引用。区分两类依赖：

**① 编解码/codegen 期依赖（须可解析，非"集成"）**：`huawei-bgp` header import 的 8 个模块 + common import 的 acl，须在 `yang-models/network-router/8.20.10/ne40e-x8x16/` 存在以供 ygot 解析。已核实全部在目录内 → `make gen-yang` 不因此阻塞（唯一变数是 R1 augment 剪枝）。**文件在目录 ≠ 该模型已作为可配置面集成**——二者不可混同。

**② 功能性配置依赖（被引模型须 USMP 可配，BGP 配置才合法/可用）**：leafref（`require-instance` 默认为真）按名引用他模型实例时，被引实例须存在于数据存储；在无数据库、模型驱动平台上即"该模型须由 USMP 可配"。核对矩阵：

| 被引模型 | MVP 引用点 | 性质 | 二期(peers/AF)引用 | 集成状态 | 结论 |
|---|---|---|---|---|---|
| `pub-type` | 类型(id-range 等) | rw | 是 | ✅ 已集成 | 就绪 |
| `ifm` | **无** | — | peer update-source 接口 | ✅ 已集成 | 就绪 |
| `network-instance`(ni) | 仅 `vpn-brief-infos`(**config false** 回读态)leafref→instance/name | **只读** | — | ⚠️ 未集成 | **非 MVP 依赖**；三期 per-VPN 才需其可配 |
| `routing-policy`(rtp) | **无** | — | **import/export route-policy by name**(common:165/551…) | ❌ 未集成 | **二期前置** |
| `xpl` | **无** | — | route-filter(common:528/1424…) | ❌ 未集成 | **二期前置** |
| `acl` | **无** | — | ACL group(common:1089…) | ❌ 未集成 | **二期前置** |
| `tunnel-management`(tnlm) | **无** | — | tunnel-policy(bgp:4628) | ❌ 未集成 | **二期前置** |
| `routing`(rt) | **无** | — | 一条 must(bgp:1073) | ❌ 未集成 | **二期前置** |

**结论 1（MVP 干净）**：MVP 区全部**可配(rw)**字段类型自包含于 `huawei-bgp` 模块（主 + `huawei-bgp-type` + `huawei-bgp-common` 子模块，同 namespace）；when/must 全为节点内 xpath；唯一跨模型引用是 `vpn-brief-infos`/`*-status`/`*-info` grouping 内的 **config-false 回读态**。故公网 `/bgp:bgp/base-process` 功能性配置依赖 = **零**，可独立完整交付，不"先欠"任何模型。

**结构性发现（修正早前分期，经全 augment 树核验）**：`huawei-bgp` 的 **peers / afs(地址族) / peer-groups 全部位于 `augment /ni:network-instance/instances/instance/bgp/base-process` 下**（peers 在 bgp.yang:3207、afs:797、peer-groups:2059），由 `when name='_public_'` 区分公网 vs VPN。独立的 `/bgp:bgp/instance-processs`（658-747）**只有进程级标量，不含 peers**。故 **公网邻居并不独立存在——它就是 `network-instance/instance[name='_public_']/bgp/.../peers`**。含义：任何 peer/AF 配置（哪怕只做公网）**强制先集成 `huawei-network-instance`**；早前"二期=peers、三期=per-VPN"的切分作废，二者本是同一 augment 树。

**依赖强度核验（决定硬路径长度）**：
- **routing(rt) 是软耦合**：BGP 全模块仅一处引用 `rt:`（bgp.yang:1073 的 `must "not(../route-relay-tunnel='true' and /rt:routing/.../relay-tunnel/enable='true')"`）。routing 未配时该 xpath 求空 → `and` 假 → `not` 真 → 约束自满足。故 routing **仅需 codegen-present（schema 可解析），无需完整可配面** → `routing`(2828)/`bfd`(4342)/`ethernet` 整条最重分支**从硬依赖路径摘除**。
- **强制字段零跨模型依赖**：整个 augment 树（公网+VPN 全部 peers/AF）仅 2 处 `mandatory true`，**无一是跨模型 leafref**（peer 的 `remote-as` 是 BGP 自有 `as-number-validate` 类型）。所有 route-policy / route-filter / ACL / tunnel-policy 引用**均为可选 leaf**。

**结论 2（修正后分期依赖 DAG，写死排序，禁止越序简化）**：
```
[已集成] ifm, pub-type
本期: 公网 /bgp:bgp/global+base-process+instance-processs 标量  (功能依赖=0, 立即可做)

    ┌─ network-instance (103行, 零未集成依赖, 极易) ── 唯一硬前置，解锁所有 peering
    ▼
二期-2a: peer/AF 强制+基础字段(address, remote-as, af-type, peer 标量/timer/GR)
        依赖 = 仅 network-instance；完整交付"基础 BGP 邻居"，无策略属性
        (routing 软 must 仅需 codegen-present，不阻塞)

二期-2b: peer/AF 可选策略属性(route-policy/route-filter/acl/tunnel 引用)
        各属性门控于其 leafref 目标模型集成（可选增强，非基础邻居阻塞）：
        ├─ tunnel-management(684, 仅 ifm✅)   ┐ 各自独立 change，可并行
        ├─ xpl(355, 仅 ifm✅)                 │
        ├─ routing-policy(2830, 依赖 tnlm)    ┘
        └─ acl(2248) ─依赖→ time-range(143,零依赖) + l3vpn(492) + network-instance
                             l3vpn ─依赖→ ni + routing-policy + tunnel-management + xpl
[摘除] routing/bfd/ethernet：软 must，仅 codegen-present
```
- **越序禁令**：2b 的某个策略属性 SHALL NOT 在其 leafref 目标模型（rtp/xpl/acl/tnlm）集成前接入——leafref `require-instance` 默认真，会生成设备侧非法配置。此为"简化=遗漏"的具体形态。
- **非阻塞澄清**：2a（基础邻居）不受 rtp/xpl/acl/tnlm 阻塞，因这些引用全可选；2a 完整覆盖强制+非策略字段即为该子能力的完整交付（非简化）。唯一硬前置是 `network-instance`（极轻）。

**完整性纪律**：本期范围 = MVP 子树**全部 rw 字段无遗漏**，禁止字段挑选（proposal "全部可配字段" / spec BGP-01 "覆盖全部"）。config-false 只读态不进写用例，但须在测试矩阵负路径确认其不被误当作可配下发。

## Decisions

### D1：MVP 切公网 `/bgp:bgp/base-process` 标量层，而非 per-VPN 或含 peers
- **选择**：接入面 = ROOT #1 的 `global` + `base-process` 标量 leaf + confederation/graceful-restart/timer 小容器；排除 `instance-process`（peers/AF）与 ROOT #2。
- **理由**：(1) `/bgp:bgp` 是独立根，零 network-instance 依赖，与 VLAN/IFM 完全同构，复用现成 playbook；(2) per-VPN BGP 受 `when(公网 enable=true)` 门控，公网层是必经地基，先做它无返工；(3) `instance-process` 是 peers/AF 深嵌套 list，体量与测试矩阵（并发/嵌套/幂等/负路径）成倍膨胀，会撞 §5.3 体积军规，拆到二期。
- **备选**：① 一期含 peers/AF —— 弃，体量超限、且 MVP 价值（进程 up + AS 号）已足以验证链路；② 一期直接 per-VPN —— 弃，须先解决 augment 生成 + network-instance 纳管，风险与体量双高，且仍依赖公网层先能配。

### D2：gen.conf 只加 `huawei-bgp`，不加 `huawei-network-instance`
- **选择**：`backend/internal/generated/huawei/gen.conf` 的 `modules` 追加 `huawei-bgp`（保持 8.20.10 目标目录），**不纳入** `huawei-network-instance`。
- **理由**：MVP 只用 ROOT #1 独立根；network-instance 仅 ROOT #2（三期）才需要。少一个生成模块=少一片生成物 + 少一批 import 闭包风险。
- **前置假设（须 spike 验证，见 R1）**：ygot 解析 `huawei-bgp` 时遇到 `augment /ni:...`（目标模块未在生成集），能**干净跳过**只生成独立 `/bgp:bgp`。若证伪，回退方案见 R1 Mitigation。

### D3：一条 `driver.Descriptor`，谓词避开 feature 模块误匹配
- **选择**：`Descriptor{Vendor:"huawei", Module:"bgp", ControllerToken:"bgp"}`，`Namespace` 显式常量 `HuaweiBgpNS = "urn:huawei:yang:huawei-bgp"`，`Schema: func() *yang.Entry { return huawei.SchemaTree["HuaweiBgp_Bgp"] }`。
- **谓词设计（关键坑）**：VLAN/IFM 用 `strings.Contains(p, "vlan:")` 这类子串匹配。BGP 前缀 `bgp:` 会被 `bgp-flow:`、`bgp-evpn:` 等 feature 模块前缀**误命中**（它们 namespace 是 `huawei-bgp-flow` 等，prefix `bgp-flow`）。故 `MatchRoute`/`MatchEncode` 须锚定 `bgp:bgp`（根容器路径）或精确前缀，`MatchDecode` 用 `bgp:bgp`。谓词的精确边界须由单测（正/负路径，含 bgp-flow/bgp-evpn 负样本）钉死。
- **理由**：与 DR-01/DR-03 一致，零 XML 代码；注册序追加在 system/vlan/ifm 之后（先注册先匹配，BGP 与三者路径不重叠，序无碰撞）。
- **注册可达性**：注册靠空白导入 `internal/drivers` 触发——须确认新增的 BGP 集成测试包/二进制若独立，也带该空白导入，否则注册表为空→编码落 `xml.Marshal` 兜底对 map 报错（历史 actor 翻车教训）。

### D4：SchemaTree 入口名以生成产物为准
- **选择**：`gen.conf` 现为 `compress_paths=false` + `generate_fakeroot=true`，公网根容器预期生成 `HuaweiBgp_Bgp`；但**确切入口 key 与结构体命名以 `make gen-yang` 实际产物为准**，spike 完成后在 tasks 中锚定，不提前臆断。

### D5：config-false 字段不按 schema 过滤
- 沿用 xmlcodec 既定语义：华为模型把部分在发字段标 `config false`，通用引擎**不按 schema 过滤**（按 schema 过滤会破坏行为等价）。BGP base-process 若有 config-false 在发字段，同样保留，由 golden 对拍锁定。

### D6：netconfsim BGP 方言 + golden 方法论
- netconfsim 增加 BGP edit-config（整树替换语义，对齐既有 RFC edit-config 通道）+ get-config 回读；改编码语义前用 `xmlcodec.Canonicalize` 冻结现状、逐 fixture 对拍，fixture/golden 落 `internal/testutil/hwfix`。

## Risks / Trade-offs

- **[R1 — 唯一能推翻 D2 的风险] ygot 对未生成 augment 目标的处理未知** → **Mitigation**：tasks 第一项即红灯 spike——`make gen-yang`（modules 含 `huawei-bgp`、不含 network-instance）跑通并检查产物：(a) 干净跳过 → 确认 D2，锚定 `HuaweiBgp_Bgp` 入口名；(b) 报错/警告 → 回退：把 `huawei-network-instance` 加入 `modules` **仅供 augment 解析**，但描述符仍只路由/编解码 ROOT #1 `/bgp:bgp`，并补 `yang-codegen-pipeline` delta 记录该模块新增。spike 结论回填 design.md 与 proposal 的 Modified Capabilities。
- **[R2] 传递 import 闭包不完整致生成失败** → **Mitigation**：`huawei-bgp` 一级 import（network-instance/ifm/pub-type/extension/routing-policy/xpl/routing/tunnel-management）+ common 的 acl 已核实存在于 8.20.10 目录；二级 import（routing-policy→routing-policy-type/acl/ext、xpl→…）在同一 spike 中随 `make gen-yang` 暴露，缺失则从 8.20.10 目录补齐路径或补 import 引用，不手写 stub。
- **[R3] BGP 前缀 `bgp:` 子串误匹配 feature 模块** → **Mitigation**：D3 谓词锚定 `bgp:bgp`；单测强制含 `bgp-flow:`/`bgp-evpn:` 负样本，证明不误命中。
- **[R4] 生成物体积撞 R04 regen-and-diff 门禁** → **Mitigation**：生成物改动合法当且仅当 `make gen-yang` 零漂移；BGP 结构体量大，须确认 pr-size/commit-msg 已排除 `generated/` 目录（memory 记为已排除，实施时复核）；勿手改 generated/。
- **[R5] 注册可达性（空白导入缺失）致注册表为空** → **Mitigation**：D3 已列，新增独立测试包/二进制补空白导入 `internal/drivers`。
- **[Trade-off] 一期能配的字段少（进程级标量）** → 接受：MVP 目标是打通链路与验证架构，peers/AF 的业务价值在二期兑现；分期路线在 proposal 与 spec 中明确，避免「看起来接了 BGP 其实不能配邻居」的误解——spec 显式登记「本期不含 peers/AF/per-VPN」。

## Migration Plan

无数据迁移（无数据库，R03）。部署即生效：`make gen-yang` 重生成 → 描述符随二进制编译期注册 → netconfsim 支持 BGP 方言。回滚 = 撤销 gen.conf 模块名 + 描述符 + 重生成（生成物回退）。分期演进（二期 peers/AF、三期 per-VPN、四期 feature）各自独立 change，互不阻塞。

## Spike 结论（2026-07-13，worktree 内实测，消解 R1/R2）

`make gen-yang VENDOR=huawei`（modules 加 `huawei-bgp`，不含 network-instance）实测：

- **R2 消解**：import 传递闭包完整可解析，生成成功（all.gen.go 73543 行），`go build ./...` 全通过。8.20.10 目录二级依赖无缺失。
- **R1 消解（第三种结局，非二选一）**：ygot **既不报错、也不"干净剪枝"**，而是**物化整个 import 闭包**——除 `HuaweiBgp_Bgp`（公网根，D4 入口名确认）外，还生成了 `HuaweiNetworkInstance`/`HuaweiAcl`/`HuaweiBfd`/`HuaweiL3Vpn`/`HuaweiRouting`/`HuaweiRoutingPolicy`/`HuaweiTimeRange`/`HuaweiTunnelManagement` 等兄弟根 struct（因 huawei-bgp augment 进 network-instance，ygot 必须物化 augment 目标及其级联）。
- **D2 结论修正**：`modules` 仍只需加 `huawei-bgp`（无须显式列 network-instance）——**但后果是自动生成 ~73k 行完整闭包**，而非早前假设的"仅 /bgp:bgp"。这不破坏方案：配置面仍由描述符仅锚定 `/bgp:bgp` 收口，闭包中其他模型 struct 是**惰性生成的 Go 类型**（R04，勿手改），无描述符即无配置通道。
- **副产物洞察（利后续分期）**：二期/2b 依赖模型（network-instance/routing-policy/acl/tunnel-management 等）的 **ygot 结构体本次已"免费"生成**；后续分期只需加**功能性描述符**，codegen 侧零新增（除非要拆分生成边界）。
- **待运行期确认（挪至任务 3/4 描述符接线时验证，非阻塞）**：`SchemaTree["HuaweiBgp_Bgp"]` 运行期解析——与 VLAN `SchemaTree["HuaweiVlan_Vlan_Vlans"]` 同机制（每个生成 struct 均注册 schema 条目），高置信可用，描述符接线时实测。
- **新权衡已决（用户拍板 2026-07-13：接受全闭包生成）**：本 change 引入 ~73k 行生成物（含尚未功能集成的 network-instance/acl/bfd/routing/l3vpn/routing-policy/time-range/tunnel-management 惰性类型）。闭包是 BGP augment 结构固有、ygot 无法剪枝（排除 augment 目标会致 R1 报错）。生成物受 R04 regen-and-diff 管控、排除于 pr-size 门禁（R4），惰性类型无描述符=无配置通道。**强制缓解（防误判）**：proposal/spec 显式登记"这些闭包 struct 是 generated-but-not-integrated，仅 `/bgp:bgp` 有功能描述符；勿因类型存在而误判 acl/routing/bfd 等已集成"。副产物利好：后续分期这些模型 codegen 侧零新增。

## G2 落地暴露的门禁阻塞与 genfix 确定性修复（2026-07-13，worktree 实证）

正式 `make gen-yang`（+huawei-bgp）后发现**生成非幂等**：连续两次生成 `all.gen.go` 差 30323 行，差异**纯在 gzip schema blob（`var ySchema`）字节**（无结构性差异；`sort_keys` 后仍差 → 数组顺序非确定）。基线（vlan/ifm/system/pub-type/extension）md5 两次一致，**确定性只被 BGP 闭包打破**。

- **根因定位**：非确定数组 = `yang.Entry.Augmented`。多模块 augment 同一目标（如 `/ifm:ifm/.../interface` 被 tunnel-management 加 `tunnel-protocol`、ethernet 加 `ethernet`、bfd 等），goyang 以非确定 map 迭代序应用 augment → ygot 序列化进 `Augmented` 数组的元素顺序逐次不同 → gzip 字节漂移。基线无此多-augment 闭包，故一直确定。ygot generator 无 sort/determin flag。
- **违反的既有契约**：CG-01 已要求"重复执行输出字节一致"——BGP 闭包使其失守，CG-03 regen-and-diff 门禁将永远 fail。故这是**硬阻塞**，须先修 genfix 才能落 G2（用户拍板：扩展 genfix 规范化 schema blob）。
- **D7 修复设计（genfix 确定性 schema 规范化，CG-02 扩展）**：genfix 新增一步——定位 `ySchema` 字节数组 → gunzip → JSON 以 `json.Decoder.UseNumber()` 解析（**关键：避免 float64 重格式化破坏数字字面量**）→ 递归对**语义无序**集合数组（首要 `Augmented`）按元素规范化内容排序、对象键排序 → 固定参数 gzip（ModTime=0、固定压缩级别，机器无关）→ 按确定格式回填字节数组（后续 gofmt 稳定）。
- **正确性护栏（TDD 断言）**：(1) 规范化后 `ygot.GzipToSchema` 仍成功且解出 schema 与规范化前**语义等价**（键集合/类型/约束不变）——证明只重排无序集合、未损结构；(2) 基线生成物经新 genfix **字节不变**（无 Augmented 漂移的包不受影响）；(3) +bgp 后 regen×2 字节一致。
- **风险 [R6]**：若 `Augmented` 之外还有非确定数组 → 1B.4 regen×2 若仍漂移则定位下一个数组纳入排序（迭代收敛）。**Mitigation**：先只排序确认的 `Augmented`（最小语义风险），验证后按需扩展；绝不盲排可能有序的构造（enum 值/ordered-by-user）。
- **风险 [R7]**：genfix 是**共享工具**（所有厂商生成走它）→ 改动须保证对存量 vlan/ifm 生成物零影响（护栏 2 覆盖）。

## 组 5 暴露的第 5 处 list-中心缺口：容器根 reconcile 写路径（2026-07-13 实测）

B2 集成测试（下发→回读→二次收敛）实测发现：BGP reconcile 写路径产出错误 XML——edit-config 发的是 **Go 结构体类型名** `<HuaweiBgp_Bgp_BaseProcess>`（而非 `<bgp><base-process>`），且按顶层子容器拆成多条 edit-config。

- **根因链**：`diff.walkStruct` 递归进 BGP 容器根、对每个顶层子容器（BaseProcess/Global…）各发一条 change，change value 是**子容器指针** `*HuaweiBgp_Bgp_BaseProcess`。VLAN/IFM 的根只有一个 list-map 字段 → diff 发**整个 map**作为 change value，`client.XMLEncoderForValue` 按 map/根类型匹配到描述符、经 xmlcodec 编码。但 BGP 的子容器类型**未注册**（描述符只登记根 `*HuaweiBgp_Bgp`）→ `XMLEncoderForValue` 未命中 → 落 `xml.Marshal` 兜底 → 发 Go 类型名。回读又因存的是类型名、解码器找不到 `<bgp>` 根 → actual 空 → **永久漂移**。
- **D8 修复（容器根 diff 收敛为单条整根 change）**：BGP `diffEngineAdapter.Diff` 在细粒度 diff 检出**任一**漂移时，收敛为**一条整根 change**（`NewValue=desired 整个 *HuaweiBgp_Bgp`）。`XMLEncoderForValue(*HuaweiBgp_Bgp)` 命中描述符 → xmlcodec container 模式编码为 `<bgp xmlns=NS><base-process>…</base-process><global>…</global></bgp>` 单条 edit-config；NETCONF merge 语义使其收敛；二次对账 desired==actual（回读无损，完备性测试已证）→ 0 change。
- **为何 localized 到 BGP adapter**：这是容器根模块的通用需求，但放在 BGP reconciler 的 adapter 里最小侵入、不动共享 diff/client（避免影响 VLAN/IFM list 路径）。通用化（框架层识别容器根并收敛）是后续可选重构，非本期必需。
- **代价/权衡**：整根下发丢失 per-field change 粒度（审计只见"BGP 配置变更"而非逐字段）；对 edit-config merge 语义无碍，收敛正确性由二次对账断言保证。可接受。

**至此 BGP 暴露 5 处「通用引擎其实是 list 中心」缺口**：genfix schema 确定性、xmlcodec 容器根编解码、（完备性测试揪出的）config 判定+leaf-list、reconcile 写路径容器根收敛。全部 VLAN/IFM 从未触发、BGP 容器根+多-augment 才暴露、且**均被测试而非真机拦下**——印证「简单场景交付的通用未必真通用」。

## Open Questions

- ~~**[阻塞 D2/D4，spike 消解]** ygot 对未生成 augment 目标的确切行为？`HuaweiBgp_Bgp` 是否为实际入口 key 名？~~ **已解**（见「Spike 结论」）：物化全闭包、`HuaweiBgp_Bgp` 确认、闭包生成权衡用户接受。
- **[非阻塞]** base-process 下是否存在 config-false 在发字段需 golden 特判？→ 生成产物 + hwfix 对拍时确认。
- **[非阻塞]** confederation/graceful-restart/timer 三小容器是否含 presence/嵌套 list？影响删除语义与测试矩阵嵌套项 → 编写 spec 场景时对照生成结构确认。
