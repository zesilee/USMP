## 0. 依赖确认（前置门禁，已在 design.md 依赖分析完成，实施时复核）

- [x] 0.1 复核 MVP 功能性配置依赖 = 零：确认公网 `/bgp:bgp/global`+`base-process` 全 rw 字段类型自包含于 huawei-bgp 主+type+common，无跨模型可配 leafref（唯一跨模型引用为 config-false 回读态）— 已核验（design 依赖分析矩阵）
- [x] 0.2 登记二期/三期前置依赖 DAG 到 roadmap — 已修正为核验后 DAG（network-instance 是 peering 唯一硬前置；routing 软 must 摘除；2a/2b 拆分），落 design/proposal/spec BGP-06

## 1. Spike：消解 gen 方案唯一阻塞风险（红灯先行）

- [x] 1.1 spike 完成：`make gen-yang VENDOR=huawei`（+huawei-bgp，不含 network-instance）生成成功（73543 行），结果=**ygot 不报错、不剪枝，物化整个 import 闭包**（含 network-instance/acl/bfd/routing 等兄弟根 struct）— 记录于 design「Spike 结论」
- [x] 1.2 import 闭包核验通过：无二级依赖缺失，`go build ./...` 全通过（R2 消解）
- [x] 1.3 入口名锚定：`type HuaweiBgp_Bgp struct` + `SchemaTree["HuaweiBgp_Bgp"]`（与 VLAN 同机制），全公网子树 struct 齐全（_Global/_BaseProcess/_Confederation/_GracefulRestart/_ReferencePeriod/_Timer/_DefaultParameter）— 回填 design D4
- [x] 1.4 结论分支：落 (a) 变体——D2 确认（modules 仅加 huawei-bgp）；~73k 行闭包生成权衡**用户已拍板接受**（2026-07-13），强制缓解=proposal/spec 登记 generated-but-not-integrated 边界（已落）

## 1B. genfix 确定性 schema 规范化（前置 G2 的共享基础设施，spike 暴露；CG-02 扩展）

> BGP 全闭包含多模块 augment 同一目标（bfd/ethernet/tunnel-management → ifm/network-instance），触发 goyang 非确定 augment 序 → schema blob 字节漂移 → R04 门禁永失败。必须先修 genfix 才能落 G2。

- [x] 1B.1 [红] genfix schema 规范化单测：确定性(5元素×4序,暴露 decorate-sort bug)/语义等价/数字保真(max uint64)/幂等——4 个新用例
- [x] 1B.2 [绿] 实现规范化：gunzip → UseNumber 解析 → 递归排序无序数组(Augmented)+对象键 → 固定参数 gzip → 回填；**关键修 decorate-sort 缺陷**(key+val 绑定同排,旧独立 keys 切片比较器错位)
- [x] 1B.3 基线：新 genfix 下基线 regen×2 确定(md5 稳定)、`go build`+API 测试(走 GzipToSchema)通过；注：基线 schema blob 一次性重规范化(合法可复现),非"字节不变"——语义等价才是保证(护栏已订正)
- [x] 1B.4 端到端确定性：+bgp regen×3 字节一致(消解 CG-03 阻塞)；**发现 Augmented 之外无其他非确定数组**(修 sort bug 后即收敛)；全量(含 openconfig)regen×2 确定
- [x] 1B.5 `yang-codegen-pipeline` delta(MODIFY CG-02)已随 specs 提交并 validate；commit cbf3c8f，go-code-review-check 通过

## 2. ygot 生成落地（R04）

- [x] 2.1 gen.conf 追加 `huawei-bgp`（无须附加模块，D2 确认），`make gen-yang` 生成；commit d8418bb
- [x] 2.2 regen-and-diff 零漂移：pre-commit R04 hook 通过、暂存后再 regen `git diff` 空；`generated/` 未手改；pr-size 排除生成物（hook 未拦体积）
- [x] 2.3 BGP-01 零漂移证据：regen×3 md5 一致 + pre-commit regen-and-diff 通过（已留证）

## 3. 驱动描述符谓词（B1 单测先行 → 实现）

- [x] 3.1 [红] 描述符谓词单测：dispatchEquivalence 表加 BGP 正路径 + 负路径（bgp-flow:/bgp-evpn:/per-VPN 均不误命中）；commit be95683
- [x] 3.2 [绿] 注册 `Descriptor{huawei/bgp}`：谓词用 **HasPrefix("/bgp:bgp")**（比 Contains 更精确，排除 per-VPN 与 feature 前缀）、显式 HuaweiBgpNS=urn:huawei:yang:huawei-bgp、Schema→SchemaTree["HuaweiBgp_Bgp"]
- [x] 3.3 注册可达性 + 全链路真值往返测试（RFC7951 解码→XML 编码带正确 namespace→回读等价），断言真值非仅非空
- [x] 3.4 **（计划外，quality gate 揪出）** 往返测试实测 Encode 报"no list map field"——xmlcodec 是 list 中心引擎、不支持容器根。已补 XC-05 plain-container 模式（commit 6113fe1）+ yang-xml-codec delta。列此为第二处"通用引擎不够通用"缺口（第一=genfix）
- [ ] 3.5 [债] 容器根 delete（EncodeDelete 目前对容器根返干净错误、未支持）→ 归入 group 4 BGP 删除语义设计

## 4. XML 编解码等价性（B1 + golden 方法论）

- [x] 4.1 编码/解码精确形态断言（container_test.go：根 namespace、嵌套、空自闭合、rpc-reply/data 包裹、namespace 前缀）——等价 golden 冻结意图；未另落 golden 文件（精确断言已覆盖）
- [x] 4.2 编码：完备性测试内断言根 namespace `urn:huawei:yang:huawei-bgp`、config-false 不入报文（commit 3fb9179）
- [x] 4.3 解码：container_test 解码 + 完备性往返；~~键式 delete~~ 容器根 delete 缺（EncodeDelete 返干净错误非 panic，API parseDeleteTarget 亦干净报错）→ **债留 group 4b**：BGP MVP 禁用走 enable=false modify，node-delete 后续单列
- [x] 4.4 **全属性可配（schema 驱动，反射枚举全 29 config-true 标量含 leaf-list，禁挑选）**：编码→解码→DeepEqual + 计数断言=29；当场揪出 default-parameter 误判 rw、confederation/as leaf-list 漏覆盖（commit 3fb9179，BGP-01 完整性）
- [x] 4.5 config-false 负用例：断言 default-parameter/error-discard-info/graceful-restart-status/vpn-brief-infos/remote-prefix-sid-states 不出现在 edit-config（commit 3fb9179，BGP-06）

## 5. 模拟网元 BGP 方言 + 端到端集成（B2）

- [x] 5.0 **（计划外必需）** BGP reconciler（internal/controller/bgp）+ main.go 控制器注册——BGP 配置光有描述符+编解码不到设备，需 reconciler 驱动对账;含 **D8 容器根 diff 收敛**（commit f632d84）
- [x] 5.1 [红→绿] `reconciler_integration_test.go`（`testing.Short()` 跳过）：下发公网 BGP→get-config 回读→二次收敛（Changes==0）；当场揪出第 5 处 list 中心缺口（写路径发 Go 类型名）
- [x] 5.2 netconfsim 无需 BGP 专门方言：sim 是通用 tree datastore（按 local 名存取），BGP 容器根 edit-config/get-config 直接跑通（on-wire XML 实证正确）
- [x] 5.3 幂等：二次对账 Changes==0 即幂等（同配置连下两次收敛，集成实证）
- [x] 5.4 下发失败降级：`ErrorOnRPC["edit-config"]` 注入 → 对账 result.Error 诚实报错、不 panic（BGP-05/§9，缓存不更新由 API 层保证）
- [ ] 5.5 [债/建议] 给 netconfsim 加 namespace 严格校验，堵上"namespace 正确性全测试套件测不出"盲区（本期 BGP 用权威 namespace，未实现严格校验；独立小改）

## 6. 完备测试矩阵补齐（yang-config-test-design / T02b）

- [x] 6.1 矩阵逐格核对（commit 8eed2fc）：全属性(组4 schema驱动29字段)/端到端(组5 B2)/嵌套(container_test)/幂等(组5 二轮0)/并发/边界/负路径(本组)——七格齐
- [x] 6.2 并发-race：64 协程并发编解码 BGP，`-race` 通过（引擎无共享状态，R09/BGP-05）
- [x] 6.3 边界+负路径：uint 最大值/leaf-list 多元素/XML 特殊字符转义往返；畸形 XML 报错不 panic/未知元素跳过（R08）。**约束校验澄清**：服务端不校 when/must（与 VLAN/IFM 一致，域约束除外），enable/as 耦合 must 与 as 的 when 由前端渲染+设备兜底——已用 FieldDef 透出用例锁死约束对 BGP 100% 数据驱动可达（§9/R05）
- [x] 6.4 覆盖率棘轮：聚合 60.6→实测 61.6，基线上调至 61.0（留 flaky 余量，T08）

## 7. 收尾与合规门禁

- [x] 7.1 `go test -race ./...` 30 包全绿；各代码单元 go-code-review-check 通过（T04/§6.2）
- [x] 7.2 verification：gen 零漂移(regen×3)、B2 集成收敛(Changes==0)、-race 全绿、覆盖率 60.6→61.0——新鲜证据齐
- [x] 7.3 16 原子提交(What/Why/How)；PR #146 人工~1941 行<3000 门禁(生成物排除)、CI 10/10 绿、已 merge(sha 2ae4ec1)
- [x] 7.4 `/opsx:sync` 三 delta 合主 spec(huawei-bgp-config 新建 / yang-xml-codec XC-05 / yang-codegen-pipeline CG-02) + `/opsx:archive` 归档（本收尾 PR）
