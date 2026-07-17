# system-architecture — 系统级架构不变量与红线契约

## Purpose

system-architecture 约束整个平台的架构边界：无数据库、模型驱动、仅 NETCONF/gNMI、单一权威栈（yang-controller-runtime）。本契约以**可观测的系统级行为**表述架构红线（CLAUDE.md §2），保证平台运行在合规边界内；子系统的细化行为契约见各能力 `spec.md`，详细架构见同目录 `design.md`。

## Requirements

### Requirement: SC-01 单一权威栈与单一进程入口

系统 SHALL 以 Stack B（yang-controller-runtime）作为**唯一权威且唯一生产栈**（R01），生产进程 SHALL 只有一个入口 `backend/main.go`（真·单进程）。原 Stack A（K8s CRD + Actor）入口 `cmd/controller` 已删除，SHALL NOT 存在第二条生产运行路径。Stack A 代码载体（`pkg/yang-runtime/actor` 包、`api/biz/v1` 旧意图 CRD 类型、`internal/crdsource` 桥接、`pkg/translator` 翻译引擎）已全部物理删除，SHALL NOT 重新引入。

#### Scenario: 生产启动入口唯一
- **WHEN** 启动生产进程
- **THEN** SHALL 经由 `backend/main.go` 单一入口拉起 yang-controller-runtime（Manager→Controller→Reconciler→Source），不存在 Stack A 入口

#### Scenario: Stack A 载体物理不存在
- **WHEN** 审计代码库
- **THEN** `pkg/yang-runtime/actor`、`backend/api/biz/v1`、`internal/crdsource`、`pkg/translator` SHALL NOT 存在（历史契约由 actor-transaction/business-crd/translation-engine 的 LEGACY spec 承载）

### Requirement: SC-02 无数据库

系统存储运行配置的期望态与缓存时 SHALL 仅使用 TTL+LRU 内存缓存；持久元信息（业务意图实例、认领、收敛状态等）SHALL 存放于 K8s CRD（经 apiserver，由平台 etcd 承载），SHALL NOT 引入 MySQL/Redis/SQLite 或任何自管外部数据库（R03）。本地 JSON 文件 SHALL NOT 作为持久元信息载体（多实例部署禁止本地存储，见 SC-06）；CRD SHALL 仅作持久化载体与 watch 事件源，SHALL NOT 作为架构通道复活 Stack A 模式。

#### Scenario: 运行时无外部数据库依赖
- **WHEN** 系统运行并存储期望配置与元信息
- **THEN** 期望配置 SHALL 仅落 TTL+LRU 内存缓存，持久元信息 SHALL 仅经 K8s apiserver（CRD），不连接任何 MySQL/Redis/SQLite，也不直连自管 etcd

#### Scenario: 持久元信息不落本地文件
- **WHEN** 需要持久化业务意图等元信息
- **THEN** SHALL 写入 CRD，SHALL NOT 写实例本地文件系统

### Requirement: SC-03 仅现代协议

系统与设备通信时 SHALL 仅使用 NETCONF(SSH 830) 或 gNMI，SHALL NOT 使用 Telnet/SNMP 等旧协议（R02）。

#### Scenario: 下发/读取仅走 NETCONF/gNMI
- **WHEN** 系统下发或读取设备配置
- **THEN** SHALL 通过 NETCONF(SSH 830) 或 gNMI 通道完成，不建立任何 Telnet/SNMP 连接

### Requirement: SC-04 模型驱动渲染

新增 YANG 模块并在前端展示配置时，前端 SHALL 由模型 schema 自动渲染表单，SHALL NOT 手写固定表单（R05）；对应 YANG 结构 SHALL 由 ygot 自动生成，SHALL NOT 手写 YANG 结构体（R04）。

#### Scenario: 新增模块零手写表单
- **WHEN** 接入一个新 YANG 模块并在前端展示
- **THEN** 前端 SHALL 依模型 schema 自动渲染表单，后端结构 SHALL 由 ygot 生成，二者均无手写固定内容

### Requirement: SC-05 异常降级不崩溃

系统遭遇设备离线、缓存过期、下发失败或校验失败时 SHALL 走对应降级路径（重连/重拉/保留原配置/行内提示），进程 SHALL NOT 崩溃（R08）。

#### Scenario: 设备离线降级
- **WHEN** 目标设备离线
- **THEN** SHALL 触发 NETCONF 自动重连并对外返回明确错误状态，进程不崩溃

#### Scenario: 下发失败保留原配置
- **WHEN** NETCONF 下发失败
- **THEN** SHALL 保留原配置、缓存不更新并向前端提示错误，进程不崩溃

### Requirement: SC-06 多实例部署约束

USMP SHALL 以 K8s 内多实例（≥2 副本）形态部署为 PaaS 底座组件：任何持久状态 SHALL NOT 依赖实例本地存储（含操作审计——SHALL NOT 写实例本地文件）；设备连接元信息 SHALL 经 CRD 跨副本共享、跨重启存活（凭据经 Secret 引用，不明文进 CR）；全部有副作用的控制器（意图面与原生周期面）SHALL 具备 leader election 门控，多副本下对同一设备的周期对账 SHALL 仅由 leader 执行。无可达集群时上述各项 SHALL 自动降级为单实例内存行为（R08）。

#### Scenario: 实例无状态可替换
- **WHEN** 任一实例被重建
- **THEN** 业务意图、认领与收敛状态、设备注册表、操作审计 SHALL 从 CRD 完整恢复，不产生数据丢失

#### Scenario: 多副本无重复下发
- **WHEN** 两副本同时运行且原生面选主开启
- **THEN** 对同一设备的周期对账（NETCONF Get/Set）SHALL 仅由 leader 副本执行

#### Scenario: 无本地持久文件
- **WHEN** 集群模式运行任意时长后检查实例文件系统
- **THEN** SHALL 不存在承载持久元信息的本地文件（审计/设备/意图均在 CRD）
