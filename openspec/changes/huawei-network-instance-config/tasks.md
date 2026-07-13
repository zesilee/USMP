## 1. worktree 与基线（TDD 红灯先行）

- [x] 1.1 `EnterWorktree` 建隔离分支 `huawei-network-instance-config`（先 `git fetch`，从 origin/main 建），验证目录已在 `.gitignore`
- [x] 1.2 基线 `go test ./...`（含 `-race` 关键包）全绿，记录覆盖率基线（`backend/.coverage-baseline`）
- [x] 1.3 产出 `yang-config-test-design` 完备测试矩阵清单（对照 NI-05：全属性/端到端/并发/边界/嵌套 list 增删改/幂等/负路径），红灯先行——先写失败用例再实现（T05）

## 2. ygot 生成（R04 regen-and-diff）

- [x] 2.1 `backend/internal/generated/huawei/gen.conf` 的 `modules` 增加 `huawei-network-instance`
- [x] 2.2 `make gen-yang`（或 `VENDOR=huawei` 单包）重生成，经 CG-02 确定性规范化
- [x] 2.3 验证 `regen-and-diff` 零漂移、`go build ./...` 全通过、`HuaweiNetworkInstance_NetworkInstance` SchemaTree 入口存在（勿手改 generated/，R04）

## 3. 驱动描述符 + 单测（B1）

- [x] 3.1 `internal/drivers/huawei.go` 新增 namespace 常量 `HuaweiNetworkInstanceNS = "urn:huawei:yang:huawei-network-instance"` 与 `xmlcodec.Spec`（entry `HuaweiNetworkInstance_NetworkInstance`）
- [x] 3.2 注册单条 `driver.Descriptor{Vendor:"huawei", Module:"network-instance"}`：`MatchRoute`/`MatchDecode`/`MatchEncode` 均 `HasPrefix("/ni:network-instance")`，`ControllerToken:"network-instance"`，`NewStruct` 返回 `*HuaweiNetworkInstance_NetworkInstance`（NI-03）
- [x] 3.3 xmlcodec 往返真值单测（RFC7951→XML 带 namespace→回读 DeepEqual，断真值非仅非空）：**当场验证嵌套 list（`instances/instance`）编解码**——若报容器根/嵌套 list 缺口则记录，进入组 3b（NI-01/NI-04）
- [x] 3.4 完备性 schema 驱动枚举单测：按 `module:"huawei-network-instance"` 过滤枚举 config-true 标量，赋值→编码→解码整体 DeepEqual + **计数断言 = 5**（NI-01）
- [x] 3.5 谓词单测：`/ni:network-instance` 及子路径命中；注册可达性（空白导入触发、不落 xml.Marshal 兜底）（NI-03）
- [x] 3.6 namespace 单测：编码报文根节点携带 `urn:huawei:yang:huawei-network-instance`（NI-02）

## 3b. （条件）容器根嵌套 list 编解码缺口修复

- [x] 3b.1 (未触发·无缺口) **仅当 3.3 实测暴露缺口**：在 `pkg/yang-runtime/xmlcodec` 补容器根子容器内嵌 list 的 encode/decode（复用 list-center 机制），加对应 golden/往返用例
- [x] 3b.2 (未触发·无 delta) 若触发，补 `yang-xml-codec` spec delta（ADDED/MODIFIED，第 6 处 list-中心缺口），走 `go-code-review-check`

## 4. Reconciler + 注册（B2 集成）

- [x] 4.1 新增 `internal/controller/networkinstance/reconciler.go`：copy `internal/controller/bgp` 容器根收敛模式，`NetworkInstancePath="/ni:network-instance"`，GoStruct `*HuaweiNetworkInstance_NetworkInstance`（D3）
- [x] 4.2 `diffEngineAdapter.Diff` 检出任一漂移收敛为单条整根 change（下发整个 desired，编码为单条 `<network-instance>…`）；reconciler 只 set 原生字段，augment 字段保持 nil（NI-03 负路径）
- [x] 4.3 `backend/main.go` 注册 network-instance 控制器
- [x] 4.4 `reconciler_integration_test.go`（B2，`testing.Short()` 跳过）：下发→回读→二次收敛无 change；含多条 instance；确认模拟网元无需专用方言（NI-04）
- [x] 4.5 幂等集成用例：连续两次 reconcile 第二次无新增 change（NI-04）

## 5. 完备矩阵补齐（NI-05 / NI-06，缺层禁合并）

- [x] 5.1 并发-race：多协程并发对不同 instance reconcile，`go test -race` 无竞态无 panic（NI-05）
- [x] 5.2 嵌套 list 增删改：新增/修改/删除 instance 条目，往返 DeepEqual + 设备侧收敛到目标集合（NI-05）
- [x] 5.3 边界与负路径：name>31 / description>242 或含 `?` / 非法 ipv4 → 校验拦截或明确错误；下发失败缓存不更新（NI-05）
- [x] 5.4 `_public_` 删除降级：对 `name='_public_'` 删除不下发 node-delete、降级保留（NI-06 负路径）
- [x] 5.5 config-false 不误下发：`sys-router-id`/`vrf-id` 不出现在下发报文（NI-06 负路径）
- [x] 5.6 覆盖率不下降（T08）：补测后 `backend/.coverage-baseline` 不低于基线，达标则上调

## 6. 评审与提交

- [ ] 6.1 `go-code-review-check` 全项通过（架构合规/ygot 类型安全/NETCONF/并发安全）
- [ ] 6.2 `go test ./...` 全绿（含 `-race`），B2 集成通过
- [ ] 6.3 `git-what-why-how-commit` 三段式提交，单 commit ≤500 行按功能原子拆分
- [ ] 6.4 `superpowers:finishing-a-development-branch`：push + 开 PR（PR ≤1000 行、CI required checks 全绿）

## 7. sync + archive（本 change 合入后另起）

- [ ] 7.1 `/opsx:sync`：`huawei-network-instance-config` delta 合入主 spec（新建 `openspec/specs/huawei-network-instance-config`；若组 3b 触发则同步 `yang-xml-codec`）
- [ ] 7.2 `/opsx:archive`：change 移入 `openspec/changes/archive/`
- [ ] 7.3 更新记忆 `bgp-onboarding-deps`：network-instance 已交付、解锁 2a 基础邻居
