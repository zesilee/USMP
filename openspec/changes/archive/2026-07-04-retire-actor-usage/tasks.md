# retire-actor-usage — tasks（P3 消除 Actor 生产使用）

> 依赖：proposal/design/specs。TDD（R06）：先测试后实现。单 commit ≤500、PR ≤800。

## 1. BusinessSwitch 探活去 Actor

- [x] 1.1 先写测试：ClientPool 桩——连接成功→`probeDevice` 无错（在线）；`Get` 返回 err→`probeDevice` 报错（离线，R08）；断言不创建 Actor
- [x] 1.2 实现：`probeDevice` 用 `client.DeviceConnectionInfo{IP,Port(默认830),Username,Password}` → `ClientPool.Get` → `IsConnected()`；移除 `actor.*`/`StatusQueryCmd`/`time.Sleep`
- [x] 1.3 `go build ./... `+ `go vet` 绿；controllers 包测试绿

## 2. 删死代码 + 验证 Actor 使用清零

- [x] 2.1 删除 `internal/controller/vlan/actor_reconciler.go`（未接线、无引用）
- [x] 2.2 grep 确认 `pkg/yang-runtime/actor` 无非测试导入方（生产使用清零）
- [x] 2.3 全量 `go test ./...` + `go build ./...` 绿（actor 自身测试仍在，随后续批次删）

## 3. 收尾

- [x] 3.1 `system-architecture/tasks.md`：D2 更新为「Actor 生产使用清零、R01 实质满足；物理删包为 pr-size 清理债」
- [x] 3.2 满足 R01（Actor 不再生产在用）/R08（探活降级）/R06（TDD）
- [~] 3.3 **暂缓**：物理删 `pkg/yang-runtime/actor`（pr-size：model_actor 1089/device_actor 615 单文件超 800，需 leaf 顺序分批 ≥7 PR）；BusinessSwitch/Route 迁 Stack B + 退 cmd/controller（device-registry 收编）留后续
