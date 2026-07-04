# system-architecture — 系统架构总览（反向还原）

> **文档性质**：本文档由 **已实现代码反向还原**（Brownfield），忠实描述当前 as-built 现状，而非目标/理想架构。凡代码与 CLAUDE.md 红线或既有设计文档不一致处，均**如实标注**并指明 `legacy` / `权威` 边界。
>
> **还原基准**：`main@b1cfbae`。所有 `file:line` 引用以该提交为准。
>
> **本文是能力地图的锚点**：各子系统的详细架构见 `openspec/specs/<capability>/design.md`（见 §5 能力地图）。REST API 北向接口三件套（`devices-api` / `config-api` / `yang-api`）已单独反向补齐，本文不重复。

---

## 1. 一句话定位

无数据库、模型驱动的交换机设备管理平台：前端由 YANG/CRD 模型自动渲染表单 → 后端将「期望配置」经声明式 Reconciler 对齐到设备 → 通过 NETCONF（SSH 830）下发。

## 2. ⚠️ 首要事实：代码里存在两套并存、相互冲突的架构栈

这是理解本系统的**第一前提**。当前代码库处于一次**未完成的架构迁移**中间态，两套 reconcile 栈都真实存在、都能运行，但**并不汇成 CLAUDE.md 所描述的单一分层管线**。

| 维度 | **Stack A — K8s CRD 栈** | **Stack B — yang-controller-runtime 栈** |
|------|--------------------------|------------------------------------------|
| 进程入口 | `backend/cmd/controller/main.go`（controller-runtime / K8s Operator） | `backend/main.go`（自研 `manager.New` + Gin API :8080） |
| 事件源 | K8s CRD 变更（etcd Watch） | `PeriodicSource` 周期轮询 / gNMI 订阅 / 文件变更 |
| 对齐机制 | Actor 模型 mailbox + 2PC（candidate/commit）+ 版本快照 | `GenericReconciler` 反射 diff + edit-config |
| 绑定 CRD | `api/v1`（**旧**，但正是在跑的生产控制器所绑定） | `api/biz/v1` + `api/core/v1`（**新**，仅它有生成的 CRD YAML） |
| 期望态存储 | etcd（K8s apiserver） | TTL+LRU 内存 ConfigStore（`internal/cache`） |
| 翻译层 | `pkg/translator`（CRD→厂商 YANG） | 无翻译层，desired 直接是 ygot 结构 |
| 代表控制器 | `backend/controllers/*_controller.go` | `backend/internal/controller/{vlan,ifm,system,interfaces}` |
| 运行状态 | 实际可运行 | 实际可运行 |

### 与 CLAUDE.md 红线的硬冲突

- **R01（强制 yang-controller-runtime，明文禁止 Actor 模型）**：Stack B 合规；但 `pkg/yang-runtime/actor/`（2PC、版本快照、mailbox actor）是代码量**最大、最活跃**的子系统，且被生产控制器 `backend/controllers/*` 依赖 → **Stack A 违反 R01**。
- **R03（禁止数据库，仅 TTL+LRU + JSON）**：Stack A 依赖 etcd 持久化 CR → **与 R03 张力**；Stack B 合规。
- **API group 冲突**：`api/v1/groupversion_info.go:13` 与 `api/biz/v1/groupversion_info.go:13` **注册了同一个 group `biz.usmp.io/v1`**，但 `BusinessVlan` schema 不兼容，二者无法同时注册进同一 scheme。

### 权威裁定（据 CLAUDE.md R01）

> **Stack B（yang-controller-runtime）为权威目标架构；Stack A（K8s CRD + Actor）标注为 `legacy`，待渐进退役。**
>
> 依据 §5.1「必须审计存量代码，标记 legacy / 新架构边界」。注意：裁定为 legacy **不等于**现状已迁移——当前**生产入口 `cmd/controller/main.go` 跑的仍是 Stack A**。迁移路径见 §7。

---

## 3. 分层视图与数据流

### 3.1 CLAUDE.md 声称的 C1–C5 分层（权威目标）

```
C1 Manager   全局生命周期：schema 加载、client 连接池、controller 注册、插件管理
   │
C2 Controller  每 YANG 模块一个，事件队列 → 调用 Reconciler
   │
C3 Reconciler  对齐 desired↔actual（diff + 推送），用户实现此接口
   │
C4 EventSource 产生 reconcile 事件：周期轮询 / gNMI 订阅 / 文件变更
   │
C5 ClientPool  设备连接池：断线重连、超时重试、异常处理
```

对应 `openspec/specs/yang-controller-runtime/design.md`。C1–C5 在 `backend/pkg/yang-runtime/` 下**均有实现**，但存在 §7 所列空转件。

### 3.2 数据流：两条真实存在的 reconcile 路径

**路径 B（Stack B，`backend/main.go` 启动、绑定 :8080）——权威**

```
Gin API (/api/v1/config POST)  或  PeriodicSource(周期轮询)
        │  ConfigStore.Set(desired) + Manager.TriggerReconcile
        ▼
Controller.Enqueue → RateLimitingQueue → worker
        ▼
GenericReconciler.Reconcile   (reconcile/reconcile.go:77)
    ConfigStore.Get(desired ygot)  +  DeviceClient.Get(actual, NETCONF get-config)
        ▼  DiffEngine.Diff(反射树 diff)
    DeviceClient.Set(changes, edit-config + commit)
        ▼
        设备 (NETCONF SSH 830)
```

**路径 A（Stack A，`cmd/controller/main.go` 启动）——legacy**

```
K8s CRD 变更 (etcd Watch)
        ▼
controllers/*Reconciler.Reconcile (controller-runtime)
    translator.TranslateConfig(CRD Spec → 厂商 ygot 结构)
        ▼
ActorManager.GetDeviceActor(ip) → DeviceActor → ModelActor(mailbox)
    TranslateCmd → PrepareCmd(candidate) → CommitCmd(running)  [2PC]
        ▼
        设备 (NETCONF SSH 830)
```

两条路径**均不经过对方**：路径 A 不走 yang-runtime 的 `Manager`/`Controller`/`queue`；路径 B 不走 Actor/翻译引擎。

---

## 4. 组件物理布局（代码坐标）

```
backend/
├── main.go                      # Stack B 入口（Manager + Gin :8080）
├── cmd/controller/main.go       # Stack A 入口（K8s Operator）  [legacy]
├── cmd/test-server/main.go      # E2E 测试用 Gin，内存 netsim
├── pkg/yang-runtime/            # 【C1-C5 框架】→ yang-controller-runtime/design.md
│   ├── manager/                 #   C1 + InMemoryConfigStore
│   ├── controller/ queue/ predicate/   # C2 事件循环 + 工作队列
│   ├── reconcile/ diff/ schema/ # C3 + 反射 diff + (空转)schema
│   ├── source/                  # C4 周期/gNMI订阅/文件
│   ├── client/                  # C5 NETCONF/gNMI + 连接池 → device-protocol/design.md
│   ├── plugin/                  #   (空转)插件注册但从不调用
│   └── actor/                   # 【Actor/2PC/版本】→ actor-transaction/design.md  [legacy per R01]
├── pkg/translator/              # 【翻译引擎】CRD→厂商YANG → translation-engine/design.md
├── api/v1/                      # 旧 CRD 类型 [legacy]         → business-crd/design.md
├── api/biz/v1/ api/core/v1/     # 新 CRD 类型（有生成 YAML）    → business-crd/design.md
├── controllers/                 # Stack A CRD 控制器 [legacy]  → business-crd/design.md
├── internal/controller/{vlan,ifm,system,interfaces}/  # Stack B reconciler → yang-controller-runtime/design.md
├── internal/cache/ttl_lru.go    # 【TTL+LRU】→ config-cache/design.md
├── internal/generated/{huawei,openconfig}/  # ygot 生成结构（R04）
├── internal/api/                # REST 北向 → devices-api/config-api/yang-api（已还原）
└── simulator/{netconfsim,netsim}/  # 【模拟器】→ netconf-simulator/design.md
frontend/                        # 【Vue3 动态表单】→ frontend/design.md
```

---

## 5. 能力地图

| 能力 design.md | 覆盖代码 | 权威性 |
|----------------|----------|--------|
| `system-architecture`（本文） | 全局 | — |
| `yang-controller-runtime` | `pkg/yang-runtime/{manager,controller,reconcile,source,queue,predicate,diff,schema,plugin}` + `internal/controller/*` | ✅ 权威（R01） |
| `device-protocol` | `pkg/yang-runtime/client`（NETCONF/gNMI/连接池） | ✅ NETCONF 权威；gNMI 为 stub |
| `config-cache` | `internal/cache/ttl_lru.go` + `manager.InMemoryConfigStore` | ✅ 权威（R03） |
| `actor-transaction` | `pkg/yang-runtime/actor/*` | ⚠️ 与 R01 冲突，但生产在用 |
| `business-crd` | `api/{v1,biz/v1,core/v1}` + `controllers/*` + `config/crd/bases` | Stack A；`api/v1` = legacy |
| `translation-engine` | `pkg/translator/*` | 仅 Huawei 实现 |
| `frontend` | `frontend/src/*` | CRD 驱动为活跃路径 |
| `netconf-simulator` | `simulator/{netconfsim,netsim}` | 测试专用 |
| `devices-api` / `config-api` / `yang-api` | `internal/api/*` | 已单独还原 |

---

## 6. 红线合规矩阵（as-built）

| 红线 | 现状裁定 | 证据 / 缺口 |
|------|----------|-------------|
| R01 禁止更换架构（yang-controller-runtime，禁 Actor） | ⚠️ **部分违反** | Stack B 合规；`pkg/yang-runtime/actor/` + `controllers/*` 仍用 Actor，且是生产入口 |
| R02 禁止旧协议（仅 NETCONF/gNMI） | ✅ 合规 | NETCONF 全功能（`client/netconf.go`）；gNMI 存在但 Get/Set 为空壳；无 Telnet/SNMP |
| R03 禁止数据库（仅 TTL+LRU + JSON） | ⚠️ **部分违反** | Stack B 用 `internal/cache` 合规；Stack A 依赖 etcd 持久化 CR |
| R04 禁止手写 YANG 结构体（ygot 生成） | ✅ 合规 | `internal/generated/{huawei,openconfig}` 真为 ygot 生成（含 `go:generate`） |
| R05 禁止手写固定表单（YANG 自动渲染） | ✅ 合规 | `parseCRDSchemaToFields` → `DynamicForm`/`FieldRenderer` |
| R08 禁止崩溃（异常降级） | 🟡 大体合规 | REST 层统一 200+错误码；但连接池 `CloseAll` 错误被吞、`source.DoneWaitGroup` 非线程安全 |
| R09 禁止数据竞态 | 🟡 存疑 | 多处 RWMutex 正确；`source/periodic.go` 的 `DoneWaitGroup` 无锁计数为潜在竞态 |
| R11/R12 禁止 AI 陈词滥调 / emoji 替代图标 | 🟡 未系统核查 | 前端为暗色网管风，未见紫粉蓝渐变；图标用法待前端设计文档核对 |

> 未列红线（R06/R07/R10/R13–R16）为流程/协作类，非架构层，见 TEAM_HANDBOOK.md 与 CI。

---

## 7. 迁移债与空转件清单

反向还原时发现的「文档说了、代码没做/没接」与半迁移遗留，是后续演进的直接工作面：

| # | 债项 | 位置 | 影响 |
|---|------|------|------|
| D1 | **双 CRD 树冲突**：`api/v1` 与 `api/biz/v1` 抢注 `biz.usmp.io/v1`，BusinessVlan schema 不兼容 | `api/v1` vs `api/biz/v1` | 无法同 scheme 注册；生产控制器仍绑 `api/v1` |
| D2 | **Actor 子系统 vs R01**：最大子系统被红线禁止但生产在用 | `pkg/yang-runtime/actor/` | 迁移到 Stack B 前无法删除 |
| D3 | **plugin 空转**：4 类插件可注册，但 `Controller`/`GenericReconciler`/Actor 均从不调用 Validate/Mutate/Pre/PostReconcile | `pkg/yang-runtime/plugin` | 扩展点形同虚设 |
| D4 | **schema 层空转**：`main.go` 从不设 `SchemeDir`，schema 树运行时为空；diff 靠反射，schema 参数传 nil | `manager.go:22`, `internal/controller/vlan/reconciler.go:27` | 路径校验/schema 感知缺失 |
| D5 | **gNMI 空壳**：`Get` 发空 `GetRequest`，`Set` 发空 Path/Val | `client/gnmi.go:97,154` | gNMI 实际不可用，AUTO 永远落 NETCONF |
| D6 | **NativeDeviceConfig 下发 = TODO**：`applyNativeConfig` 仅 `time.Sleep`，无真实 NETCONF | `controllers/nativedeviceconfig_controller.go:223` | 原生配置通道未实现 |
| D7 | **ConfigStore.List/ListDevices = stub**：返回 `nil,nil` | `manager.go:55,61` | `PeriodicSource(deviceIDs=nil)` 无法枚举设备 |
| D8 | **多厂商翻译仅 Huawei**：Cisco/H3C/Juniper 仅枚举占位 | `pkg/translator/factory.go:21` | 单厂商 |
| D9 | **前端双代动态表单**：新 CRD 驱动路径活跃，旧 `components/yang/*` 静态路径未接路由 | `frontend/src` | 死代码待清理 |
| D10 | **两个模拟器**：`netconfsim`(真 SSH，集成测试用) 与 `netsim`(内存，test-server 用) 无关并存 | `simulator/*` | 概念重叠 |

## 8. 后续演进建议（仅列，不在本次范围内决策）

1. **确立单栈**：将生产入口从 `cmd/controller/main.go`（Stack A）切到 `backend/main.go`（Stack B），或反之——需一次专门的 `/opsx:propose`。
2. **收敛 CRD 树**：退役 `api/v1`，统一到 `api/biz/v1`+`api/core/v1`（唯一有生成 YAML 的一套）。
3. **偿还空转件**：接通 plugin 钩子、加载 schema、补 gNMI 或明确弃用、实现 NativeDeviceConfig 下发。
4. 每条迁移遵循 §5.3「旧代码保留 + 新代码并行 + 双路径验证 → 切换 → 删除」。

---

## 9. 关联文档

- 权威框架设计：根 `yang-controller-runtime.md`、`spec/openconfig-vlan-controller.md`（Stack B 忠实基线）
- legacy 参考（Stack A，已被红线取代）：`backend/docs/architecture/overview.md`、`refactor-by-crd.md`、`docs/superpowers/specs/2026-05-03-k8s-native-crd-architecture.md`
- 北向 API：`openspec/specs/{devices,config,yang}-api/`、`backend/docs/api/rest-api.md`
