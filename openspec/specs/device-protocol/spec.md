# device-protocol — 统一设备接入层（连接池 + NETCONF/gNMI 客户端）

## Purpose

device-protocol 是 Stack B 的设备接入层（C5）：连接池按设备 IP 复用连接，NETCONF/gNMI 协议客户端向上暴露 Get/Set/Subscribe/DiscardCandidate。连接信息（IP/端口/凭据/协议）由调用方从共享 DeviceStore 解析后传入，本层只负责按传入 info 建连——netconf 客户端**不再有 admin/admin 兜底**：未注册设备以空凭据建连、SSH 干净失败，而非静默掩盖缺失的注册。当前实现存在若干契约缺口：gNMI Get/Set 为空壳（发空请求）、NETCONF `Subscribe` 未实现、`Release` 为 no-op、`CloseAll` 仅返回最后一个错误（吞掉其余）；这些缺口在下方对应 Requirement 的负路径 Scenario 中显式记录。

## Requirements

### Requirement: DP-01 连接复用（双检锁）

`ClientPool.Get(info)` SHALL 按设备 IP 复用已建立且 `IsConnected()` 为真的 client；否则 SHALL 经 factory 新建并存入池。新建路径 SHALL 用双检锁（读锁快路径命中已连接 client，否则取写锁后再次检查）防止并发重复建连。`Release(ip)` 为 no-op（每设备常驻一个 client，不回收）。

#### Scenario: 复用已连接 client
- **WHEN** 某设备 IP 已有 client 且 `IsConnected()=true`，再次 `Get(info)`
- **THEN** SHALL 直接返回现有 client，不经 factory 新建

#### Scenario: 并发新建走双检锁
- **WHEN** 池中无该 IP 的可用 client，多个协程并发 `Get(info)`
- **THEN** 取写锁后 SHALL 再次检查已存在的已连接 client 并复用，仅一个协程经 factory 新建，`TotalConnections` 相应递增

#### Scenario: Release 为 no-op（负路径）
- **WHEN** 调用 `Release(ip)`
- **THEN** SHALL NOT 关闭或移除连接，client 保持常驻池中

### Requirement: DP-02 协议选择（AUTO 按端口）

client factory SHALL 依据 `DeviceConnectionInfo.Protocol` 选择协议：`NETCONF`→NETCONF、`GNMI`→gNMI、`AUTO`→按端口判定（端口 0 或 830→NETCONF、9339→gNMI、其余默认 NETCONF）。未知协议 SHALL 返回错误。**当前模拟器为纯 NETCONF，实践中 AUTO 恒落 NETCONF。**

#### Scenario: AUTO 按端口落 NETCONF
- **WHEN** `Protocol=AUTO` 且端口为 0 或 830（或其他非 9339 端口）
- **THEN** SHALL 建 NETCONF client（端口 0 补 830）

#### Scenario: AUTO 端口 9339 落 gNMI
- **WHEN** `Protocol=AUTO` 且端口为 9339
- **THEN** SHALL 建 gNMI client

#### Scenario: 未知协议报错（负路径）
- **WHEN** `Protocol` 非 NETCONF/GNMI/AUTO
- **THEN** factory SHALL 返回 "unsupported protocol" 错误，不建连

### Requirement: DP-03 NETCONF 读（get-config running）

`Client.Get(ctx, path, WithDatastore("running"))` SHALL 对 running 数据源执行 get-config，并以 path 构造 XPath filter 只取目标子树，返回原始 XML 结果供上层反序列化为 ygot 结构。空响应 SHALL 作为错误返回而非静默成功。

#### Scenario: 读取 running 子树
- **WHEN** 以 running 数据源 + 目标 path 调用 `Get`
- **THEN** SHALL 执行 get-config(running) + XPath filter，返回携带 XML 数据的 `GetResult`

#### Scenario: 空响应（负路径）
- **WHEN** 设备返回空结果
- **THEN** SHALL 返回 `error="empty response"`，`Data=nil`

### Requirement: DP-04 NETCONF 写（candidate→commit 两阶段）

`Client.Set(ctx, changes, WithCommit(true))` SHALL 逐个 change 执行 edit-config(candidate)，全部成功后 commit(running)；任一 change 或 commit 返回 `<rpc-error>`（`resp.Failed != nil`）SHALL 标记失败并向调用方返回错误。`DiscardCandidate(ctx)` SHALL 丢弃 candidate 配置以在 commit 前中止事务。

#### Scenario: 两阶段提交成功
- **WHEN** 提交一组合法 change 且 `WithCommit(true)`
- **THEN** SHALL 逐个 edit-config(candidate) 成功后 commit(running)，返回 `Success=true`

#### Scenario: edit-config 或 commit 报错（负路径）
- **WHEN** 某 change 的 edit-config 返回 `<rpc-error>`，或 commit 返回 `<rpc-error>`
- **THEN** SHALL 标记该结果失败、`Success=false`，并向调用方返回错误（配置不落 running）

#### Scenario: 回滚 candidate
- **WHEN** commit 前调用 `DiscardCandidate`
- **THEN** SHALL 丢弃 candidate；device 报错时 SHALL 上抛 discard 失败

### Requirement: DP-05 惰性重连

连接断开（`!IsConnected()`）后，下次 `Get`/`Set`/`DiscardCandidate` SHALL 触发单次 reconnect（无退避、无重试计数）。重连失败 SHALL 上抛错误，由上层（reconciler）requeue，本层 SHALL NOT panic（R08）。

#### Scenario: 断线后下次操作触发重连
- **WHEN** client 处于 `!IsConnected()`，随后调用 `Get`/`Set`/`DiscardCandidate`
- **THEN** SHALL 尝试单次 reconnect 后继续操作

#### Scenario: 重连失败上抛（负路径）
- **WHEN** 单次 reconnect 失败
- **THEN** SHALL 返回错误交由上层 requeue，不重试、不 panic

### Requirement: DP-06 契约缺口（已知未实现/降级）

本层部分能力当前为占位或降级实现，SHALL 记录为已知缺口，SHALL NOT 被上层当作已生效功能依赖：gNMI Get/Set 为空壳（发空请求）、NETCONF `Subscribe` 未实现、`CloseAll` 仅返回最后一个错误（吞掉其余）。

#### Scenario: NETCONF Subscribe 未实现（负路径）
- **WHEN** 调用 NETCONF `Subscribe`
- **THEN** SHALL 返回 "subscription not implemented for NETCONF" 错误，不 panic

#### Scenario: CloseAll 吞错（负路径）
- **WHEN** `CloseAll` 关闭多个连接、其中多个 `Close()` 报错
- **THEN** SHALL 关闭全部连接并清空池，仅返回最后一个错误（其余错误被吞、`Errors` 计数递增）
