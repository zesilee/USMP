# business-intent-orchestration (delta)

## ADDED Requirements

### Requirement: BIO-01 CRD watch 事件源（C4）

意图 CR 的变更 SHALL 经 CRD watch 事件源进入 Stack B 控制器队列（C4 Source），复用 Manager→Controller→Reconciler→Source 分层（R01）；CRD SHALL 仅作持久化载体与事件源，SHALL NOT 复活 Stack A 式「CRD 当架构」（双重建模/Actor/前端消费 CRD）。无 K8s 集群环境（本地开发/CI 单测）SHALL 优雅降级：意图功能不可用但进程正常运行（R08）。

#### Scenario: CR 变更触发对账
- **WHEN** 意图 CR 被创建/更新（generation 变化）
- **THEN** SHALL 产生 reconcile 事件进入意图控制器队列，由意图 Reconciler 处理

#### Scenario: 无集群优雅降级
- **WHEN** 进程启动时无可用 kubeconfig/apiserver
- **THEN** SHALL 记录日志并跳过意图源注册，原生配置链路不受影响

### Requirement: BIO-02 意图展开（确定性编排）

意图 Reconciler SHALL 将意图实例确定性展开为 N×(设备 IP, 原生模块, 配置片段)：同一意图输入 SHALL 产生相同展开结果（可测试）；展开结果 SHALL 记录为归属认领 (设备, 模块, 路径) 集合。展开 SHALL 只依赖意图 spec 与设备清单，SHALL NOT 依赖本地文件或实例内存中的历史状态。

#### Scenario: 展开纯函数性
- **WHEN** 对同一意图 spec 重复执行展开
- **THEN** SHALL 得到逐字节一致的 (设备, 模块, 片段) 集合与认领集合

### Requirement: BIO-03 跨设备两阶段提交下发

意图的首次/变更下发 SHALL 走跨设备 2PC：阶段一对全部目标设备 edit-config 写 candidate（prepare），任一设备 prepare 失败 SHALL 对全体已 prepare 设备 discard-changes 并中止；阶段二 SHALL 使用 confirmed-commit（带超时）提交，全部设备确认后发 confirming commit；confirming 阶段部分失败 SHALL 依赖超时自动回滚兜底并在 status 呈现。事务成功后 SHALL 将展开片段写入 desired ConfigStore 并触发原生对账；事务失败 SHALL NOT 写 desired（防止周期对账绕过事务下发）。

#### Scenario: 全体成功
- **WHEN** 意图展开覆盖 2 台设备且两台 prepare、confirmed-commit、confirming 均成功
- **THEN** 两台设备 running SHALL 含展开配置，desired ConfigStore SHALL 写入片段，status.deviceStates 全部 synced

#### Scenario: prepare 部分失败即全体中止
- **WHEN** 设备 A prepare 成功、设备 B edit-config 返回 rpc-error
- **THEN** SHALL 对 A 执行 discard-changes，两台设备 running 均不变，desired 不写入，deviceStates 标 failed+原因

#### Scenario: confirmed-commit 超时回滚兜底
- **WHEN** confirming commit 阶段某设备失联
- **THEN** 该设备 SHALL 依 confirmed-commit 超时自动回滚，status SHALL 呈现该设备 failed 与事务不一致告警

### Requirement: BIO-04 稳态声明式收敛

2PC 成功后的漂移修复 SHALL 走既有每设备声明式对账（周期 Source + diff + 下发），不再要求跨设备事务性；意图 Reconciler SHALL 在每次对账时重写 desired ConfigStore（对冲 desired TTL 过期）。

#### Scenario: 事务后漂移被单设备修复
- **WHEN** 某设备上被意图管理的 VLAN 被手工删除
- **THEN** 下一周期对账 SHALL 仅对该设备重新下发，不触发跨设备事务

### Requirement: BIO-05 删除生命周期（finalizer→DELETE 命令通道）

意图 CR SHALL 带控制器 finalizer：删除请求（deletionTimestamp 置位）SHALL 触发展开为 DELETE 命令通道调用（声明式通道不承载删除），逐设备清理意图认领的配置并使 desired/缓存失效；全部设备清理成功后 SHALL 摘除 finalizer 放行 CR 删除；部分失败 SHALL 保留 finalizer、status 呈现失败设备并重试。

#### Scenario: 删除意图清理全部设备
- **WHEN** 删除覆盖 2 台设备的意图 CR
- **THEN** 两台设备上认领的配置 SHALL 被 DELETE 命令清除，之后 CR SHALL 消失

#### Scenario: 部分设备清理失败
- **WHEN** 设备 B 离线导致清理失败
- **THEN** CR SHALL 因 finalizer 保留，deviceStates 标 B failed，恢复后重试直至清理完成

### Requirement: BIO-06 收缩差集（孤儿配置清理）

意图 spec 变更导致展开范围收缩（如 devices 列表移除一台设备、端口列表移除端口）时，Reconciler SHALL 用 status 中持久化的上一代展开摘要与本代展开求差集，对被移除目标下发 DELETE 清理；差集计算 SHALL NOT 依赖实例内存（多实例就绪）。

#### Scenario: 移除设备触发孤儿清理
- **WHEN** 意图 devices 从 [A,B] 改为 [A]
- **THEN** 设备 B 上该意图认领的配置 SHALL 被清理，B 从 deviceStates 与 claims 移除

### Requirement: BIO-07 软归属认领

展开成功后认领集合 SHALL 写入 CR status 并经 USMP API 暴露；原生配置写路径命中认领路径时 SHALL 返回警告（不拦截，硬锁留二期）。认领 SHALL 随意图删除/收缩同步清除。

#### Scenario: 认领可查询
- **WHEN** 查询某设备某模块的归属信息
- **THEN** SHALL 返回认领该路径的意图 CR 名称列表

### Requirement: BIO-08 多实例就绪

意图层 SHALL 无本地持久状态（意图在 CR spec、状态在 CR status、展开可重算）；意图控制器 SHALL 预留 leader election 接缝（单实例部署时可关闭）。全局 HA（device store 共享化、全控制器 leader election、audit 迁出本地文件）不在本变更范围。

#### Scenario: 换实例接管
- **WHEN** 处理某意图的实例被杀、另一实例接管
- **THEN** 新实例 SHALL 仅凭 CR spec/status 恢复展开与收敛，行为与原实例一致
