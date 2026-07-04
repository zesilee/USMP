# crd-intent-source-stackb — tasks（P2 场景② 意图面收编 Stack B）

> 依赖：`proposal.md` `design.md` `specs/*`。TDD（T01/R06）：先测试后实现。渐进迁移（§5.3）：旧留→并行→双路径验证→切换→删。
> 单 commit ≤500 行；PR ≤800（>20 文件则 ≤3000）；每步 `go test ./...` 全绿方进下一步。退役 Actor 风险最高——每 CRD 双路径验证等价后再删对应用法，最后删包。

## 0. 基线

- [ ] 0.1 基线 `go test ./...` 全绿；快照 Stack A（Actor）路径对 BusinessVlan 产出的 desired ygot 作对拍基准
- [ ] 0.2 审计 controllers/* 的 Actor 调用点、translator 消费点、api/v1 vs api/biz/v1 引用，列迁移清单

## 1. KubernetesCRDSource（C4，translate-and-project）

- [ ] 1.1 先写测试：源 add/update→`TranslateConfig`→`ConfigStore.Set`→`Enqueue`；delete→`ConfigStore.Delete`→Enqueue；deviceID/path 提取；translator 错误降级（R08）
- [ ] 1.2 实现 `pkg/yang-runtime/source/KubernetesCRDSource`：参数化（GVK/vendor/configType/deviceIDFn/pathFn），基于 controller-runtime informer/client-go watch，实现 `controller.Source`
- [ ] 1.3 单测用 fake K8s client（controller-runtime fake）驱动事件，断言 ConfigStore/enqueue 效果，含 -race

## 2. BusinessVlan 接入 Stack B（并行 + 双路径验证）

- [ ] 2.1 `backend/main.go` 为 BusinessVlan 注册 CRD 源（vendor=huawei, configType=vlan, deviceID=spec.deviceID, path=/vlan:vlan/vlan:vlans），与 Stack A Actor 路径**并行**
- [ ] 2.2 双路径对拍测试：同一 BusinessVlanSpec，Actor 路径 desired vs CRD-source 路径 desired **语义等价**
- [ ] 2.3 集成测试（netconfsim，T02）：创建 BusinessVlan CR（fake K8s）→ CRD 源 → reconcile → netconfsim 落配 → 断言与 Actor 路径一致

## 3. 迁移 Interface/Route/Switch

- [ ] 3.1 BusinessInterface 接入 CRD 源 + 双路径验证 + netconfsim 集成
- [ ] 3.2 BusinessRoute/Switch：补齐 Route/System ygot 翻译（或显式标注受限）；接入 + 验证
- [ ] 3.3 NativeDeviceConfig（core.usmp.io）如需，评估经 CRD 源或维持 config-api（记边界）

## 4. 切换生产入口 + 退役 Actor

- [ ] 4.1 生产入口切到 `backend/main.go` 单进程跑全部 CRD 源 + 原生面 + 北向 API（K8s informer 与 Stack B manager 生命周期协调、优雅退出）
- [ ] 4.2 删 `controllers/*` 的 Actor 调用（改为 CRD 源已承接，controllers 逻辑迁移/删除）
- [ ] 4.3 删 `pkg/yang-runtime/actor` 包（D2 退役，合 R01）；删 `cmd/controller/main.go` 入口
- [ ] 4.4 全量 `go test ./...` 绿（含 netconfsim 集成，-race）；`go build ./...` 绿

## 5. 收敛 CRD 树（解 D1）

- [ ] 5.1 退役 `api/v1`（旧 BusinessVlan 等）；统一 `api/biz/v1`+`api/core/v1`；清理 scheme 注册与引用
- [ ] 5.2 更新 CRD 生成 YAML（`config/crd/bases`）与部署清单一致性；grep 无 `api/v1` 残留

## 6. 收尾与验收

- [ ] 6.1 全量 `go test ./...` + `go build ./...` + `go build ./cmd/netconf-simulator` 绿；前端 vitest 绿
- [ ] 6.2 更新 `openspec/specs/{yang-controller-runtime,business-crd,actor-transaction,translation-engine,system-architecture}` 主 spec；勾除迁移债 D1/D2
- [ ] 6.3 验收（design §Goals）：CRD 意图经 Stack B 单进程下发；Actor/`cmd/controller`/`api/v1` 已删；两配置面汇入同一核心
- [ ] 6.4 满足 R01（删 Actor、收编 Stack B）/R03（意图投影内存 ConfigStore）/R04（ygot desired）/R06（TDD）；不碰场景①（P1）
