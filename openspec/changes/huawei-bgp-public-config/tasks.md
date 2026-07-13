## 0. 依赖确认（前置门禁，已在 design.md 依赖分析完成，实施时复核）

- [x] 0.1 复核 MVP 功能性配置依赖 = 零：确认公网 `/bgp:bgp/global`+`base-process` 全 rw 字段类型自包含于 huawei-bgp 主+type+common，无跨模型可配 leafref（唯一跨模型引用为 config-false 回读态）— 已核验（design 依赖分析矩阵）
- [x] 0.2 登记二期/三期前置依赖 DAG 到 roadmap — 已修正为核验后 DAG（network-instance 是 peering 唯一硬前置；routing 软 must 摘除；2a/2b 拆分），落 design/proposal/spec BGP-06

## 1. Spike：消解 gen 方案唯一阻塞风险（红灯先行）

- [x] 1.1 spike 完成：`make gen-yang VENDOR=huawei`（+huawei-bgp，不含 network-instance）生成成功（73543 行），结果=**ygot 不报错、不剪枝，物化整个 import 闭包**（含 network-instance/acl/bfd/routing 等兄弟根 struct）— 记录于 design「Spike 结论」
- [x] 1.2 import 闭包核验通过：无二级依赖缺失，`go build ./...` 全通过（R2 消解）
- [x] 1.3 入口名锚定：`type HuaweiBgp_Bgp struct` + `SchemaTree["HuaweiBgp_Bgp"]`（与 VLAN 同机制），全公网子树 struct 齐全（_Global/_BaseProcess/_Confederation/_GracefulRestart/_ReferencePeriod/_Timer/_DefaultParameter）— 回填 design D4
- [x] 1.4 结论分支：落 (a) 变体——D2 确认（modules 仅加 huawei-bgp）；~73k 行闭包生成权衡**用户已拍板接受**（2026-07-13），强制缓解=proposal/spec 登记 generated-but-not-integrated 边界（已落）

## 2. ygot 生成落地（R04）

- [ ] 2.1 正式在 `gen.conf` 追加 `huawei-bgp`（含 spike 结论要求的任何附加模块），`make gen-yang` 生成 `backend/internal/generated/huawei/*`
- [ ] 2.2 验证 regen-and-diff 门禁零漂移（本地 pre-commit 对称）；确认 `generated/` 未手改；确认 pr-size/commit-msg 已排除 `generated/`（design R4）
- [ ] 2.3 BGP-01 场景：ygot 生成物零漂移用例纳入验证证据

## 3. 驱动描述符谓词（B1 单测先行 → 实现）

- [ ] 3.1 [红] 写描述符谓词单测：`MatchRoute`/`MatchDecode`/`MatchEncode` 正路径（`/bgp:bgp/base-process/...` 命中）+ **负路径**（`bgp-flow:`/`bgp-evpn:`/`bgp-l2vpnad:` 不误命中）+ 未命中 `ok=false` 降级（BGP-03）
- [ ] 3.2 [绿] 在 `backend/internal/drivers/huawei.go` 注册 `Descriptor{Vendor:"huawei", Module:"bgp", ControllerToken:"bgp"}`：谓词锚定 `bgp:bgp`、显式 `HuaweiBgpNS = "urn:huawei:yang:huawei-bgp"` 常量、`Schema` 闭包指向公网根 SchemaTree 入口（BGP-02/BGP-03/D3）
- [ ] 3.3 注册可达性单测：断言 BGP 集成测试二进制空白导入 `internal/drivers`，`Lookup("huawei","/bgp:bgp/...")` 返回 `ok=true`（BGP-03，design R5）

## 4. XML 编解码等价性（B1 + golden 方法论）

- [ ] 4.1 [红] 用 `xmlcodec.Canonicalize` 建立公网 BGP 编码/解码 golden fixtures，落 `internal/testutil/hwfix`（`-args -update-golden` 刷新）
- [ ] 4.2 编码单测：`/bgp:bgp/base-process` 全字段编码→根 namespace 正确（BGP-02）、config-false 在发字段不被 schema 过滤（D5）
- [ ] 4.3 解码单测：running config XML → `HuaweiBgp_Bgp` 结构体，Decode 锚定顶层容器；键式 delete 场景（若小容器/leaf 涉及）
- [ ] 4.4 全属性可配单测：**从生成结构体枚举 `/bgp:bgp/global`+`base-process` 全部 rw 字段**（禁字段挑选），逐一编码下发+原值回读；含 confederation/graceful-restart/reference-period/timer/default-parameter 全部可配字段（BGP-01 完整性）
- [ ] 4.5 config-false 负用例：断言 `base-process` 下 config-false 态（vpn-brief-infos/graceful-restart-status/error-discard-info/remote-prefix-sid-states）不出现在 edit-config 下发报文（BGP-01/BGP-06）

## 5. 模拟网元 BGP 方言 + 端到端集成（B2）

- [ ] 5.1 [红] 写 `*_integration_test.go`（`testing.Short()` 跳过）：下发公网 BGP → get-config 回读 → Reconciler 收敛（BGP-04）
- [ ] 5.2 [绿] `simulator/netconfsim` 增加 BGP edit-config（整树替换，对齐既有 RFC 通道）+ get-config 回读方言
- [ ] 5.3 幂等集成用例：同配置连续下发两次，第二次判定 no-op（BGP-04）
- [ ] 5.4 下发失败用例：netconfsim 模拟 edit-config 失败 → 缓存不更新、保留原配置、明确错误码（BGP-05/§9）

## 6. 完备测试矩阵补齐（yang-config-test-design / T02b）

- [ ] 6.1 触发 `yang-config-test-design`，对照产出矩阵逐格核对：全属性/端到端/并发/边界/嵌套/幂等/负路径（T02b/BGP-05）
- [ ] 6.2 并发-race 用例：多协程并发 reconcile/编解码 BGP 路径，`-race` 通过（R09/BGP-05）
- [ ] 6.3 边界+约束负路径：越界 AS 号、违反 `must`/`when`（`enable=false` 带 `as`）→ 校验失败不崩溃（R08/BGP-05）
- [ ] 6.4 覆盖率棘轮：补测后后端 `backend/.coverage-baseline` 不下降，如上升同步上调基线（T08）

## 7. 收尾与合规门禁

- [ ] 7.1 `go test ./...` 全绿（含 `-race`）；`go-code-review-check` 技能通过（T04/§6.2）
- [ ] 7.2 `superpowers:verification-before-completion`：留新鲜验证证据（gen 零漂移、集成收敛、race 通过）
- [ ] 7.3 分拆提交（每 commit ≤500 行、单原子功能、What/Why/How 三段式）；PR ≤3000 行，超限按「gen 生成物→驱动+编解码→sim+集成」拆顺序 PR
- [ ] 7.4 `/opsx:sync` 把 `huawei-bgp-config` delta（及 spike 触发的 `yang-codegen-pipeline` delta，若有）合入主 spec；`/opsx:archive` 归档 change
