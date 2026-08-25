# system-architecture — 迁移债 / 补全清单（反向还原）

> 本清单为 as-built 与目标架构的差异 + 演进待办，非本次实施步骤。勾选 = 已在代码中解决。

## spec 与代码差异（迁移债）

- [x] **D1 双 CRD 树抢注同 group**：已消除——translator + crdsource 迁到部署权威 `api/biz/v1`（修复此前解码错位 bug：用户能设的 biz/v1 字段被忽略），删除 `api/v1` 整包（Route/Switch/Native/Vlan/Interface/groupversion），CRD 树统一 `api/biz/v1`+`api/core/v1`（converge-crd-tree-bizv1）
- [x] **D2 Actor 子系统 vs R01**：**生产使用清零**——Vlan+Interface Actor 下发路径已删（crd-intent-source-stackb 组4）；BusinessSwitch 探活改 ClientPool 直连、删死码 `vlan/actor_reconciler.go`（retire-actor-usage）→ `pkg/yang-runtime/actor` 无任何非测试引用，R01 实质满足。物理删包（4709 行，`model_actor.go` 单文件 1089 > pr-size 800）为机械清理债，同 `datastore.go`/`yang-schema.ts`
- [x] **D3 plugin 空转**：已消除——plugin 整包物理删除（retire-idle-scaffolds），扩展点由真实需求驱动再设计
- [x] **D4 schema 层空转**：已消除——`internal/yangschema.Load` 从 ygot 模型构建 schema 树、manager `WithSchema` 挂载、设备 capabilities 收敛模块集合（device-native-lowcode-config）
- [x] **D5 gNMI 空壳**：已消除——空壳物理删除（retire-idle-scaffolds），gNMI 定位为规划能力，工厂返回显式未实现错误（R02 口径）
- [x] **D6 NativeDeviceConfig 下发 = TODO**：已随 Stack A 控制器整体退役（retire-stacka-residue），原生配置通道由 Stack B 直连（config-api）取代
- [x] **D7 ConfigStore.List/ListDevices = stub**：已消除——基于 `cache.Keys()` 枚举设备/路径（device-native-lowcode-config）
- [x] **D8 多厂商翻译仅 Huawei**：已随 `pkg/translator` 整包删除而消解（retire-businessvlan-bridge）；厂商扩展现走驱动注册表（device-driver-registry，P5-4 拍板不接非华为设备、能力零成本待命）
- [x] **D9 前端双代动态表单**：设备侧静态 YANG 渲染路径（`components/yang/*`、`useDeviceConfig`、`api/crd`）已删；yang-api 动态 schema 经 DynamicForm 渲染。（残留死类型文件 `types/yang-schema.ts` 待后续删）
- [x] **D10 两个模拟器并存**：已消除——删除 netsim，收敛为单一结构化 netconfsim，test-server 改内存 REST 桩（refactor-netconf-simulator）

## 改进建议（演进，需专门 /opsx:propose）

- [x] 确立单栈：生产入口从 cmd/controller/main.go(Stack A) 切到 backend/main.go(Stack B)——`cmd/controller` + `controllers/*`（Business{Vlan,Interface,Switch,Route}/NativeDeviceConfig 控制器）全部退役；设备探活收编进 DeviceHandler `/api/devices`；`backend/main.go` 为唯一生产入口（retire-stack-a-entry）
- [x] 收敛 CRD 树：终态为仅存 `api/core/v1`（api/v1 随 converge-crd-tree-bizv1 删除，api/biz/v1 随 retire-businessvlan-bridge PR#186 删除）
- [x] 偿还空转件：plugin/gNMI 空壳物理删除、schema 已加载、NativeDeviceConfig 由 Stack B 直连取代（台账见上表）
- [x] 迁移全程遵循 §5.3 渐进策略，2026-07-18 架构优化收官（docs/memory/arch-optimization-roadmap.md）
