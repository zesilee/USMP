# devices-api delta — 集群模式设备持久可观测行为

## ADDED Requirements

### Requirement: BR-13 设备注册跨重启存活（集群模式）

集群模式下经 `POST /api/v1/devices` 注册的设备 SHALL 在任一实例重建后仍出现在 `GET /api/v1/devices` 列表中（凭据完整可用于探活/回读）；无集群降级模式保持现状（重启即丢）。集群模式下 apiserver 不可达导致注册/删除持久化失败时 SHALL 返回 5xx 统一错误信封，SHALL NOT 假装成功。

#### Scenario: 注册后重启仍在列表
- **WHEN** 集群模式注册设备成功后实例重建
- **THEN** `GET /api/v1/devices` SHALL 仍含该设备，状态探活可用

#### Scenario: 持久化失败返回 5xx
- **WHEN** 集群模式注册设备时 apiserver 不可达
- **THEN** SHALL 返回 5xx 统一错误信封（设备不入库），SHALL NOT 崩溃（R08）
