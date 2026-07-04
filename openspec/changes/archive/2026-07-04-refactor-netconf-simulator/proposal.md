# 重构 NETCONF 模拟网元 — proposal

> OpenSpec change：`refactor-netconf-simulator` | 阶段：propose
> 关联反向还原产物：`openspec/specs/netconf-simulator/{spec,design,tasks}.md`（迁移债 D10）

## What（做什么）

将 `backend/simulator/` 下两个职责重叠、保真度割裂的模拟器（`netconfsim` 真 SSH+2PC / `netsim` 裸 TCP 无事务）**统一重构为单一、分层、协议保真、测试与部署解耦的 NETCONF 模拟网元**，并将其内部表示从"不透明 XML 字节 blob"改为"结构化 ygot 配置树"。

范围（5 项）：
1. **合一**：统一到单一 `netconfsim`，删除 `netsim`。
2. **测试/生产解耦**：core 去掉 `testing`/`testify` 包级依赖，`Assert*` 助手移到独立测试子包；新增 `cmd/netconf-simulator` 独立可部署二进制，对齐 `backend/deploy/manifests/netconf-simulator`。
3. **结构化 datastore**：内部表示由 XML blob 改为 ygot `Device` 树；`edit-config` 走 ygot Unmarshal + 合并/删除语义，`get-config` 从树 Marshal 并支持 `<filter>`；消灭 1068 行 string-parsing、三套 `Extract*` 手术与平行 `*TestData` 结构。
4. **真 XML 解析**：RPC 分发与内容提取用 `encoding/xml` 解码，替换 `strings.Contains`/`strings.Index`。
5. **协议保真**：capability 正确广告（`base:1.0`/`1.1`、`:candidate`、`:writable-running`）、`get-config` filter、`edit-config` merge/replace/delete。

## Why（为什么）

反向还原审计（见 `netconf-simulator/design.md`）发现现状存在硬性不合理：

| 严重度 | 问题 | 证据 |
|--------|------|------|
| 🔴 | 两个模拟器职责重叠、保真度割裂；`test-server` 把裸 TCP、无 NETCONF 的 `netsim` 打印成"NETCONF Simulator"，名实不符 | `cmd/test-server/main.go:61,99`；`netsim/simulator.go`（grep ssh=0） |
| 🔴 | 生产代码包 `import "testing"`+testify，模拟器与测试框架焊死，**无法编译为独立二进制/容器**（却存在 deploy 清单） | `simulator.go:8-10`、`scenarios.go` |
| 🔴 | 三套并存数据模型（openconfig ygot / huawei ygot / netsim `VLANConfig`）+ 三个平行复制 ygot 的 `*TestData` 结构 | `datastore.go:544,716,1022` |
| 🟠 | "datastore" 是不透明 XML blob，靠 1068 行逐厂商逐 XML 形态 string-parse 才能"理解"配置 | `datastore.go` 全文；`426-428` 双形态兼容 |
| 🟠 | **edit-config 无 merge/delete 语义**：整体覆盖候选库 → 无法测增量下发/删除/幂等 | `server.go:320` |
| 🟠 | **get-config 忽略 filter** → 无法测子树/xpath 过滤 | `server.go:245-257` |
| 🟠 | RPC 分发靠 `strings.Contains`，能力声明（仅 `base:1.0`）与行为（实现 candidate/commit）不一致 | `server.go:161,88-91` |

核心价值：模拟网元是全部集成测试（T02）的对端，其保真度直接决定测试能覆盖多少真实 NETCONF 行为。当前设计**测不了增量配置、删除、过滤**——恰是配置管理平台最该验证的路径。重构后测试价值与可维护性同时提升，并让模拟器可作为独立容器部署。

## 影响范围

- **改动**：`backend/simulator/netconfsim/{server,datastore,simulator,scenarios}.go`；新增 `cmd/netconf-simulator/main.go`。
- **删除**（迁移末期）：`backend/simulator/netsim/`。
- **受影响消费方**：`internal/controller/{vlan,ifm,system,interfaces}/*_integration_test.go`、`pkg/yang-runtime/actor/*_integration_test.go`、`test/integration/e2e_test.go`（走 netconfsim）；`cmd/test-server/main.go`（走 netsim → 改接口）。
- **红线**：R04（ygot 树为唯一模型，消除手写 XML 解析）、R02（NETCONF 保真）、R06/T01（测试先行，先补对拍测试再切换）、§5.3（渐进迁移）。

## 非目标（本次不做）

- 不实现 gNMI 模拟（另属 `device-protocol` 空转件）。
- 不追求完整 RFC6241 一致性（仅覆盖平台实际使用的 RPC 与语义）。
- 不改动被测的 reconciler/actor 业务逻辑，仅改模拟器与其消费方接线。
