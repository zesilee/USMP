# system-architecture delta — SC-06 全局 HA 收敛

## MODIFIED Requirements

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
