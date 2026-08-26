# Tasks: config-write-validation

> 全程 TDD（T01/T05）：每步先写测试再写实现。存量改造按 §5.3 军规：
> 旧实现保留 → 新实现并行 → 双路径验证 → 切换 → 删除旧实现。

## 1. 校验实现迁入 `schema` 包（解循环依赖）

- [x] 1.1 先跑基线：记录 `pkg/yang-runtime/validate` 现有用例全绿状态（等价性基准）
- [x] 1.2 校验实现整体迁入 `pkg/yang-runtime/schema`，导出为 `ValidateObject(Node, interface{}) error`；
      **语义冻结包注释（mandatory 不校验 / min-elements 仅对存在的空 list / must·when 不求值）
      随代码原样迁移，一字不改**（D2）
- [x] 1.3 迁移 `validate` 包的全部用例到新位置，断言逐条通过（等价性看门人）
- [x] 1.4 `validate` 包暂留为薄转发（并行期），确认 `internal/intent` 行为不变
- [x] 1.5 三处调用点改指新位置：`internal/intent/cr.go`、`intent/validate_dual_test.go`、
      `controller/networkinstance/validation_test.go`
- [x] 1.6 确认 `internal/intent` 的快照/双跑用例（`validate_snapshot_test`/`validate_dual_test`）全绿
      —— 这是「意图层结论未漂」的硬证据
- [x] 1.7 删除 `validate` 包（切换完成后）

## 2. 填实 `DefaultSchema.Validate`

- [x] 2.1 先写用例：路径不存在 → 报错；路径存在且配置合法 → nil；路径存在但配置违约 → 报错并指明叶路径
- [x] 2.2 `Validate` 委托到 `ValidateObject`，删除 TODO 与「will be implemented in a later iteration」
- [x] 2.3 同步更新 `openspec/tasks/code-todo-backlog.md` A2 条目状态

## 3. 配置写路径接入（本次的用户可感知改动）

- [x] 3.1 先写 B3 用例：违反 YANG 约束的配置提交 → `code=400`，且断言**零副作用**
      （ConfigStore 未写入、未触发对账、未触达设备）
- [x] 3.2 先写 B3 用例：合法配置提交 → 照常成功（防止接入把正常路径打死）
- [x] 3.3 `SetConfig` 在 `convertConfig` 之后、`storeConfigMerged` 之前接入校验（D3 顺序）
- [x] 3.4 确认校验对象是**本次提交的 incoming**、不是 merge 后的全量 desired（D4）
- [x] 3.5 保留既有 VLAN ID 硬编码校验（覆盖面不重叠，D3）

## 4. 暴露拒绝面并逐条定性（D6 风险闭环）

- [x] 4.1 跑全量 `go test -race ./...`：31 包全绿、零失败
      （31 = 基线 32 − 已删除的 `validate` 包）
- [x] 4.2 模拟网元端到端集成用例随全量跑通，种子配置未被拒
- [x] 4.3 `make e2e-local` Playwright staging smoke（结论见下）
- [x] 4.4 逐条定性（见「拒绝面实测结论」）
- [x] 4.5 未发现校验器缺陷；发现的是**测试载荷**错误，已订正

### ⚠️ 关键方法论教训：全量测试全绿 ≠ 校验没问题

开始时全量测试直接全绿，一度以为拒绝面为空。**这是假的安全感**——
`internal/api` 下几乎所有用例都用 `manager.New()`，那是**空 schema**，
新接入的校验在它们里面根本不执行。生产才经 `manager.WithSchema(yangschema.Load())`
装载真实 IR。

补 `config_validation_no_false_reject_test.go` 用**真实装载的 schema** 重测才拿到
真实拒绝面，并配一条**反向探针**（喂已知违约值必须被拒）防止「校验静默跳过 →
全过 → 假绿」这个陷阱重演。新增模块接入设备配置时，把合法载荷补进该表。

### 拒绝面实测结论

| 被拒输入 | 定性 | 处置 |
|---|---|---|
| `mac-address: "00:11:22:33:44:55"`（既有全属性测试载荷） | **校验器做对了** | 订正载荷 |

判据：华为 `pub-type:mac-address` 的 pattern 是 `[0-9a-f]{4}(-[0-9a-f]{4}){2}`，
即 `0011-2233-4455` 形态；该叶在 YANG 里明确是 **config** 叶（"Config MAC Address"），
冒号形态真机同样会拒。属「以前没校验、错了也没人发现」，不是校验器过严。
已把 `internal/api/config_handler_test.go` 与守护用例的载荷一并订正为华为形态。

用户侧影响面确认：前端把 mac-address 作为**只读状态字段**展示（`config=false` 的
`dynamic` 子树下另有同名叶），用户不会提交它；模拟器种子里的冒号 MAC 是设备**回读**
的状态数据，读路径不校验。故此项**不构成用户可感知的破坏**。

## 5. spec 同步与收尾

- [x] 5.1 delta spec 已先行产出（本 change `specs/` 下，R17 spec-first）
- [x] 5.2 覆盖率棘轮 75.0% → 76.8%，已上调 `backend/.coverage-baseline`
- [x] 5.3 `gofmt` 干净、`go vet` 无告警
- [x] 5.4 delta 已合入主 spec（config-api BR-08、yang-native-runtime YN-04），change 已归档

## 验收标准

- 违反 YANG 约束的配置提交被 400 拒绝，且零副作用（不写 desired / 不对账 / 不触达设备）
- 合法配置提交行为不变
- 意图层校验结论逐字节不变（快照/双跑用例为证）
- `Schema.Validate` 不再是空壳，`validate` 包已删除，全仓无循环依赖
- 拒绝面已逐条定性，无「合法配置被误拒」遗留
