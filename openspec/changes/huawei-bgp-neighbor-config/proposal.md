## Why

BGP 一期交付了公网 `/bgp:bgp` 进程标量层，但**所有 peers/afs/peer-groups 在 huawei-bgp 里是 `augment` 到 `/ni:network-instance/instances/instance/bgp/base-process` 下的**——公网邻居即 `instance[_public_]/bgp/base-process/peers/peer`。两个硬前置已就位：**network-instance 配置面**（#148/#149）把 `/ni:network-instance` 接成单一 ygot 根的可配模块，**per-node namespace XC-06**（#151）让根下 huawei-bgp 子树编码带正确 namespace。本 change 交付 **2a 基础邻居**：公网 BGP peer 的读改下发闭环。

**关键架构事实（决定本 change 体量小）**：network-instance 描述符与 reconciler **已覆盖整棵 `/ni:network-instance` 子树**（含结构上属于它的 huawei-bgp augment），reconciler 走容器根整根收敛。故 2a **不新增描述符、不新增控制器**——peer 字段本就在 `HuaweiNetworkInstance_..._Instance.Bgp` 生成物内，desired 填上即经既有链路编码（XC-06 namespace）下发。本 change 主体是**证明 peer/af 面端到端流通 + 完备测试矩阵**，生产代码预期极少（若深层嵌套/枚举/namespace 暴露缺口再补）。

## What Changes

- **接入公网 BGP peer 基础邻居配置面**：`/ni:network-instance/instances/instance[_public_]/bgp/base-process/peers/peer`（key=`address`）下**全部 config-true（rw）标量**（~27 个，含 mandatory `remote-as` + `description`/`connect-mode`/`ebgp-max-hop`/`tcp-mss`/`password-*`/`tracking-*`/`valid-ttl-hops` 等，**不做字段挑选**，完备性由 schema 驱动用例保证）。
- **基础子容器**：peer 级 `timer`、`graceful-restart`、`bfd-parameter` 的 config-true 标量。
- **地址族条目**：`afs/af` list（key=`af-type`，enum `E_HuaweiBgp_AfType`）——**仅建立 AF 条目**（af-type key），验证嵌套 list-under-list + 枚举 key 编解码。
- **复用既有链路，零新描述符/控制器**：经 network-instance 描述符（谓词 `HasPrefix("/ni:network-instance")` 已覆盖 peer 路径）+ ni reconciler 容器根收敛 + XC-06 per-node namespace（`<bgp>`↓ 带 huawei-bgp namespace）。
- **完备测试矩阵**（`yang-config-test-design` / T02b）：schema 驱动枚举 huawei-bgp peer config-true 标量 + 计数断言 / B2 下发→回读→收敛（含 per-node namespace 真值断言）/ 并发-race / 边界（remote-as/address 域）/ 嵌套（peer→afs→af）/ 幂等 / 负路径。
- **明确排除（2b/后续）**：af 内**策略属性**（import/export route-policy、route-filter、ACL group、tunnel-policy）——门控于未集成的 routing-policy/xpl/acl/tunnel-management；`peer-groups`、`dynamic-peer-prefixes`、`egress-engineer(-parameter/-peer-sets)`、`fake-as-parameter`；config-false 状态子容器（`*-state`/`peer-bfd-session-states`）；per-VPN peers（非 `_public_` 实例，后续）。
- **删除语义**：peer/af 删除沿用平台声明式 subset 语义（天然不删）+ DELETE 命令通道推迟债（同 network-instance NI-06）。

## Capabilities

### New Capabilities
- `huawei-bgp-neighbor-config`: 华为公网 BGP 基础邻居（`instance[_public_]/bgp/base-process/peers/peer` config-true 标量 + timer/graceful-restart/bfd-parameter + afs/af af-type）的模型驱动配置管理——覆盖字段清单、复用 ni 描述符与 XC-06 namespace、分期边界（策略属性门控 2b）、完备测试矩阵。

### Modified Capabilities
- `yang-xml-codec`: **ADD XC-07**——通用引擎新增 YANG `empty` 类型（ygot `YANGEmpty`，非指针 bool，presence-only）编解码。**触发原因**（apply 期 B2/往返实测）：peer/bfd-parameter/compatible 是首个走此路径的驱动字段，既有引擎报 `unsupported field form bool`。既有类型行为不变。device-driver-registry（复用 ni 描述符）、huawei-network-instance-config 不变；af-type 枚举 key 与深层嵌套经实测**无缺口**（无需改动）。

## Impact

- **代码**：预期主要是测试（`internal/drivers` peer 编码断言 + `pkg/yang-runtime/xmlcodec` 深层嵌套/枚举 key 用例 + `internal/controller/networkinstance` B2 集成含 peers）；生产代码预期极少或零（描述符/reconciler/namespace 均已就位）。若暴露缺口按 TDD 补。
- **依赖**：network-instance（✅ #148/#149）+ XC-06 per-node namespace（✅ #151）——2a 全部前置已交付。
- **前端**：peer 配置经通用模块控制台按 YANG 自动渲染（R05），不新增硬编码表单。
- **合规**：R04、T02b、B2 集成、worktree 隔离、≤500 行/commit、PR ≤1000 行。
