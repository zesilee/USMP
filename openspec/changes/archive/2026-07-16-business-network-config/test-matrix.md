# usmp-business-vlan 测试矩阵（T02b，yang-config-test-design 产出）

> 意图模型不直接下设备：意图 CR → 展开（expand）→ 每设备 huawei-vlan+huawei-ifm 原生片段 → 跨设备 2PC → 声明式稳态。矩阵按此两跳架构适配技能十二类 + 意图生命周期三类。
> 各项标注落地波次；**缺项=未完成禁止合并**（T06）。四高危陷阱（嵌套 marshal/拼写保真/合并语义/并发存储）各有兜底用例。

## A. 技能矩阵十二类的适配

| # | 类别 | 层 | 用例（本模型具体化） | 波次 |
|---|------|----|---------------------|------|
| A1 | 全属性到设备 | 集成(双sim) | 意图配齐 vlan-id+name+devices[A(access+trunk),B(trunk)] → 2PC → 两台 sim running 逐项断言 vlan(id,name) 与端口模式/PVID | 7.1 |
| A2 | 嵌套 list 到设备 | 集成(双sim) | devices 2 条、每条 access-ports/trunk-ports 各多口 → 逐设备逐口断言（ifm 精确叶子以 generated structs golden 锁定） | 5.1+7.1 |
| A3 | 合并防抹除 | 集成(sim) | 设备上既有手工 vlan 300 + 意图下发 vlan 100 → 300 仍在；意图 A(100)、B(200) 同设备 → 双双在 | 7.1 |
| A4 | 编辑保留属性 | 集成(双sim) | 改意图 name → 重展开收敛 → 端口配置不丢不重下 | 7.3 |
| A5 | 幂等 | 集成(双sim) | 同 generation 二次 reconcile → 不再触发 2PC、status 不变、无 error | 7.3 |
| A6 | 读回一致 | 集成(双sim) | 收敛后 sim 查询/config-api 读回与展开一致 | 7.1 |
| A7 | 边界拒绝 | envtest+单元 | vlan-id 0/4095 被 CRD OpenAPI 拒（apiserver 写入时）；devices 空(min-elements)/坏 IP pattern 拒；USMP API 写路径 400 | 6.1 |
| A8 | 前端表单校验 | F3/F4 | 业务控制台缺 vlan-id/越界值点提交 → el-form 拦截行内提示不提交 | 10.3/10.5 |
| A9 | 并发无竞态 | 集成(-race) | 并发多意图对同设备（每设备事务互斥）+ 并发 config-api 手改同设备 → -race 零竞态、无丢更新 | 7.1 |
| A10 | 畸形/负路径降级 | 集成(双sim) | ①prepare 部分失败→全体 discard 零残留 ②confirming 失联→超时回滚+status 不一致告警 ③:confirmed-commit 缺失→降级普通 commit+status 标注 ④设备离线→deviceStates failed+退避重试 ⑤must 违约 CR→Validated=False 零下发 | 7.1/6.1 |
| A11 | 动态表单渲染 | F4 | 业务菜单组出现→进业务控制台→schema 驱动渲染 devices 嵌套 list | 10.5 |
| A12 | SPA 内导航 | F2+F4 | 原生模块控制台 ⇄ 业务控制台切换，schema 不串台 | 10.1/10.5 |

## B. 意图生命周期三类（本模型新增维度）

| # | 类别 | 层 | 用例 | 波次 |
|---|------|----|------|------|
| B1 | finalizer 删除 | 集成(双sim)+envtest | 删 CR → 两台设备认领配置 DELETE 清除 → finalizer 摘除 CR 消失；单台离线 → finalizer 保留+status failed+恢复后重试完成 | 8.1 |
| B2 | 收缩差集 | 集成(双sim) | devices [A,B]→[A] → 仅 B 被清理、A 不动；差集仅依赖 CR status（换实例重放同结果） | 8.1 |
| B3 | 软归属 | B3(API) | 认领写入 status/查询 API 返回；config-api 手改命中→ownershipWarning；两意图认领同 (设备,vlan-id) → 后者 Validated=False | 9.1 |

## C. 展开引擎纯函数矩阵（先于集成，golden）

| 用例 | 断言 |
|------|------|
| 双设备全属性展开 | golden：N×(设备,模块,路径,片段) 逐字节稳定 |
| 幂等重入 | 同 spec 重复 expand 结果一致（含认领集合） |
| 空 devices / 单设备 / 仅 access / 仅 trunk | 边界形状正确，不 panic（R08） |
| name 缺省 | 落 VLAN<id> 缺省名 |

## D. 高危陷阱兜底映射

| 陷阱 | 兜底用例 |
|------|---------|
| 嵌套 marshal 崩溃 | A2（ifm 端口嵌套下发到设备） |
| YANG 拼写保真 | A1 全属性 + C golden（展开片段按 huawei 模型原拼写） |
| 合并语义抹除 | A3（意图 desired 走 storeConfigMerged 同款并入，不得整树覆盖手工配置） |
| 并发竞态丢更新 | A9（-race） |

## E. 前置设施缺口（波次 5/7 落地时补）

- `simulator/netconfsim/query.go`：`RunningHuaweiInterfaces` 需覆盖 l2 属性（link-type/PVID/trunk 放行表）断言字段。
- `testsupport/asserts.go`：新增 AssertHuaweiVlanWithPorts / AssertIntentConverged 类断言助手。
- envtest：意图控制器测试用 sigs.k8s.io/controller-runtime/pkg/envtest（CRD manifest 来自 deploy/crds/）。

## F. 交付门禁（DoD）

- [ ] A1-A12、B1-B3、C 全部有用例且绿（对应波次勾稽）
- [ ] 并发用例 -race 绿；全后端 `go test ./...` 0 失败
- [ ] sim 能解析/断言 ifm l2 属性；四陷阱兜底各一条
- [ ] 前端 typecheck 0 + staging-smoke 含 A8/A11/A12
- [ ] TDD：每项先红后绿
