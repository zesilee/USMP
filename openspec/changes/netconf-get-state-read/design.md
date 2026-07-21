# design — netconf-get-state-read

## Context

前端 config=false 只读字段恒为空的根因不在渲染层：FieldDef.Readonly → disabled 控件、xmlcodec.Decode 不过滤 config-false、`EmitJSON`(RFC7951) 原样带出、useConfigForm(FE-14) 已把 readonly 排除出提交 payload——整条显示管道就绪。缺口在读链路两端：

1. 后端 `NETCONFClient.Get` 只发 `<get-config>`（`netconf.go:189`），按 RFC6241 它不返回状态数据——真机也读不到；
2. 模拟网元只分发 get-config/edit-config/commit/discard-changes，`<get>` 落 default 回 `<ok/>`（`server.go:167`），且只有 running/candidate 配置树、没有状态数据概念。

scrapligo v1.4.0 原生提供 `Driver.Get(filter string, opts...)`（`driver/netconf/get.go:35`），协议层零障碍。

## Goals / Non-Goals

**Goals:**
- 模拟网元实现 `<get>`：running 配置树 + 状态 overlay 树合并，支持 subtree filter；状态树可注入、不受写操作影响。
- 客户端 `WithStateData()` option：置位发 `<get>`，缺省行为不变。
- `GET /config` 切 `<get>`，只读字段端到端点亮（真机同样受益）。
- standalone 模拟网元注入 `DemoStateSeed`，staging E2E 可断言。

**Non-Goals:**
- 对账（Reconciler）读路径不动：仍 `<get-config>`，diff 只比配置数据。
- 不做状态数据的独立缓存/独立 TTL——合并结果沿用 RunningCache（key=ip|path，TTL 30s）。
- 不做前端改动（渲染管道已就绪）；不做 gNMI（R02 规划能力）。
- 不实现 `<get>` 的 with-defaults、origin 等扩展语义。

## Decisions

### D1 状态存储：treeDatastore 加第三棵树（state overlay）

`treeDatastore` 新增 `state *dataNode`（与 running/candidate 并列，同一把 `mu` 保护）。`SetStateDataXML` 解析后整树替换 state。**写操作（EditConfig/Commit/Discard/confirmed-commit 回滚）不触碰 state**——状态数据生命周期独立，天然满足 NS-08「edit-config 不触碰状态树」。

替代方案：状态叶直接种进 running 树（方案 A）。否决：get-config 会泄漏状态数据（违反 RFC6241 语义，也让 NS-08 的「get-config 不含状态」场景不可测）。

### D2 合并算法：复用 edit-config 的 merge walk，但「只并入已存在的 list 条目」

`<get>` 响应 = `mergeState(running.clone(), state)` 后套 filter。合并复用 `findMatch`/`wellKnownListKeys`/`countSameName` 既有 list-key 匹配机制（`editconfig.go`），语义与 merge 一致，仅一处收紧：**keyed list 条目在配置树中无匹配时丢弃该状态子树，不创建幽灵条目**（配置条目被删后，其残留状态不得在 `<get>` 中复活）；非 list 的容器/叶（含纯状态顶层容器）照常并入。同名叶以状态树为准（状态语义覆盖配置回显值）。

替代方案：直接 `applyEdit(state)`。否决：merge 语义会为无配置匹配的状态条目创建幽灵 list 条目。

### D3 RPC 分发：结构化 classify 增加 `<get>`

`rpcEnvelope` 加 `Get *struct{} \`xml:"get"\``、`rpcKind` 加 `rpcGet`、`handleGet` 提取 filter（复用 get-config 的 filter 提取路径）→ `store.GetFiltered`（合并+过滤）。注意 classify 顺序：`<get-config>` 的 envelope 不会误判为 `<get>`（字段独立，结构化解码天然区分）。故障注入 `ErrorOnRPC` 同步支持 `get`。

### D4 客户端：GetOption 而非新接口方法

`GetOptions` 加 `IncludeState bool`，`WithStateData()` 构造 option。`NETCONFClient.Get` 按其选 RPC：置位 → `driver.Get(filter)`，缺省 → `driver.GetConfig(datastore, withFilter)`。断线自愈与现有逻辑共用（标记失效→重连→重试一次；`<get>` 幂等）。opMu 串行化不变（scrapligo 非并发安全，见 scrapligo-concurrency-pitfalls）。

替代方案：Client 接口加 `GetState` 方法。否决：接口面扩散，所有实现/mock 都要跟着改；option 对既有调用方零侵入。

### D5 API：只改 fetchFromDevice 一行语义

`config_handler.go fetchFromDevice` 的 `cli.Get` 追加 `client.WithStateData()`。缓存键/TTL/降级/超时全部不动（BR-02/03/04 语义不变）。写侧安全性依赖既有事实：desired 来自前端 payload（FE-14 排除 readonly），diff 为 desired⊆actual 子集比对，actual 里多出的状态叶不触发漂移或删除。

### D6 演示种子：DemoStateSeed 只覆盖 IFM dynamic

`DemoStateSeed` 为 5 条 demo 接口提供 `<dynamic>`（oper-status/link-status/physical-status/mac-address/bandwidth/line-protocol-up-time 等，设备侧数字枚举形态，与 DemoSeedConfig 注释约定一致）。VLAN 状态不进 demo 种子（demo 无 VLAN 配置，合并会按 D2 丢弃），由集成测试覆盖（测试内自行种 vlan 配置 + `statistics` 状态——该型号 vlan 条目无 `status` 叶，config false 状态面是 `statistics` 计数器容器）。`cmd/netconf-simulator/main.go` 启动时注入。

## Risks / Trade-offs

- [状态值与真机形态不符] 种子用设备侧数字枚举形态，与既有 seed/解析约定对齐；集成测试断言经 xmlcodec 解码后的 RFC7951 值，形态错会红灯。
- [staging E2E 对回读做精确断言时新增字段导致断裂] apply 时先审 `staging-smoke.spec.ts` 既有断言；只加「只读字段有值」正向断言，不做全量相等断言。
- [schema-less 合并的 list 匹配启发式误判] 与 edit-config 同源机制，已被 wellKnownListKeys 兜底（vlans/vlan、interfaces/interface 均登记）；新模型踩坑沿既有登记表模式修。
- [`<get>` 对真机返回体量大（含统计）] filter 仍按 path 构造 subtree，体量与模块子树同阶；10s 读超时（BR-03）兜底。
- [合并结果进 RunningCache 后 force_refresh/失效语义] 键与 TTL 未变，下发后失效逻辑照旧作用于合并数据，无新状态需管理。

## Migration Plan

单 PR 交付（预估 <1000 行，TM04 内）：sim（D1-D3）→ client（D4）→ API（D5）→ seed+binary（D6），每步先红后绿（T05/T01）。回滚 = revert 单 PR；API 行为回退仅意味着只读字段重新变空，无数据/契约损伤。

## Open Questions

（无——scrapligo 能力、合并机制、写侧安全性均已实测/实读核实。）
