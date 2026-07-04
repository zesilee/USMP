# system-architecture — 迁移债 / 补全清单（反向还原）

> 本清单为 as-built 与目标架构的差异 + 演进待办，非本次实施步骤。勾选 = 已在代码中解决。

## spec 与代码差异（迁移债）

- [ ] **D1 双 CRD 树抢注同 group**：`api/v1` 与 `api/biz/v1` 均注册 `biz.usmp.io/v1`，BusinessVlan schema 不兼容。（部分进展：CRD 意图源统一走 api/v1 与 translator/Actor 同树；收敛到 api/biz/v1 被 D8 阻塞——Switch/Route 仍用 api/v1，见 crd-intent-source-stackb）
- [x] **D2 Actor 子系统 vs R01**：**生产使用清零**——Vlan+Interface Actor 下发路径已删（crd-intent-source-stackb 组4）；BusinessSwitch 探活改 ClientPool 直连、删死码 `vlan/actor_reconciler.go`（retire-actor-usage）→ `pkg/yang-runtime/actor` 无任何非测试引用，R01 实质满足。物理删包（4709 行，`model_actor.go` 单文件 1089 > pr-size 800）为机械清理债，同 `datastore.go`/`yang-schema.ts`
- [ ] **D3 plugin 空转**：四类插件可注册但从不被调用
- [x] **D4 schema 层空转**：已消除——`internal/yangschema.Load` 从 ygot 模型构建 schema 树、manager `WithSchema` 挂载、设备 capabilities 收敛模块集合（device-native-lowcode-config）
- [ ] **D5 gNMI 空壳**：Get 发空 GetRequest，Set 发空 Path/Val
- [ ] **D6 NativeDeviceConfig 下发 = TODO**：`applyNativeConfig` 仅 time.Sleep
- [x] **D7 ConfigStore.List/ListDevices = stub**：已消除——基于 `cache.Keys()` 枚举设备/路径（device-native-lowcode-config）
- [ ] **D8 多厂商翻译仅 Huawei**：Cisco/H3C/Juniper 仅枚举占位
- [x] **D9 前端双代动态表单**：设备侧静态 YANG 渲染路径（`components/yang/*`、`useDeviceConfig`、`api/crd`）已删；yang-api 动态 schema 经 DynamicForm 渲染。（残留死类型文件 `types/yang-schema.ts` 待后续删）
- [x] **D10 两个模拟器并存**：已消除——删除 netsim，收敛为单一结构化 netconfsim，test-server 改内存 REST 桩（refactor-netconf-simulator）

## 改进建议（演进，需专门 /opsx:propose）

- [x] 确立单栈：生产入口从 cmd/controller/main.go(Stack A) 切到 backend/main.go(Stack B)——`cmd/controller` + `controllers/*`（Business{Vlan,Interface,Switch,Route}/NativeDeviceConfig 控制器）全部退役；设备探活收编进 DeviceHandler `/api/devices`；`backend/main.go` 为唯一生产入口（retire-stack-a-entry）
- [ ] 收敛 CRD 树：退役 api/v1，统一 api/biz/v1 + api/core/v1
- [ ] 偿还空转件：接通 plugin 钩子、加载 schema、补 gNMI 或明确弃用、实现 NativeDeviceConfig 下发
- [ ] 每条迁移遵循 §5.3「旧保留 + 新并行 + 双路径验证 → 切换 → 删除」
