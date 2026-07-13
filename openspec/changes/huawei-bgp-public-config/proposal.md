## Why

USMP 已交付华为 VLAN、IFM（接口）两类设备配置的模型驱动闭环，但 BGP —— 数据中心/城域网最核心的路由协议 —— 尚无任何配置管理能力。BGP 模型体量巨大（`huawei-bgp.yang` 5806 行 + 2 子模块 + 14 个 feature 增补模块），无法一次接入；需要沿用 VLAN/IFM 已验证的驱动注册表 playbook，从**公网 BGP 进程基础配置**这一自包含、零外部依赖的最小子集切入，为后续 peer/地址族、per-VPN、EVPN 等分期扩展打地基。

## What Changes

- **新增公网 BGP 配置面**：接入 `/bgp:bgp`（`huawei-bgp` 模块顶层独立根容器）的 `global` + `base-process` 子树的**全部可配（rw）字段，无遗漏**——含 `global`（yang-enable、memory-overload-exception-discard-route、paf-controls/paf-control）、`base-process` 直属全部标量 leaf（enable、as、keep-all-routes、check-first-as、router-id-auto-select、shutdown、local-ifnet-mtu、private-4byte-as、local-cross-no-med、as-path-limit、dynamic-session-limit、peer-up-route-lowest-priority、delay-time）、以及 `confederation` / `graceful-restart` / `reference-period` / `timer` / `default-parameter` 全部子容器可配字段。SchemaTree 入口 `HuaweiBgp_Bgp`，与 `HuaweiVlan_Vlan_Vlans` / `HuaweiIfm_Ifm_Interfaces` 同构。**不做字段挑选**（商用交付：任何"覆盖部分字段"都是遗漏）。`base-process` 下的 config-false 只读态（vpn-brief-infos、graceful-restart-status、error-discard-info、remote-prefix-sid-states）不是配置目标，仅随 ygot 生成物存在、可只读呈现，不进 edit-config 写用例。
- **ygot 生成**：`backend/internal/generated/huawei/gen.conf` 的 `modules` 增加 `huawei-bgp`，`make gen-yang` 重生成强类型结构体（R04：禁手写、禁改 generated/）。
- **驱动注册**：`backend/internal/drivers/huawei.go` 新增一条 `driver.Descriptor{Vendor:"huawei", Module:"bgp"}`，含 `MatchRoute`/`MatchDecode`/`MatchEncode` 谓词、**显式** `Namespace "urn:huawei:yang:huawei-bgp"`、SchemaTree 入口闭包。编解码全部走通用引擎 `pkg/yang-runtime/xmlcodec`，**零 XML 代码**。
- **模拟网元方言**：`simulator/netconfsim` 增加 BGP edit-config/get-config 方言，支撑 B2 端到端集成测试。
- **完备测试矩阵**：触发 `yang-config-test-design`（T02b），覆盖全属性可配 / 端到端到设备 / 并发-race / 边界 / 幂等 / 负路径。
- **明确排除（分期）**：`instance-processs/instance-process`（peers、地址族 af）、per-VPN augment（`/ni:.../instance/bgp`）、全部 feature 模块（evpn/flow/l2vpnad/link-state 等）→ 二/三/四期，不在本 change 范围。
- **登记分期前置依赖链（不可简化，依赖先交付；经全 augment 树核验，见 design.md「依赖分析」）**：
  - **结构真相**：BGP 的 peers/afs/peer-groups 全在 `augment /ni:.../instance/bgp/base-process` 下，`when name='_public_'` 区分公网 vs VPN。**公网邻居即 `network-instance/instance[_public_]/.../peers`**，不独立存在。
  - **二期唯一硬前置 = `huawei-network-instance`**（103 行，零未集成依赖，极易）——它是所有 peering（公网+VPN）的 augment 根。
  - **二期-2a（基础邻居：address/remote-as/af-type/peer 标量）依赖 = 仅 network-instance**：所有跨模型引用均为可选 leaf，唯一 mandatory 是 BGP 自有类型的 remote-as，故 2a 可在 network-instance 就绪后完整交付、零策略依赖。
  - **二期-2b（可选策略属性：route-policy/route-filter/acl/tunnel）**：各属性门控于其 leafref 目标模型集成——`tunnel-management`/`xpl`(仅 ifm)→ `routing-policy`(依赖 tnlm)→ `acl`(依赖 time-range+l3vpn+ni)，各自独立 change、可并行；未集成前禁接入对应属性（leafref require-instance 会致设备侧非法）。
  - **已摘除硬路径**：`routing`(2828)/`bfd`(4342)/`ethernet` —— BGP 仅一处软 `must` 引用 routing，不配时自满足，仅需 codegen-present。
  - **本期（公网 base-process）依赖 = 零**（全部 rw 字段自包含于 huawei-bgp 主+子模块），可独立完整交付、不阻塞任何后续波次。

## Capabilities

### New Capabilities
- `huawei-bgp-config`: 华为公网 BGP（`/bgp:bgp/base-process` 标量层）的模型驱动配置管理——覆盖字段清单、命名空间登记、根 SchemaTree 入口、路由/编码/解码谓词语义、模拟网元 BGP 方言、分期边界，以及完备测试矩阵要求。

### Modified Capabilities
- `yang-codegen-pipeline`: **MODIFY CG-02**——扩展 `genfix` 后处理器范围，新增「确定性 schema 规范化」：解压 ygot 内嵌 gzip schema blob、对无序集合数组（首要为 `Augmented`）与对象键做稳定重排、以固定参数重压回填，使 CG-01「重复生成字节一致」在多-augment 闭包下成立。**触发原因**（spike 实证，见 design.md）：加 `huawei-bgp` 因 augment 级联拉入多方 augment 同一目标的闭包（bfd/ethernet/tunnel-management 等 augment ifm/network-instance），触发 goyang 非确定 augment 应用序 → schema blob 字节逐次漂移 → R04/CG-03 regen-and-diff 门禁永远 fail。这正是 proposal 早前预留的"若 spike 证伪自包含假设则补 yang-codegen-pipeline delta"契机。device-driver-registry / yang-xml-codec / translation-engine 仍不变（BGP 作为数据流经，DR-01/XC-01~04/TE-05~06 不动）。

## Impact

- **代码**：`backend/internal/generated/huawei/gen.conf`（+1 模块名）、`backend/internal/generated/huawei/*`（regen，勿手改）、`backend/internal/drivers/huawei.go`（+1 描述符 + BGP namespace 常量）、`simulator/netconfsim`（+BGP 方言）、新增 `*_integration_test.go` + xmlcodec/driver 单测 + hwfix golden。
- **依赖闭包**：`huawei-bgp` import `huawei-network-instance`/`huawei-ifm`/`huawei-pub-type`/`huawei-extension`/`huawei-routing-policy`/`huawei-xpl`/`huawei-routing`/`huawei-tunnel-management`，子模块 `huawei-bgp-common` import `huawei-acl`；全部须在 `yang-models/network-router/8.20.10/ne40e-x8x16/` 可解析（一级依赖已核实齐全，二级待 `make gen-yang` spike 确认）。
- **版本**：以 `8.20.10/ne40e-x8x16` 为准（gen.conf 现行目标），非最初提及的 8.20.0。
- **前端**：BGP 配置经通用「模块控制台」由 YANG 模型自动渲染（R05），本期不新增前端硬编码表单。
- **风险（已由 spike 消解）**：ygot 对 `augment /ni:...` 的处理——实测为**物化整个 import 闭包**（不报错、不剪枝），生成成功且 `go build ./...` 全通过，`HuaweiBgp_Bgp` 入口确认。详见 design.md「Spike 结论」。
- **生成物边界登记（generated-but-not-integrated，防误判）**：加 `huawei-bgp` 连带自动生成 network-instance/acl/bfd/routing/l3vpn/routing-policy/time-range/tunnel-management 的 ~73k 行惰性 struct（ygot 因 augment 级联，无法剪枝）。**这些模型仅有生成类型、无功能描述符/无配置通道/无测试覆盖，不视为已集成**；本 change 唯一功能集成的配置面是 `/bgp:bgp`。后续分期接入这些模型时按依赖 DAG 补功能描述符，codegen 侧已就绪。
- **合规**：R01/R02/R03/R04/R17、T02b、B2 集成、worktree 隔离、≤500 行/commit、PR ≤3000 行。
