## MODIFIED Requirements

### Requirement: OA-01 审计记录写入

系统 SHALL 在配置下发被接受（POST）与行删除成功（DELETE）时各写入一条审计记录（字段：DeviceIP/Path/Summary/Triggered/Actor/Timestamp，及可选 Forced/ForcedOwners）；被拒绝的请求（4xx，含归属硬锁 409）SHALL NOT 记录。force 覆盖归属的下发 SHALL 记录 `Forced=true` 与被覆盖的认领意图列表（ForcedOwners），SHALL 经 `GET /logs` 可辨识。写入 SHALL NOT 阻断主流程：持久化失败仅记日志，配置操作照常返回（R08，与现状一致）。

#### Scenario: 下发接受后留痕
- **WHEN** `POST /config` 下发被接受
- **THEN** SHALL 产生一条含设备 IP、YANG 路径与提交摘要的审计记录

#### Scenario: force 覆盖留痕可辨识
- **WHEN** 携 `force=true` 覆盖意图 X 的认领路径下发成功
- **THEN** 审计记录 SHALL 含 `Forced=true` 且 ForcedOwners 含意图 X，`GET /logs` SHALL 透出该标记

#### Scenario: 归属拒绝不留痕
- **WHEN** 无 force 的下发被归属硬锁以 409 拒绝
- **THEN** SHALL NOT 产生审计记录

#### Scenario: 写入失败不阻断下发
- **WHEN** 审计持久化后端不可用时发生配置下发
- **THEN** 下发流程 SHALL 正常完成，审计失败仅记日志，SHALL NOT 崩溃
