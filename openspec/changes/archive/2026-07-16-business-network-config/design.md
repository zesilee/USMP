# Design: business-network-config

## Context

原生配置全链路已交付（模块控制台 + config-api + driver registry + xmlcodec + gen-yang）。本变更在其上叠加业务意图层。部署约束（2026-07-15 拍板，见记忆 k8s-paas-deployment-constraints）：USMP 是 K8s 内 PaaS 底座组件、必须多实例、禁止本地存储、持久化能力 = K8s CRD。历史边界：Stack A 死于「CRD 当架构」（Go/YANG 双重建模 + Actor + 前端消费 CRD），本设计只把 CRD 当**持久化载体 + watch 事件源**，Stack B 对账架构不变（R01）。

现状关键事实（explore 核实）：
- `internal/crdsource` 是活的 CRD→Stack B 桥接先例（`main.go:113`，无集群优雅降级），client-go/controller-runtime v0.19 已在 go.mod。
- desired ConfigStore 是纯内存 TTL 缓存（TTL 1 分钟、重启即失，`manager.go:162`）——只能当派生缓存，不能当意图存储。
- NETCONF 客户端已有 candidate→commit + DiscardCandidate（DP-04）；netconfsim 已有 running/candidate/commit/discard-changes（NS-02）。2PC 缺口仅剩 confirmed-commit 原语（客户端 + sim 各一块）。
- 前端菜单按模块 category 自动分桶（task-name 构建期 codegen），新菜单组零前端菜单代码；但现有模块控制台是设备作用域，平台作用域控制台需要新页面骨架。

## Goals / Non-Goals

**Goals:**
- 意图 CR 作为公共契约（kubectl/GitOps 直写受支持），YANG 唯一 schema 源，YANG→CRD codegen。
- 意图 Reconciler：确定性展开 + 跨设备 2PC 下发 + 完整生命周期（finalizer 删除、收缩差集、代际追踪）。
- 首能力「跨设备 VLAN 打通」端到端闭环（后端 + 前端 + 模拟网元集成测试）。
- 意图层多实例就绪（无本地状态、leader election 接缝）。

**Non-Goals:**
- 全局 HA（device store 共享化、存量控制器 leader election、audit 迁出本地文件）——另立 openspec 任务。
- 归属硬锁（409 拒绝手改）——二期。
- 非华为厂商展开、第二个业务能力——后续按本模式复制。
- admission webhook——apiserver OpenAPI + 约束引擎收敛侧兜底已覆盖拍板要求，webhook 的可用性耦合不引入。

## Decisions

### D1 CRD 形态：每能力一个 Kind，OpenAPI 由 YANG 构建期生成

- 备选：单一通用 CRD（payload 自由 JSON）成本最低，但 CR 是公共契约（用户拍板）——kubectl 直写需要 apiserver 侧字段级校验、explain/printer columns、按能力 RBAC。手写 CRD 类型 = Stack A 双重建模坟场，排除。
- 结论：新增构建期工具 `tools/crdgen`（与 tasknamegen 同模式）：读意图 YANG（goyang Entry）→ 生成 CRD manifest（OpenAPI v3 schema + printer columns + status 子资源声明）。生成物提交入库 + regen-and-diff 门禁（golden 方法论，R04 同款）。
- 类型映射：uint*/int*→integer+min/max（range）、string+pattern→pattern、enumeration→enum、leaf-list→array、嵌套 list→array<object>（list key → required）；when/must/leafref 不映射（约束引擎收敛侧兜底，BIC-03）。
- Go 侧类型：CR spec 反序列化目标复用 **ygot 生成的意图结构体**（R04），不再生成第二套 typed client——unstructured→JSON→ygot Unmarshal。
- API group `biz.usmp.io`、Kind `BusinessVlanService`、版本 `v1alpha1`（storage）。沿用 legacy group 名但 Kind 不与 `backend/api/biz/v1` 存量类型冲突（存量是 BusinessVlan）；CRD 版本升级策略：v1alpha1 期间允许破坏性变更 + 重建实例，进 v1beta1 后才引入 conversion。

### D2 意图 YANG 模型位置与管线

`backend/internal/yang/models/usmp-business-vlan.yang`（openconfig 本地模型同款位置），gen.conf 新增 usmp 业务条目 → ygot 生成到 `internal/generated/business/`。模块级 task-name 扩展值 = 新 category「业务网络配置」→ tasknamegen 收割 → 前端菜单组自动出现。意图模块**不注册** driver registry（不下设备），但需在 `/yang/schema`、`/yang/modules` 暴露（R05 前端渲染）。

### D3 编排层落位：泛化 crdsource 模式为意图控制器

- 意图控制器 = 标准 Stack B Controller + `IntentReconciler`（C3）；事件源 = controller-runtime informer watch（C4，泛化 `internal/crdsource` 现有实现），generation 变化与周期 resync 双触发。无 kubeconfig 时跳过注册（BIO-01 降级）。
- 展开引擎 = 纯函数 `expand(spec) → []Fragment{device, module, path, config}`（BIO-02 可测试性），VLAN 打通的展开规则：
  - huawei-vlan：`/vlan:vlan/vlan:vlans` 下 `{id: vlan-id, name}` 条目（与模块控制台同路径，复用删除通道的嵌套顶层容器修复）。
  - huawei-ifm：access 口 → 接口 `l2-attribute` 配 PVID=vlan-id + linkType access；trunk 口 → linkType trunk + trunk 放行表加 vlan-id。精确叶子名以 generated structs 为准，apply 时先写 golden 展开测试锁定。
- 存量改造边界：旧 `crdsource.BusinessVlan` 桥接**并行保留**（不动）→ 新意图控制器落地验证后另行 PR 退役旧桥接（渐进替换）。

### D4 跨设备 2PC 与声明式对账的分工（用户拍板：2PC）

- 首次/变更下发 = 事务：`TxCoordinator` 经 ClientPool 对全部目标设备执行 prepare（edit-config candidate）→ 任一失败全体 `DiscardCandidate` → `CommitConfirmed(timeout)` → 全部成功后逐台 `ConfirmCommit`。confirming 部分失败：依赖设备侧超时回滚兜底，status 标不一致告警（BIO-03 残余窗口，诚实呈现）。
- **事务成功后才** `ConfigStore.Set` + TriggerReconcile；失败不写 desired（防周期对账绕过事务）。
- 稳态漂移修复 = 既有每设备声明式对账（BIO-04）；意图 Reconciler 周期 resync 时重写 desired，对冲 desired TTL 1 分钟过期的存量隐患。
- 设备无 :confirmed-commit 能力：客户端明确报错，事务协调器降级为普通 commit 并在 status 标「非事务下发」（华为 NE 与 sim 都有能力，降级路径主要服务负路径测试）。
- 并发安全：事务协调器复用 client 的 opMu 串行化（scrapligo 非并发安全，见记忆）；跨意图对同一设备的事务用每设备互斥（内存锁即可——多实例下同一时刻仅 leader 执行）。

### D5 生命周期机制

- **finalizer** `biz.usmp.io/cleanup`：删除时展开认领 → 逐设备 DELETE 命令通道（复用 config_delete 链路）→ 全部成功摘 finalizer；部分失败保留重试（BIO-05）。
- **收缩差集**：status 持久化上一代展开摘要（认领集合即可承载）；本代展开 vs 上一代求差 → 差集走 DELETE。差集只依赖 CR status（多实例就绪，BIO-06）。
- **status 结构**（BIC-04）：observedGeneration、conditions[Validated, Converged]、deviceStates[]、claims[]。status 更新用 patch + 冲突重试（乐观并发）。

### D6 归属与 config-api

认领索引由意图控制器从全量 CR status 聚合（内存索引，watch 驱动更新，可重建——无持久化需求）；config-api SetConfig/Delete 查索引命中则响应附 `ownershipWarning`（BR-11，不改状态码不拦截）。归属查询 API：`GET /api/v1/ownership?device=&path=`（design 细化到 apply 期）。

### D7 前端

- 菜单组：零代码（category 分桶存量机制）。
- 平台作用域控制台：新页面 `BusinessConsolePage`（列表 + 详情 + 表单三态），表单复用 YANG 动态渲染框架（devices 嵌套 list 增删改 → F3 真浏览器测试层强制）；数据面走 USMP API（前端不碰 apiserver——USMP API 代理 CR 的 CRUD，公共契约的 kubectl 直写是旁路而非前端路径）。
- 原生控制台徽标：模块控制台列表行按归属查询 API 附徽标（FE-18）。

### D8 多实例就绪的接缝

意图控制器启动包一层 `RunWithLeaderElection(enabled bool)`：enabled=false（默认，兼容现部署）直接跑；enabled=true 用 controller-runtime Lease 选主。全库其余控制器不动（Non-Goal）。

## Risks / Trade-offs

- [confirming 阶段部分失败窗口] → confirmed-commit 超时自动回滚兜底 + status 不一致告警；窗口不可为零，spec 已诚实声明（BIO-03 场景 3）。
- [YANG→OpenAPI 映射长尾（union/typedef/leafref）] → 一期意图模型主动只用可映射类型集（BVS-01 字段全在集合内）；crdgen 遇不可映射类型 fail-fast 报错而非静默降级。
- [CRD 安装/升级与应用滚动的顺序耦合] → helm/初始化脚本先 apply CRD 再滚 USMP；v1alpha1 期间允许重建实例，避免过早背 conversion webhook。
- [kubectl 直写绕过 USMP API 的校验缺口（when/must）] → BIC-03 Validated 条件 + 不展开；文档写明直写的校验时延差异。
- [双通道写冲突（意图 vs 手改原生）] → 软归属警告 + 意图收敛覆盖手改（明确语义）；同 vlan-id 跨意图认领冲突 → 后者 Validated=False（BVS-04）。
- [CI/本地无 K8s 集群] → 单测用 envtest/fake client；B2 集成测试意图侧用 fake apiserver + 真 netconfsim；无集群运行时优雅降级已是 spec 要求（BIO-01）。
- [scrapligo 并发坑] → 事务协调器不引入新并发写路径，复用 opMu 串行化；-race 全程（R09）。
- [范围膨胀] → 交付按 tasks.md 波次切 PR（≤1000 行/PR，TM04）：sim 能力 → 客户端原语 → crdgen → 意图模型+展开 → 控制器+2PC → 生命周期 → config-api 归属 → 前端。

## Migration Plan

1. 每 PR 独立可合（详见 tasks.md 波次），旧 `crdsource.BusinessVlan` 桥接全程并行保留。
2. 部署：新增 CRD manifest 与 RBAC（ServiceAccount 读写 biz CR + status + Lease）进部署编排；无 CRD/无集群环境行为不变（降级）。
3. 回滚：删除意图 CRD 即关停意图层（finalizer 需先摘除），原生链路不受影响。
4. 全部落地后：另立 PR 退役旧 BusinessVlan 桥接与 `pkg/translator` 中被替代部分；全局 HA 另立 openspec 任务。

## Open Questions

- huawei-ifm trunk 放行表的精确叶子路径（apply 首个任务用 generated structs + golden 锁定，不阻塞设计）。
- 归属查询 API 的路径粒度（模块级 vs 精确 path 前缀）——apply 期按前端徽标最小需求定。
- CommitConfirmed 超时值（初值 60s，集成测试校准）。
