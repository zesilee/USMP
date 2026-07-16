# operation-audit — 操作审计存储与查询

## Purpose

操作审计记录配置下发/删除操作供 `GET /logs` 查询（前端操作日志 + reconcile 状态 live-join）。集群模式以 AuditRecord CRD 持久（每条一 CR，多实例可见、跨重启保留、容量滚动清理），无集群降级进程内存；本地审计文件已退役（SC-06）。

## Requirements

### Requirement: OA-01 审计记录写入

系统 SHALL 在配置下发被接受（POST）与行删除成功（DELETE）时各写入一条审计记录（字段：DeviceIP/Path/Summary/Triggered/Actor/Timestamp）；被拒绝的请求（4xx）SHALL NOT 记录。写入 SHALL NOT 阻断主流程：持久化失败仅记日志，配置操作照常返回（R08，与现状一致）。

#### Scenario: 下发接受后留痕
- **WHEN** `POST /config` 下发被接受
- **THEN** SHALL 产生一条含设备 IP、YANG 路径与提交摘要的审计记录

#### Scenario: 写入失败不阻断下发
- **WHEN** 审计持久化后端不可用时发生配置下发
- **THEN** 下发流程 SHALL 正常完成，审计失败仅记日志，SHALL NOT 崩溃

### Requirement: OA-02 CRD 持久化与多实例可见

集群模式下审计记录 SHALL 以每条一个 `AuditRecord` CR 持久化（SC-02：CRD 仅当载体；label 含设备 IP 供筛选），SHALL NOT 写实例本地文件（SC-06）；任一副本写入的记录 SHALL 对所有副本的查询可见（最终一致），实例重建后审计历史 SHALL 完整保留。无可达集群时 SHALL 降级为进程内存存储（重启即丢，记日志说明）。

#### Scenario: 跨副本可见
- **WHEN** 副本 A 记录一条审计后向副本 B 查询 `GET /logs`
- **THEN** watch 收敛后 SHALL 含该记录

#### Scenario: 重启保留历史
- **WHEN** 集群模式写入若干审计记录后实例重建
- **THEN** `GET /logs` SHALL 仍返回这些记录

#### Scenario: 无集群降级内存
- **WHEN** 无可达 kubeconfig 时启动
- **THEN** 审计 SHALL 走进程内存实现，读写接口行为不变，SHALL NOT 崩溃（R08）

### Requirement: OA-03 容量上限与清理

审计记录 SHALL 有容量上限（1000 条，保最新）：集群模式下写入方在超限时 SHALL 按时间删除最旧 `AuditRecord` CR，清理 SHALL 幂等（并发多副本重复删除 SHALL 容忍 NotFound，SHALL NOT 报错中断）。

#### Scenario: 超限删最旧
- **WHEN** 已有 1000 条记录时写入第 1001 条
- **THEN** 最旧一条 SHALL 被清理，最新 1000 条保留

#### Scenario: 并发清理无害
- **WHEN** 两副本同时触发对同一最旧记录的清理
- **THEN** 一方删除成功、另一方收到 NotFound 并忽略，SHALL NOT 产生错误或崩溃

### Requirement: OA-04 查询契约（GET /logs）

`GET /logs` SHALL 支持按设备/状态筛选与分页，并与 reconcile 状态 live-join 出 Outcome/DiffCount（现契约不变，前端零改动）。查询 SHALL 读进程内镜像（watch 维护），SHALL NOT 逐请求访问 apiserver。

#### Scenario: 契约兼容
- **WHEN** 前端以现有参数请求 `GET /logs`
- **THEN** 响应结构、筛选与分页语义 SHALL 与迁移前一致

### Requirement: OA-05 本地审计文件退役

系统 SHALL NOT 再写审计本地文件：兼容期内设置 `USMP_AUDIT_FILE` SHALL 仅产生弃用警告并走内存降级路径（不写文件）；历史 `data/audit.json` 不做自动导入（审计为滚动窗口数据，可接受截断）。

#### Scenario: 弃用变量仅警告
- **WHEN** 设置 `USMP_AUDIT_FILE` 启动
- **THEN** SHALL 记弃用警告、不写任何审计文件，进程正常运行
