# device-protocol — 设备协议层架构设计（反向还原）

> **权威性**：✅ NETCONF 为权威（R02）；gNMI 存在但 Get/Set 为**空壳 stub**。
> **还原基准**：`main@b1cfbae`，代码根 `backend/pkg/yang-runtime/client/`。
> **上层导航**：`openspec/specs/system-architecture/design.md`；本能力即 C5。

## 1. 职责

统一设备接入层：为每台设备维持连接、屏蔽 NETCONF/gNMI 协议差异，向上暴露 `Get/Set/Subscribe/Close/IsConnected/DiscardCandidate` 接口，向 Reconciler / Actor 提供 actual 读取与配置下发能力。

## 2. 组件

### 2.1 `Client` 抽象
- `Client` 接口 `client/client.go:109`：Get / Set / Subscribe / Close / IsConnected / **DiscardCandidate**（2PC abort 钩子）。
- `DeviceConnectionInfo` `client.go:90`（IP/Port/Username/Password/Protocol）；`GetOption`/`SetOption` 函数式选项含 `WithCommit`/`WithDatastore`/`WithTimeout`（`client.go:36-88`）。
- `Change`/`ChangeType`(Add/Delete/Modify)、`GetResult`/`SetResult`/`Notification`：`client/changes.go`。

### 2.2 NETCONF 客户端 `client/netconf.go`（~1050 LoC）
- **传输**：scrapligo `netconf.Driver` over SSH，默认端口 **830**（`NETCONFDefaultPort` `netconf.go:23`）；`WithAuthNoStrictKey()` 跳过 host-key，`WithTimeoutSocket`。构造时**即时连接**（`netconf.go:47`）。
- **操作**：
  - `Get` → `driver.GetConfig(datastore, filter)`，默认 `"running"`，XPath 过滤 `constructFilter`（`netconf.go:124,318`）。
  - `Set` → 逐 change `driver.EditConfig(datastore, xml)`，默认 `"candidate"` + `Commit:true`；若 `setOpts.Commit && success` 再 `driver.Commit()`（`netconf.go:170,229`）；检查 `resp.Failed`/`<rpc-error>`。
  - `DiscardCandidate` → `driver.Discard()`（`netconf.go:306`），供 2PC abort。
  - `Subscribe` → **未实现**，返回 error（`netconf.go:258`）。
- **重连/超时**：默认 socket 超时 10s（`netconf.go:38`）；**无 retry loop**——Get/Set/Discard 各做一次 lazy reconnect（`!connected` 则 `connect()` 一次），无退避、无重试计数。
- 文件主体（`netconf.go:324+`）是 ygot 结构 / JSON map → 手工 YANG-kebab XML 的编解码，含 Go camelCase→YANG 名的字符串修正。

### 2.3 gNMI 客户端 `client/gnmi.go` — ⚠️ 空壳
- 用真实 `openconfig/gnmi` protobuf + grpc；默认端口 9339(明文)/9340(TLS)；无 TLS 时 `grpc.WithInsecure()`。
- **`Get` 发空 `GetRequest{}`**，path 参数从不编码（`gnmi.go:97`）→ 无法定位路径。
- **`Set` 发空 Path/Val**（注释「would be populated by caller」，`gnmi.go:154`）→ 结构性空更新。
- `Subscribe` 最完整：真实 `ON_CHANGE` 流式订阅 + goroutine 分发（`gnmi.go:186`）。
- `DiscardCandidate` 为文档化 no-op；构造失败仍返回 client。**未接入任何生产入口**。

### 2.4 协议选择 `pool.go:119` `DefaultClientFactory`
`switch info.Protocol`：`NETCONF`→NETCONF；`GNMI`→gNMI；`AUTO`→按端口（0/830→NETCONF，9339→gNMI，默认 NETCONF）。
调用方：device handler（`device_handler.go:103`）、Actor `buildConnInfo`（`model_actor.go:977`）、各 reconciler 均默认 `ProtocolAUTO`。**实践中 AUTO 恒解析为 NETCONF**（仅显式 9339 才 gNMI），故 gNMI 在生产装配中从不生效。

### 2.5 连接池 `client/pool.go`
- `ClientPool` 接口：Get / Release / CloseAll / Stats（`pool.go:11`）。`DefaultClientPool` = `map[ip]Client` + `sync.RWMutex` + `ClientFactory`。
- `Get` `pool.go:54`：若现有 client `IsConnected()` 则复用；否则双检锁经 factory 新建——这是池唯一的「重连」。
- **`Release` 为 no-op**（`pool.go:87`）：每设备一个 client 永久保留。
- `CloseAll` `pool.go:92`：关闭并删除全部；错误被吞（R08 瑕疵）。
- `Stats` `PoolStats{ActiveConnections, TotalConnections, Errors}`。

## 3. 数据流

```
Reconciler/Actor
   → ClientPool.Get(DeviceConnectionInfo)          # 复用或新建
   → Client.Get(path, WithDatastore("running"))    # NETCONF get-config → ygot
   → Client.Set(changes, WithCommit(true))         # edit-config(candidate) → commit
   → [2PC] Client.DiscardCandidate()               # abort 时回滚 candidate
```

## 4. 并发与异常

- 池：RWMutex 保护 map，`Get` 双检锁。Stats 仅在持锁段内一致。
- 无健康检查/重连线程——重连是 lazy，触发于下次 `Get/Set`。
- 异常：NETCONF `resp.Failed` → error 上抛由 Reconciler requeue；连接失败 lazy 重连一次。**缺口**：无退避重试、`CloseAll` 错误被吞。

## 5. as-built 缺口

| 缺口 | 位置 | 影响 |
|------|------|------|
| gNMI Get/Set 空壳 | `gnmi.go:97,154` | gNMI 实际不可用；R02 虽列 gNMI 但仅 NETCONF 落地 |
| NETCONF 无重试退避 | `netconf.go:86` | 抖动网络下单次 reconnect 后即失败，靠上层 requeue 兜底 |
| `Release` no-op | `pool.go:87` | 连接不主动释放，长期驻留 |
| `CloseAll` 吞错 | `pool.go` | 关闭异常不可见（R08） |

## 6. 关联
- `netconf-switch-protocol` 技能（SSH 830 对接规范）；`netconf-simulator/design.md`（对端模拟）；`yang-controller-runtime/design.md`（C5 上下文）。
