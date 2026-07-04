# business-crd — 业务 CRD 层架构设计（反向还原）

> **权威性**：属 **Stack A（K8s CRD 栈）**。其中 `api/v1` 树标注为 **`legacy`**；`api/biz/v1`+`api/core/v1` 为较新一代（唯一有生成 CRD YAML）。整个 Stack A 与 R01/R03 张力，见 `system-architecture/design.md` §2/§6。
> **还原基准**：`main@b1cfbae`。

## 1. 职责

以 Kubernetes CRD 表达**厂商中立的配置意图(intent)**，经 controller-runtime Reconciler → 翻译引擎 → Actor 2PC → NETCONF 下发到设备。CRD 同时充当**前端动态表单的 schema 来源**（`+custom:*` 标注 → OpenAPI `x-` 扩展）。

## 2. ⚠️ 两棵并存的 CRD 类型树

| | **新树** `api/biz/v1`(`biz.usmp.io`) + `api/core/v1`(`core.usmp.io`) | **旧树** `api/v1`（也注册 `biz.usmp.io`！）|
|---|---|---|
| DeepCopy | 代码生成 `zz_generated.deepcopy.go` | 手写内联 |
| CRD YAML | ✅ `config/crd/bases/` 有 | ❌ 无生成 YAML |
| 消费方 | 仅 `internal/controller/vlan/actor_reconciler.go` | **生产控制器 `backend/controllers/*` + `cmd/controller/main.go`** |
| 表单标注 | `+custom:label/group/placeholder/dynamic` | 无 |
| 裁定 | 目标 | **legacy** |

**冲突（迁移债 D1）**：两树抢注同一 group `biz.usmp.io/v1`（`api/v1/groupversion_info.go:13` == `api/biz/v1/groupversion_info.go:13`），但 `BusinessVlan` schema 不兼容 → 无法同 scheme 注册；靠两个不同 main.go 各注册各的规避。`NativeDeviceConfig` 更有两套不兼容语义（见 §4）。

## 3. CRD 类型（新树，`api/biz/v1` + `api/core/v1`）

| Kind | Spec 要点 | 定义 |
|------|-----------|------|
| BusinessRoute | DeviceID, Destination(CIDR), NextHop, Preference, Description, BfdEnabled | `api/biz/v1/businessroute_types.go:94` |
| BusinessSwitch | DeviceID, Vendor(huawei/h3c/cisco/juniper), Model, ManagementIP, AdminStatus, Location, Tags | `businessswitch_types.go:99` |
| BusinessVlan | DeviceID, VlanID(1-4094), Name, AdminStatus(up/down), BroadcastDiscard, UnknownMulticastDiscard, MacLearning | `businessvlan_types.go:129` |
| BusinessInterface | DeviceID, IfName, AdminStatus, Mode(access/trunk/hybrid), AccessVlan, TrunkVlans[], MTU, EnableLldp, EnableStormControl | `businessinterface_types.go:134` |
| NativeDeviceConfig | DeviceID, Module(YANG 模块名), Config(`map[string]interface{}` 原始 YANG JSON，`+custom:dynamic=true`) | `api/core/v1/nativedeviceconfig_types.go:70` |
| 共用枚举 | `ConfigPhase{Pending,Updating,Ready,Failed}` | 各 `types_common.go` |

## 4. 意图 CRD vs 原生 CRD（抽象分层）

- **意图/业务 CRD**：BusinessRoute/Switch/Vlan/Interface（`biz.usmp.io`）——厂商中立期望态，经翻译引擎映射到厂商 YANG。
- **原生/设备 CRD**：`NativeDeviceConfig`——低层逃生舱，两套冲突定义：
  - `core/v1`：模型化——`Module` + 原始 YANG `Config` map，schema 运行时按 Module 动态加载（意图正确的「YANG 逃生舱」）。
  - `api/v1`（legacy）：传输化透传——`Format`(CLI/YANG/XML/JSON) + `Content` 字符串，**绕过翻译引擎**（`api/v1/nativedeviceconfig_types.go:135`）。注意 `format: CLI` 与 R02（仅 NETCONF/gNMI）软张力。

## 5. 控制器族（Stack A，`backend/controllers/*`）

均 import **旧** `api/v1`，struct 持 `netconfclient.ClientPool`。

- **BusinessVlanReconciler.Reconcile** `controllers/businessvlan_controller.go:68`：watch CR → finalizer → 建 per-device VLAN Actor（`createVlanActor:207`）→ **翻译** `translateBusinessVlanToHuawei:231`（`translator.TranslateConfig(Huawei,Vlan,spec)`）→ 发 `TranslateCmd` → Prepare(candidate) → Commit(running) → 读回 `fetchActualVlanStatus:401` → Phase=Synced，requeue 5min。错误分类 temporary/permanent + 指数退避（`classifyError:446`、`calculateBackoff:483`）。
- **NativeDeviceConfigReconciler.Reconcile** `controllers/nativedeviceconfig_controller.go:55`：用 api/v1 形态；`applyNativeConfig:219` 是 **stub/TODO**（`time.Sleep`，无真实 NETCONF，`:223`）→ 原生下发未实现（迁移债 D6）。
- Route/Switch/Interface 控制器：同构。

> 另有 **Stack B 侧**的 `internal/controller/{vlan,ifm,system,interfaces}` reconciler（内嵌 `GenericReconciler`，轮询设备路径、不 watch CR），归属 `yang-controller-runtime/design.md`。`internal/controller/vlan/actor_reconciler.go` 是唯一 watch **新** `api/biz/v1.BusinessVlan` 的实验实现，但其 `k8sConfigStore` 为 stub。

## 6. 装配

- **`cmd/controller/main.go`（legacy 生产入口）**：`ctrl.NewManager`（`:45`）注册 `api/v1` scheme（`:27`），建一个 `DefaultClientPool`（`:55`），为 5 个 `controllers.*Reconciler` `SetupWithManager`（`:58-105`，均注「仅支持华为」）。经 Actor + 翻译引擎下发。
- 仅新树有 `config/crd/bases/*.yaml`：`biz.usmp.io_business{routes,switches,vlans,interfaces}.yaml` + `core.usmp.io_nativedeviceconfigs.yaml`。**旧树无 CRD YAML → 未发布**，进一步佐证 `api/v1` = legacy。

## 7. as-built 缺口 / 迁移债

| # | 债项 | 位置 |
|---|------|------|
| D1 | 双 CRD 树抢注同 group、schema 不兼容 | `api/v1` vs `api/biz/v1` |
| D6 | NativeDeviceConfig 下发 = TODO stub | `nativedeviceconfig_controller.go:223` |
| — | 生产控制器仍绑 legacy `api/v1`，新树仅被实验 reconciler 消费 | `cmd/controller/main.go` |
| — | NativeDeviceConfig 双语义（core/v1 模型化 vs api/v1 透传） | 两处 types |

## 8. 关联
- `translation-engine/design.md`（意图→厂商 YANG）、`actor-transaction/design.md`（下发机制）、`frontend/design.md`（CRD schema→表单）、`yang-controller-runtime/design.md`（权威替代栈）。
