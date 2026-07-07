# shared-device-store — tasks（共享设备连接信息注册表）

> 依赖：proposal/design。TDD（R06，先测后码）。单 commit ≤500 行、PR ≤800。§5.3 渐进迁移：双写→切读→删旧路径，禁止一次性重写。每阶段 `go build ./... && go test ./... -race` 绿。

## 1. DeviceStore（新能力，独立）

- [x] 1.1 先写测试：`Put/Get/Delete/List` 正常路径；未命中 `Get` 返回 `ok=false`；并发读写无 race（`-race`，R09）
- [x] 1.2 实现 `DeviceStore` 接口 + 默认内存实现（`sync.RWMutex`，值用 `client.DeviceConnectionInfo`），无 DB（R03）
- [x] 1.3 `Manager` 接口加 `GetDeviceStore()`；实现持有默认 store；`go build ./...` 绿

## 2. DeviceHandler 双写 + 读库（迁移阶段①，新旧并存）

- [x] 2.1 先写测试：种子设备与 `AddDevice` 后 `DeviceStore.Get("192.168.1.1")` 命中且字段完整（IP/Port/User/Pass/Protocol=AUTO）；`RemoveDevice` 从库删除
- [x] 2.2 实现：DeviceHandler 构造时把种子写入 store；`AddDevice`/`RemoveDevice` 同步写/删 store（旧私有 map 暂留，双写）
- [x] 2.3 `/api/devices`（ListDevices，含 #47 在线探活）+ GetStatus 改从 store 读（nil 兜底回退 map）；api 包全量 -race 绿

## 3. reconciler 从库解析（迁移阶段②，切读）

- [x] 3.1 先写测试：`deviceClient.resolveConn` 用纯 DeviceID + 已注册 store → 带正确凭据；未注册 → AUTO 兜底无凭据（R08，不崩）；无 store → 回退字符串解析（迁移兼容）
- [x] 3.2 实现：ifm `deviceClient.Get/Set` 注入并**优先查 `DeviceStore`**，未命中回退 `parseDeviceID`（§5.3 新旧并存，DRY 掉 Get/Set 重复解析）；`New` 加 resolver 参
- [x] 3.3 vlan / system 同构改造；`main.go`/crdsource 三处 `New` 传 `mgr.GetDeviceStore()`；集成测试暂以 `New(..., nil)` 保持绿（store 化改写并入阶段5删解析）
- [x] 3.4 全量 `go test ./... -race` 绿（flaky `TestDelayingQueueAddAfter` 重跑过）

## 4. config 回读 + 周期源从库（迁移阶段②续）

- [x] 4.1 先写测试：`fetchFromDevice` 对已注册设备建连带 Port/凭据/Protocol（捕获型 pool 断言 info 完整），fakePoolManager 加 GetDeviceStore
- [x] 4.2 实现：`config_handler.fetchFromDevice` 查 `DeviceStore`（命中用库 info，未注册回退 IP+AUTO）
- [x] 4.3 先写测试：周期源用 lister 每 tick 发事件；空 lister 不发（回归 #101 的 nil 空转）
- [x] 4.4 实现：新增 `DeviceLister` 接口 + `NewPeriodicSourceWithLister`（tick 时动态 `List()`，D-3b）；`main.go` 三个 controller 传 `mgr.GetDeviceStore()`

## 5. 删旧路径（迁移阶段③，切换完成）

- [ ] 5.1 删 `reconciler.go` 的 `parseDeviceID`（`user:pass@ip:port` 解析）+ 相关 helper（ifm/vlan/system）；集成测试改「先 `DeviceStore.Put(id,info)` 再纯 id 触发」（取代 `New(...,nil)` + `user:pass@ip:port`）
- [ ] 5.2 删 `netconf.go` 的 `admin/admin` 空凭据兜底（#100）及其测试；保留 Port/Timeout 兜底
- [ ] 5.3 删 DeviceHandler 旧私有 `devices` map（双写切单一来源）
- [ ] 5.4 全量 `go test ./... -race` + `go vet ./...` 绿

## 6. 收尾与验收

- [ ] 6.1 端到端：注册设备→提交接口配置→对账认证成功下发→回读可见→二次对账收敛；周期源持续复验
- [ ] 6.2 更新 `config-api` / `yang-controller-runtime` 主 spec + 新增 `device-store` spec（sync 阶段）
- [ ] 6.3 满足 R01/R03/R06/R08/R09；#100 兜底与 #101 controller 复验保留（复验仍是快速收敛路径，与周期源互补）
- [ ] 6.4 记债：设备信息持久化到本地 JSON（§8）留后续增量；真机 gNMI（D3/D5）不在范围
