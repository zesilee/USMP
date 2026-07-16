# system-architecture (delta)

## MODIFIED Requirements

### Requirement: SC-02 无数据库

系统存储运行配置的期望态与缓存时 SHALL 仅使用 TTL+LRU 内存缓存；持久元信息（业务意图实例、认领、收敛状态等）SHALL 存放于 K8s CRD（经 apiserver，由平台 etcd 承载），SHALL NOT 引入 MySQL/Redis/SQLite 或任何自管外部数据库（R03）。本地 JSON 文件 SHALL NOT 作为持久元信息载体（多实例部署禁止本地存储，见 SC-06）；CRD SHALL 仅作持久化载体与 watch 事件源，SHALL NOT 作为架构通道复活 Stack A 模式。

#### Scenario: 运行时无外部数据库依赖
- **WHEN** 系统运行并存储期望配置与元信息
- **THEN** 期望配置 SHALL 仅落 TTL+LRU 内存缓存，持久元信息 SHALL 仅经 K8s apiserver（CRD），不连接任何 MySQL/Redis/SQLite，也不直连自管 etcd

#### Scenario: 持久元信息不落本地文件
- **WHEN** 需要持久化业务意图等元信息
- **THEN** SHALL 写入 CRD，SHALL NOT 写实例本地文件系统

## ADDED Requirements

### Requirement: SC-06 多实例部署约束

USMP SHALL 以 K8s 内多实例（≥2 副本）形态部署为 PaaS 底座组件：任何持久状态 SHALL NOT 依赖实例本地存储；新增有副作用的控制器 SHALL 具备 leader election 接缝（本变更交付意图层就绪，存量控制器全局 HA 另立任务跟踪）。

#### Scenario: 实例无状态可替换
- **WHEN** 任一实例被重建
- **THEN** 业务意图、认领与收敛状态 SHALL 从 CRD 完整恢复，不产生数据丢失
