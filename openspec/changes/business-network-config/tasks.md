# Tasks: business-network-config

> 波次即 PR 边界（TM04 ≤1000 行）；每任务 TDD 红绿（T01/T05），按 §5.6 选层补齐（T06）。

## 1. 模拟网元 confirmed-commit（NS-07，B2 前置）

- [x] 1.1 测试先行：sim confirmed-commit 三场景（确认转正/超时回滚/能力开关）红灯
- [x] 1.2 实现 netconfsim candidate 快照 + 确认计时器 + hello capability 开关，绿灯（-race）

## 2. NETCONF 客户端 2PC 原语（DP-08）

- [x] 2.1 测试先行：CommitConfirmed/ConfirmCommit 单测 + 对 sim 集成测试（含超时回滚回读、能力缺失负路径）红灯
- [x] 2.2 实现客户端原语（复用 opMu 串行化，不新增并发写路径），绿灯

## 3. YANG→CRD codegen（BIC-01/BIC-02）

- [x] 3.1 测试先行：crdgen golden 测试（类型映射矩阵：range/pattern/enum/leaf-list/嵌套 list+key 必填；不可映射类型 fail-fast）红灯
- [x] 3.2 实现 `tools/crdgen`（goyang Entry → CRD manifest，printer columns + status 子资源），生成物入库 + regen-and-diff 门禁接 CI

## 4. 意图 YANG 模型与管线接入（BVS-01，D2）

- [x] 4.1 编写 `usmp-business-vlan.yang`（vlan-id/name/devices{ip,access-ports,trunk-ports}，task-name=业务网络配置）
- [x] 4.2 gen.conf 接入 + ygot 生成 `internal/generated/business/` + `/yang/modules`、`/yang/schema` 暴露（含 category 断言测试）
- [x] 4.3 触发 `yang-config-test-design` 产出意图模型完备测试矩阵（T02b），矩阵项落入后续任务验收

## 5. 展开引擎（BIO-02，BVS-02）

- [x] 5.1 测试先行：expand 纯函数 golden 测试（双设备展开/幂等重入/空 devices 边界；ifm trunk 放行精确叶子以 generated structs 锁定）红灯
- [x] 5.2 实现 expand(spec)→[]Fragment 与认领集合推导，绿灯

## 6. 意图控制器与校验分层（BIO-01，BIC-03/BIC-04）

- [x] 6.1 测试先行：envtest/fake client——CR 创建触发 reconcile、无 kubeconfig 优雅降级、must 违约 Validated=False 不展开、status 结构（observedGeneration/conditions/deviceStates/claims）红灯
- [x] 6.2 泛化 crdsource 为意图 watch 源 + IntentReconciler 骨架 + 约束引擎 admission-by-watch 校验 + status patch 冲突重试，绿灯（旧 BusinessVlan 桥接并行保留）

## 7. 跨设备 2PC 下发（BIO-03/BIO-04，BVS-03/BVS-04）

- [x] 7.1 测试先行：TxCoordinator 对双 sim 集成测试——全体成功、prepare 部分失败全体 discard 不留残配置、confirming 失联超时回滚告警、能力缺失降级普通 commit、并发 -race 红灯
- [x] 7.2 实现 TxCoordinator（prepare→discard-all/CommitConfirmed→ConfirmCommit；成功才写 desired+TriggerReconcile；每设备互斥）
- [x] 7.3 稳态：意图周期 resync 重写 desired（对冲 TTL 过期）+ 漂移单设备修复集成测试

## 8. 生命周期（BIO-05/BIO-06）

- [x] 8.1 测试先行：finalizer 删除双设备清理、部分失败保留重试、devices 收缩差集孤儿清理（差集仅依赖 CR status）红灯
- [x] 8.2 实现 finalizer + DELETE 命令通道展开 + 上一代认领差集清理，绿灯

## 9. 软归属与 config-api（BIO-07，BR-11）

- [x] 9.1 测试先行：B3——认领索引聚合/重建、SetConfig/Delete 命中返回 ownershipWarning、未命中无警告、归属查询 API 红灯
- [x] 9.2 实现认领内存索引（watch 驱动）+ config-api 警告字段 + `GET /ownership`，绿灯

## 10. 前端（FE-17/FE-18）

- [ ] 10.1 测试先行：F2——BusinessConsolePage 列表/详情/表单三态、deviceStates 聚合呈现、校验错误态；F1——ownership/状态聚合 util
- [ ] 10.2 实现平台作用域业务控制台（表单复用 YANG 动态渲染，数据面走 USMP API 代理 CR CRUD）
- [ ] 10.3 F3 真浏览器：devices 嵌套 list 增删改全覆盖
- [ ] 10.4 原生控制台归属徽标 + 手改非阻断警告提示（F2）
- [ ] 10.5 F4 staging-smoke：业务菜单组出现、意图创建到收敛状态呈现（`make e2e-local` 全绿）

## 11. 部署与收官

- [ ] 11.1 CRD manifest + RBAC（biz CR/status + Lease）进部署编排，CRD 先于应用滚动；leader election 接缝（默认关）
- [ ] 11.2 覆盖率棘轮上调（T08）+ 全量 `go test ./...` -race + 前端全层绿
- [ ] 11.3 /opsx:sync 同步 delta→主 spec；修订 CLAUDE.md §8/R03 措辞与任务文件「禁止复活 CRD 通道」表述；回写 openspec/tasks/business-network-config.md
- [ ] 11.4 归档 change + 更新记忆（k8s-paas-deployment-constraints 交付状态；全局 HA/硬锁/旧桥接退役三项 follow-up 立任务）
