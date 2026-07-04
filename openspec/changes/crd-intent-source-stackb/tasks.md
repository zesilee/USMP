# crd-intent-source-stackb — tasks（P2 场景② 意图面收编 Stack B）

> 依赖：`proposal.md` `design.md` `specs/*`。TDD（T01/R06）：先测试后实现。渐进迁移（§5.3）：旧留→并行→双路径验证→切换→删。
> 单 commit ≤500 行；PR ≤800（>20 文件则 ≤3000）；每步 `go test ./...` 全绿方进下一步。退役 Actor 风险最高——每 CRD 双路径验证等价后再删对应用法，最后删包。

## 0. 基线

- [x] 0.1 基线 `go test ./...` 全绿（已验证）；translator 输入为 `bizv1.BusinessVlanSpec`（对拍基准在组2 采集）
- [x] 0.2 审计：`controllers/*` 用 `actor.ModelActor`(Translate/Prepare/Commit 2PC)；translator `TranslateConfig(vendor,configType,spec)`；biz CRD 在 `api/biz/v1`；旧 `api/v1` 待退（D1）
- [x] 1.1 先写测试：`handleUpsert`→project(translate)→`ConfigStore.Set`→UpdateEvent；`handleDelete`→`Delete`→DeleteEvent；translate 错误/空 deviceID 降级（R08）；Start 无 cache 报错
- [x] 1.2 实现 `pkg/yang-runtime/source/KubernetesCRDSource`：**框架泛用**（`ProjectFunc` 由 app 提供 translate+提取，不 import translator/bizv1），`controller.Source`；Start 用 controller-runtime cache informer 挂 add/update/delete
- [x] 1.3 项目化核心（handleUpsert/handleDelete）用 fake ConfigStore 单测（-race 全绿）；informer 挂载为薄封装，端到端留组2（backend 接线）

## 2. BusinessVlan 接入 Stack B（并行 + 双路径验证）

- [x] 2.1 `internal/crdsource`：`VlanProjectFunc`（**用 api/v1，与 Actor/translator 同树**——D1 发现：translator 绑 api/v1，非 api/biz/v1）+ `RegisterVlanIntentSource`（graceful 无 K8s 跳过）；`main.go` 注册 CRD 源，与 Stack A Actor 路径并行
- [x] 2.2 双路径对拍测试：`VlanProjectFunc` 的 desired `reflect.DeepEqual` `translator.TranslateConfig(Huawei,Vlan,spec)`（Actor 路径翻译）——两路同调 translator 故等价
- [~] 2.3 reconcile→netconfsim 一段已由既有 Huawei VLAN reconciler 集成套件覆盖（同 ConfigStore→reconcile→netconfsim 同 desired）；ProjectFunc→ConfigStore 已单测；真·CR-watch e2e 需 envtest/集群，留后续

## 3. 迁移 Interface/Route/Switch

- [x] 3.1 BusinessInterface 接入：`InterfaceProjectFunc`（→ Huawei ifm ygot，path=/ifm:ifm/ifm:interfaces）+ register.go 收敛为**单 cache 双源**（Vlan+Interface）；双路径等价单测（DeepEqual translator）；reconcile→netconfsim 一段由既有 ifm reconciler 集成套件覆盖
- [~] 3.2 BusinessRoute/Switch：**显式标注受限**——`TranslateRoute` 返回裸 map、`TranslateSystem` 未支持（D5/D8）；按 R04 不把非 ygot desired 投影进 ConfigStore，故暂不注册其 CRD 源（register.go 注释说明），待补 ygot 翻译后接入
- [~] 3.3 NativeDeviceConfig（core.usmp.io）：维持 config-api 路径（场景①原生面，P1 已覆盖），不经 business CRD 源；记为边界

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
