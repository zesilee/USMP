# tasks — netconf-get-state-read

> TDD（T01/T05）：每组先写红灯测试再实现。测试层按 §5.6：B1（sim/client 纯逻辑+race）、B2（集成模拟网元）、B3（API handler）。

## 1. 模拟网元：状态 overlay 与 `<get>`（NS-08）

- [x] 1.1 B1 红灯：treeDatastore 状态树测试——SetState 注入、GetFiltered 合并（list 条目按键并入、无匹配 list 条目丢弃、纯状态容器并入、同名叶状态覆盖）、未注入时 get 等值 get-config、EditConfig/Commit/Discard 不触碰状态树、并发 get×写 race 干净
- [x] 1.2 实现 treeDatastore.state + SetState + mergeState walk（复用 findMatch/wellKnownListKeys 机制，D1/D2），1.1 转绿
- [x] 1.3 B1 红灯：server 层 `<get>` 分发测试——classifyRPC 识别 rpcGet、handleGet 套 subtree filter 返回合并树、get-config 回复不含状态叶、ErrorOnRPC 注入 get 生效
- [x] 1.4 实现 rpcEnvelope.Get/rpcGet/handleGet + Simulator.SetStateDataXML 门面 + ScenarioConfig 支持 get 注入（D3），1.3 转绿

## 2. 客户端：WithStateData option（DP-09）

- [ ] 2.1 B1 红灯：GetOptions.IncludeState 测试——WithStateData 置位走 `<get>`、缺省仍 get-config、get 路径传输层错误重连重试一次
- [ ] 2.2 实现 WithStateData + NETCONFClient.Get 按 option 选 driver.Get/driver.GetConfig（D4），2.1 转绿
- [ ] 2.3 B2 红灯→绿：集成测试（`*_integration_test.go`，testing.Short 跳过）——sim 种配置+状态 overlay，client.WithStateData 回读经 xmlcodec 解码含 `dynamic` 状态叶；不带 option 回读不含

## 3. API：GET /config 携状态（BR-01）

- [ ] 3.1 B3 红灯：config_handler 测试——fetch 返回含状态子树的 XML 时响应 data 含 RFC7951 状态字段；无状态子树时响应与改动前等值（不构造占位）
- [ ] 3.2 实现 fetchFromDevice 追加 client.WithStateData()（D5），3.1 转绿
- [ ] 3.3 B2 端到端：sim（配置+状态 seed）→ manager/clientpool → GET /config 返回含 `dynamic` 字段；VLAN 场景（测试内种 vlan 配置+status 状态）同断言

## 4. 演示种子与 standalone 二进制

- [ ] 4.1 B1 红灯→绿：DemoStateSeed 常量 + seed 测试（5 条接口 dynamic 全给值、数字枚举形态断言、与 DemoSeedConfig 键对齐）
- [ ] 4.2 cmd/netconf-simulator 启动注入 DemoStateSeed；审 staging-smoke.spec.ts 既有断言不受新增字段影响，追加只读字段有值断言（F4，`make e2e-local` 全绿）

## 5. 收尾门禁

- [ ] 5.1 `go test ./...` 全绿 + `-race` 干净（R09）；覆盖率不低于 backend/.coverage-baseline（T08，补测后上调）
- [ ] 5.2 go-code-review-check 通过（T04）；What/Why/How 提交（≤500 行/commit，超出拆分）
