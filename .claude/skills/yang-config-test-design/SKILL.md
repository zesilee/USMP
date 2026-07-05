---
name: yang-config-test-design
description: 新增/接入任何 YANG 模型到设备配置(Stack B)时强制触发，产出完备测试用例矩阵，保证「全属性可配、端到端到设备验证、并发/边界/嵌套/幂等/负路径都有防线」的稳定交付。VLAN 交付中此矩阵暴露过 4 个真机高危 bug。
---

# 交换机 YANG 模型配置测试设计技能

> 沉淀自 VLAN 稳定交付：补测过程暴露并修复 4 个真机高危 bug（member-ports 下发崩溃、
> 增量配置抹除既有、嵌套字段拼写静默丢弃、并发配置竞态+丢更新）。证明「happy-path 管道
> 测试」不能保证交付。本技能把这套测试设计固化为**每个新 YANG 模型接入的强制门禁**。

## 一、激活时机（一定触发，禁止跳过）

出现以下任一情形，**必须**先按本技能设计完整用例矩阵再实现（TDD）：
- 新增/接入任何华为（或其它厂商）YANG 模型到设备配置管理（`convertMapToHuaweiXxx` / `mapEntryToXxx`）
- `buildYangSchema*` 支持新模块、`GetSchema` 新增 module 分支
- `DeviceConfigPage` 新增一条配置路由（新 module/configPath/listKey）
- 新增/修改 `/api/v1/config/*` 的类型编解码
- 修改配置下发链路（convert→merge→store→reconcile→NETCONF）任一环

交付未覆盖下表全部类别 → 视为**未完成，禁止合并**。

## 二、测试用例矩阵（每个新 YANG 模型必须全覆盖）

| 类别 | 层 | 用例 | 为什么必须（VLAN 真实教训） |
|------|----|------|------|
| **全属性到设备** | 集成(sim) | 一个实例配齐**所有可配置属性**(扁平枚举/数值/嵌套容器)，逐项断言到设备运行配置 | VLAN 曾静默丢 `unkown-unicast-discard`（converter 拼写错）；builder 曾漏字段 |
| **嵌套/list 到设备** | 集成(sim) | 含嵌套 list（如 member-ports）多条 + 端口模式枚举，断言落到设备 | VLAN member-ports 曾**下发直接崩溃**（marshal 不支持嵌套 map） |
| **合并防抹除** | 集成(sim) | 分两次配置不同实例(A、B) → 断言 **A 仍在**（不被 B 覆盖删除） | VLAN 曾配 B 抹掉 A（**数据丢失**，声明式对账把 desired 当完整状态） |
| **编辑保留属性** | 集成(sim) | 配全属性 → 回填完整条目改单项 → 未改属性不丢 | 编辑语义（整条替换）验证 |
| **幂等** | 集成(sim) | 同 desired 二次对账 → 无 error、无 requeue、状态不变 | 防重复下发副作用 |
| **读回一致** | 集成(sim) | 配置后读回设备运行配置，属性一致 | 前端 list/编辑依赖读回 |
| **边界拒绝** | 单元+集成 | key/范围越界（如 VLAN ID 0/4095、名称超长）→ 后端 400 拒绝 | YANG 常未编码业务范围，非法值会静默下发 |
| **前端表单校验** | e2e(真浏览器) | 缺主键/非法值点下发 → 被 `el-form` 拦截、行内提示、不提交 | §9：不提交、行内提示 YANG 约束 |
| **并发存储无竞态** | 集成(-race) | 并发 `storeConfigMerged` 配不同实例 → 无 DATA RACE、无丢更新、全部保留 | VLAN 曾并发下发**竞态+丢更新(R09)** |
| **畸形/负路径降级** | 单元+集成 | 畸形输入(坏 key 类型)不 panic、跳过坏条目；设备离线→503 保留原配置(R08) | 禁崩溃、异常必降级 |
| **动态表单渲染** | e2e(真浏览器) | 进配置页→选设备→新增→schema 驱动字段渲染出来 | VLAN 曾表单恒空(裸相对 fetch/接错架构) |
| **SPA 内导航** | 单元+e2e | 从别的配置页 SPA 内切到本页 → 加载**本模型**字段(非沿用) | 同组件多路由复用曾致 schema 不重载 |

## 三、实现要点与代码骨架（复用现有设施）

**后端集成测试**（`backend/internal/api/<model>_*_test.go`，复用 VLAN 的 helper 范式）：
```go
sim, cs, pool, deviceID := newXxxSimStack(t)   // 起 sim + cache + configstore + pool + deviceID
defer sim.Stop(); defer pool.CloseAll()
// 配置：走生产同款「合并存储」路径（带锁，防抹除/竞态）
typed, _ := convertMapToHuaweiXxx(map[string]interface{}{...前端形状,枚举用字符串名...})
storeConfigMerged(cs, deviceID, path, typed)   // internal/api/config_handler.go
xxxctl.New(cs, pool).Reconcile(ctx, reconcile.Request{DeviceID: deviceID, Path: path})
testsupport.AssertHuaweiXxx...(t, sim, ...)     // 断言到设备运行配置
```
- **必须扩展 `simulator/netconfsim/query.go`** 解析新模型的运行配置 + 在 `testsupport/asserts.go` 加 `AssertHuaweiXxx*`，否则「到设备」无从断言（VLAN 的 member-ports 就补过解析+断言）。
- 枚举用 `enumInt(v, "E_HuaweiXxx_...")`（数字直通 + `huawei.ΛEnum` name→value 反查）。
- **-race** 跑并发用例：`go test ./internal/api/ -run TestXxx.*Concurrent -race`。

**前端**：`useDeviceConfig` 参数化 + `DeviceConfigPage` props 接入；测试进 staging-smoke（真浏览器）。

## 四、必查高危陷阱清单（VLAN 4 个 bug 的通用化）

1. **嵌套/list marshal**：对账 diff 的实体 map 会走通用 `xml.Marshal`（不支持嵌套 map）→ 必须路由到专用 builder 并**补全该 builder 全字段**；写「嵌套到设备」用例兜住。
2. **YANG 字段拼写保真**：华为模型有真实拗口/错拼字段（如 `unkown-unicast-discard` 缺 n）。converter case、sim 解析、builder 三处**都要按 YANG 原拼写**，别"纠正"。写「全属性」用例兜住。
3. **合并语义**：声明式对账 desired=完整状态，UI 只发单个 → 必须 `storeConfigMerged` 并入既有；写「合并防抹除」用例兜住。
4. **并发存储**：Get-merge-Set 必须原子（`configMergeMu` 锁）+ merge 构造新副本（不原地改共享 map）；写并发 `-race` 用例兜住。
5. **枚举字符串**：表单发枚举名字符串，converter 必须 `enumInt` 双兼容；只认数字会静默丢弃。
6. **边界**：YANG 未编码的业务范围 → `validateConfig` 显式校验返回 400。
7. **裸相对 fetch**：前端取 schema/数据必须走 api 客户端（staging nginx 不代理 /api）。

## 五、交付门禁（Definition of Done，全 ✓ 才可合并）

- [ ] 后端全后端 `go test ./...` 0 失败；并发用例 `-race` 绿
- [ ] 二表「测试用例矩阵」每一类都有对应用例且通过
- [ ] sim 能解析本模型运行配置 + testsupport 有到设备断言
- [ ] 四高危陷阱各有一条用例兜住
- [ ] 前端 typecheck 0 + staging-smoke 有本模型「动态渲染 + SPA 导航 + 表单校验」真浏览器断言
- [ ] 用例遵循 TDD：先写（红）后实现（绿）

## 六、相关设施索引

- 测试范式：`backend/internal/api/vlan_hardening_integration_test.go`（newVlanSimStack/applyVlan/storeConfigMerged）、`vlan_comprehensive_test.go`、`vlan_validation_test.go`
- 设备断言：`backend/simulator/netconfsim/testsupport/asserts.go`
- 设备运行配置解析：`backend/simulator/netconfsim/query.go`（`RunningHuaweiXxxFull`）
- 配置链路：`backend/internal/api/config_handler.go`（convertConfig/mergeConfig/storeConfigMerged/validateConfig）、`pkg/yang-runtime/client/netconf.go`（marshalChange/buildHuaweiXxxXML）
- 前端：`frontend/src/composables/useDeviceConfig.ts`、`views/DeviceConfigPage.vue`、`tests/staging-smoke.spec.ts`
- 关联技能：`netconf-sim-integration-test`、`tdd-test-driven-dev`、`go-code-review-check`
