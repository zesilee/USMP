---
name: arch-optimization-roadmap
description: 两配置面架构优化路线：P1/P2/P3 全完成、Stack A 载体清零、D1-D9 与 D3/D5 空转债全部勾销(2026-07-18)——历史脉络参考,新债另立
metadata: 
  node_type: memory
  type: project
  originSessionId: 885fb078-4483-407e-b202-c54d39217185
---

USMP 两个业务场景对应两个「配置面」，架构优化按面分阶段推进（见已归档 change `openspec/changes/archive/2026-07-04-device-native-lowcode-config`）：

- **场景① 设备原生面**（NETCONF/YANG 直配，YANG 驱动低码）→ 落 R01 权威的 Stack B。
- **场景② 意图/自动化面**（K8s CRD 声明 → translator → 设备配置，CRD 驱动低码）→ 现为 Stack A（legacy）。

三项已定架构决策：① 意图存储 K8s CRD 保留但 desired 投影进内存 ConfigStore（etcd 仅存意图，符 R03）；② Actor/2PC 退役，统一走 Stack B `ConfigStore→GenericReconciler`；③ 设备 schema 混合源（NETCONF capabilities 定模块集合 + ygot 模型树定属性 schema）。

**P1（场景① 真低码）已完成**（PR #27–#37）：修迁移债 **D4**（`internal/yangschema.Load` 构建 schema 树 + manager `WithSchema` + capabilities 收敛）、**D7**（ConfigStore 基于 `cache.Keys` 枚举）、**D9 设备侧**（删 `components/yang/*`）；yang-api 从 schema 树动态生成 FieldDef（vlan 出 23 字段）；config-api 通用 path↔ygot 编解码（`config_codec.go` 注册表 + `ygot.Unmarshal`）；前端 `useConfigPage` 原生分支修信封解包 bug 接动态 schema。

**P1 遗留已全收口（2026-07-17，change rfc7951-input-contract，PR #197-#199）**：配置写入收敛为单一 RFC7951 契约——body=以 path 为根的子树，服务端按 driver 描述符 `EncodeAnchor`（DR-05）锚点相对包裹后根级 ygot.Unmarshal，未注册/不可解显式 400（raw-map 静默入库消灭）；desired 存储/触发 key 归一化为锚点（子路径下发周期对账可见）。task 2.5（yang-api alias 假 schema）、3.6（convertToTypedStruct 685 行）、yang-schema.ts/DeviceConfigPage 全删。

**P2（场景②意图面收编）部分完成**（PR #38–#44，change `archive/2026-07-04-crd-intent-source-stackb`）：新增 `pkg/yang-runtime/source.KubernetesCRDSource`（C4 泛用 translate-and-project 源，app 提供 `ProjectFunc`）；`internal/crdsource` 为 BusinessVlan/Interface 提供 ProjectFunc（**用 api/v1**——translator/Actor 同树）+ `RegisterIntentSources`（单 K8s cache 双源，无集群优雅跳过）；`backend/main.go` 注册，与 Actor 并行、双路径 `reflect.DeepEqual` 验证 desired 等价；删 `controllers/businessvlan_controller.go`+`businessinterface_controller.go`（共享 retry 助手抽到 `controllers/retry.go`），去 cmd/controller 对应注册。**D2 部分退役**。

**P2 剩余被 D8 阻塞（→P3）**：BusinessSwitch 走 System 翻译=stub（D8），无 CRD 源替代 → 整包退役 `pkg/yang-runtime/actor`（D2 完成）+ 退 `cmd/controller` + 收敛 CRD 树 api/v1→api/biz/v1（D1）均需先补 System ygot 翻译。Route 翻译现返回裸 map（R04 故不投影）。用户决策：只删已验证等价的 Vlan+Interface 路径。

**P3 首里程碑达成**（PR #45–#46，change `archive/2026-07-04-retire-actor-usage`）：**纠正 D8 误判**——BusinessSwitch 是设备生命周期 CRD（借 Actor 探活，非配置意图），无需 System 翻译。`businessswitch_controller.probeDevice` 改 `ClientPool.Get`+`IsConnected` 直连探活（连接成功=在线，失败=离线 R08）；删死码 `internal/controller/vlan/actor_reconciler.go`。→ `pkg/yang-runtime/actor` **零非测试引用，R01（禁 Actor）实质满足、D2 达成**。

**Stack A 全退役、真·单进程达成**（PR #47–#49，change `archive/2026-07-04-retire-stack-a-entry`）：`cmd/controller` 3 控制器均冗余/空壳（Switch=探活写 CR status、Route/Native 下发=TODO）。DeviceHandler `/api/devices` 补 `probeOnline`（ClientPool.Get+IsConnected，REST 暴露在线状态）承接 Switch 探活 → 删 `cmd/controller/main.go` + `controllers/*`（Business{Vlan,Interface,Switch,Route}/NativeDeviceConfig 5 控制器 + retry.go），`controllers` 包清空。**`backend/main.go` 为唯一生产入口**（原生面 reconciler + CRD 意图源 + 北向 REST + 设备注册/探活）。前端不依赖 BusinessSwitch CR status，无损失；CRD 类型 api/v1 暂留（D1）。

**D1 双 CRD 树收敛已达成**（PR #50–#52，change `archive/2026-07-05-converge-crd-tree-bizv1`）：发现 translator/crdsource 绑 `api/v1`、但部署的 CRD YAML 出自 `api/biz/v1`（用户实际创建的 schema）→ 解码错位（biz/v1 的 BroadcastDiscard/MacLearning/UnknownMulticastDiscard/L2 模式被静默忽略）。修复：translator（huawei_vlan/interface/huawei.go）+ crdsource 原子迁 `api/biz/v1`、重写字段映射（VLAN 1:1、Interface 收敛 L2、Route map 用 biz/v1 字段）+ 补 translator TDD 单测 → 删 `api/v1` 整包（1208 行分 2 批，按 SyncPhase 等共享类型依赖顺序：先 Route/Switch/Native，后 Vlan/Interface/groupversion）。CRD 树统一 `api/biz/v1`+`api/core/v1`。

**actor 物理删除已勾销（2026-07-17，change retire-businessvlan-bridge，PR #186-#191）**：`pkg/yang-runtime/actor`（4718 行）、`internal/crdsource` 旧桥接、`pkg/translator`、`api/biz/v1` 四旧 Kind 全部物理删除；BR-04 厂商门禁改 `driver.VendorSupported`（DR-04）；意图面唯一路径=`internal/intent`（BusinessVlanService）。配套 TM04 契约变更（用户批准）：pr-size/commit-msg 增设纯删除豁免（insertions≤50 上限 6000）。**D1 终局 + 残余清零（2026-07-17，retire-stacka-residue PR #194-#196）**：NativeDeviceConfig 类型闭包（types+zz+types_common/SyncState）、backend/{deploy,config} 目录、backend/test/{e2e,integration} B0 载体、backend/Makefile ~30 个死 target 全部物理删除。**D3/D5 已清（2026-07-18，retire-idle-scaffolds PR #200-#201）**：plugin 整包删除（Manager 接口收口，扩展点由真实需求驱动再设计）；gNMI 空壳 client+订阅源删除，factory 对 GNMI/AUTO-9339 显式「尚未实现（规划能力）」错误（假成功路径消灭，CRD 枚举/协议规划身份保留）。**Stack A 载体清零 + 架构优化路线全部债务清零。** **openconfig 模型支持已退役（2026-07-18，retire-openconfig-models PR #202 + 归档 #203）**：generated/openconfig 包、controller/interfaces 死 Reconciler、netconf.go openconfig 特判分支、internal/yang/models 下 openconfig .yang 全部物理删除；契约固化为 yang-api BR-11（schema 树仅 huawei+usmp，vendor 仅两值）+ yang-xml-codec XC-04（降级链=注册表→xml.Marshal，无 openconfig 分支）。ietf 无独立支持：代码中 ietf 仅协议命名空间（保留）+ yang-models submodule 构建期依赖（保留，submodule 不裁剪）。后端覆盖率棘轮 69.7。

相关：[[dual-stack-migration]]、[[frontend-ci-gotchas]]、[[openspec-cli]]。
