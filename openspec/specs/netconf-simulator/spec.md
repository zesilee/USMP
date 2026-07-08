# netconf-simulator — 可编程 NETCONF 模拟网元

## Purpose

netconf-simulator 为 YANG 业务提供可编程的模拟设备对端，支撑端到端集成测试（T02）。它是**单一、结构化、协议保真、测试/部署解耦**的 NETCONF 模拟网元：支持 SSH+NETCONF 会话、candidate/commit 数据存语义、故障注入与断言助手；同时以诚实命名的内存 REST 桩承担前端 E2E 后端（非 NETCONF），并可编译为无 `testing` 依赖的独立二进制部署。历史迁移债 D10（两个模拟器并存）已消除。反向还原自 `backend/simulator/netconfsim/`，忠实 as-built（经 `refactor-netconf-simulator` 重构）。

## Requirements

### Requirement: NS-01 NETCONF 会话

模拟器 SHALL 经 SSH（随机端口，密码认证 admin/admin，`PasswordCallback` 校验，SHALL NOT 使用 `NoClientAuth`）暴露 netconf subsystem。连接建立后 SHALL 发送 NETCONF hello，广告 `base:1.0` + `:candidate` + `:writable-running` 能力，随后进入 RPC 分发循环。

#### Scenario: 建立 NETCONF 会话
- **WHEN** 集成测试启动 `netconfsim.NewSimulator()`，客户端经 SSH（admin/admin）连接 netconf subsystem
- **THEN** 模拟器 SHALL 发送 hello 广告 `base:1.0`/`:candidate`/`:writable-running`，进入 RPC 分发

#### Scenario: 认证凭据错误
- **WHEN** 客户端以错误密码尝试 SSH 认证
- **THEN** `PasswordCallback` SHALL 拒绝认证，SHALL NOT 建立会话

### Requirement: NS-02 数据存操作

模拟器 SHALL 经 `classifyRPC` 对收到的 RPC 做结构化解码判定类型，并按类型操作 running/candidate 数据存：get-config SHALL 返回对应树序列化 XML（可套 subtree filter）；edit-config SHALL 按 operation 写入 candidate；commit SHALL 将 candidate 提升为 running；discard-changes SHALL 丢弃 candidate；其余 RPC SHALL 返回 `<ok/>`。

#### Scenario: edit-config 写 candidate 后 commit
- **WHEN** 客户端 edit-config 写入 candidate，随后 commit
- **THEN** 模拟器 SHALL 将 candidate 内容提升为 running，后续 get-config running SHALL 读到该配置

#### Scenario: discard-changes 丢弃 candidate
- **WHEN** candidate 存在未提交改动时收到 discard-changes
- **THEN** 模拟器 SHALL 丢弃 candidate 改动，running 保持不变

#### Scenario: get-config 套 subtree filter
- **WHEN** get-config 携带 subtree filter
- **THEN** 模拟器 SHALL 返回过滤后的树序列化 XML

### Requirement: NS-03 故障注入

模拟器 SHALL 支持通过 `ScenarioConfig{ErrorOnRPC, RejectAuth}` 注入故障：命中 `ErrorOnRPC` 的 RPC SHALL 返回错误，`RejectAuth` 置位时认证 SHALL 被拒绝，以覆盖设备离线、下发失败等降级路径（R08）。

#### Scenario: 指定 RPC 注入错误
- **WHEN** `ScenarioConfig.ErrorOnRPC` 匹配当前 RPC 类型
- **THEN** 模拟器 SHALL 返回错误响应，供测试验证下发失败降级

#### Scenario: 拒绝认证
- **WHEN** `ScenarioConfig.RejectAuth` 置位，客户端尝试认证
- **THEN** 模拟器 SHALL 拒绝认证，供测试验证设备离线降级

### Requirement: NS-04 断言助手（查通用树）

模拟器 SHALL 通过 `Simulator.Running` 暴露通用 XML 数据树（`HuaweiVLANs`/`HuaweiVLANsFull`/`HuaweiInterfaces`/`HuaweiSystem`/`OCInterfaces`），供 `testsupport.Assert*` 助手在结构化数据树上查询属性做断言，SHALL NOT 依赖 XML 字符串解析。

#### Scenario: 下发后结构化断言
- **WHEN** 下发配置后测试调用 `testsupport.Assert*` 读取 `Simulator.Running` 上对应树
- **THEN** 助手 SHALL 在通用数据树上结构化查询出属性供断言

### Requirement: NS-05 前端 E2E 后端（内存 REST 桩）

前端 Playwright 套件的后端 SHALL 由 `cmd/test-server` 承担：进程内 `vlanStore`（内存 VLAN 存储）经 Gin REST 直供 VLAN CRUD，SHALL NOT 经 NETCONF，命名 SHALL 诚实标识其为内存桩而非 NETCONF 模拟。

#### Scenario: 前端 E2E 经 REST 桩操作 VLAN
- **WHEN** 前端 Playwright 套件启动 `cmd/test-server` 并发起 VLAN CRUD 请求
- **THEN** 进程内 `vlanStore` SHALL 经 Gin REST 直接响应，不经 NETCONF

### Requirement: NS-06 独立可部署二进制

模拟器 SHALL 可编译为 `cmd/netconf-simulator` 独立可执行文件（flag `-addr`/`-port`），SHALL NOT 依赖 `testing` 包，SHALL 绑定指定端口并支持优雅退出，以支撑容器化部署（迁移债 D10 已消除、netconfsim 与 testing 解耦可独立部署）。

#### Scenario: 构建并运行独立二进制
- **WHEN** 构建 `cmd/netconf-simulator` 并以 `-addr`/`-port` 启动
- **THEN** 产出的可执行文件 SHALL 无 `testing` 依赖、绑定指定端口，收到退出信号时 SHALL 优雅退出
