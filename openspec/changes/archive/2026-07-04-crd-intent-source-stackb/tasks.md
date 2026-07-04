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

## 4. 退役 Actor（部分——用户决策：只删已验证等价的 Vlan+Interface）

> BusinessSwitch 走 System 翻译（stub，D8），无 CRD 源替代——整包退役 Actor +
> 退 cmd/controller 被 D8 阻塞，留待 System 翻译补齐（P3 或 P2 扩展）。

- [x] 4.2a 删 BusinessVlan Actor 路径：删 `controllers/businessvlan_controller.go`，共享助手抽到 `controllers/retry.go`，去 cmd/controller 注册（组4a）
- [x] 4.2b 删 BusinessInterface Actor 路径：删 `controllers/businessinterface_controller.go`，去 cmd/controller 注册（组4b）
- [~] 4.1/4.3 **暂缓**（D8 阻塞）：保留 `pkg/yang-runtime/actor` 包 + `controllers/businessswitch_controller.go` + `cmd/controller` 入口，待 System ygot 翻译补齐后再整包退役 D2、切单入口
- [x] 4.4 全量 `go test ./...` + `go build ./...` 绿（每步验证）

## 5. 收敛 CRD 树（解 D1）—— 暂缓（D8 阻塞）

- [~] 5.1/5.2 **暂缓**：退役 `api/v1`、统一 `api/biz/v1` 需 translator + 全部消费方脱离 api/v1；而 Switch/Route 控制器 + translator 仍绑 api/v1（System 翻译=stub，D8）。CRD 意图源已与 translator/Actor 统一走 api/v1（无新增分裂）；完整收敛留 P3（随 D8 System 翻译一并做）

## 6. 收尾与验收

- [x] 6.1 全量 `go test ./...` + `go build ./...` 绿（每步 + 本组验证）；netconfsim 集成沿用既有套件
- [x] 6.2 `system-architecture/tasks.md` 标注 D1 部分进展、D2 部分退役；主 spec 因**部分退役**（Actor 未全删）不 sync `actor-transaction` REMOVED，留待整包退役时更新
- [x] 6.3 验收（部分）：**Vlan+Interface 意图经 Stack B CRD 意图源单进程下发**（双路径验证等价），其 Actor 路径已删；Switch/Route + actor 包 + cmd/controller 保留（D8 阻塞）；两配置面在 ygot 支持范围内汇入同一核心
- [x] 6.4 满足 R01（方向：删已验证的 Actor 路径、收编 Stack B）/R03（意图投影内存 ConfigStore）/R04（仅 ygot desired 投影，Route/System stub 不投影）/R06（TDD）；未碰场景①（P1）

## 遗留（→ P3，随 D8）

- System ygot 翻译（D8）→ BusinessSwitch CRD 源 → 删 Switch Actor 路径 → 整包退役 `pkg/yang-runtime/actor`（D2 完成）+ 退役 `cmd/controller` 入口（4.1/4.3）。
- 完整收敛 CRD 树 api/v1→api/biz/v1（D1 完成，5.1/5.2）。
- BusinessRoute ygot 翻译（现返回裸 map）。
