## MODIFIED Requirements

### Requirement: FE-18 原生控制台归属徽标

原生模块控制台渲染被业务意图认领的对象/路径时 SHALL 显示「由业务配置 <意图名> 管理」徽标。用户对认领路径提交手改被后端归属硬锁拒绝（信封码 409 携 intents）时，SHALL 弹阻断确认框：列出认领意图名称并警示「意图收敛会覆盖手改」；用户确认后 SHALL 携 `force=true` 重发同一请求，取消则 SHALL 中止流程且不置错误态。force 放行后的响应含 `ownershipWarning` 时 SHALL 保留非阻断提示（一期行为）。

#### Scenario: 认领对象带徽标
- **WHEN** 原生 vlan 控制台列表中某 VLAN 被意图 X 认领
- **THEN** 该行 SHALL 显示归属徽标（含意图名）

#### Scenario: 硬锁 409 触发阻断确认
- **WHEN** 提交手改收到信封码 409 且 data.intents 含意图 X
- **THEN** SHALL 弹确认框列出意图 X 与覆盖警示，SHALL NOT 直接置为下发失败

#### Scenario: 确认后 force 重发
- **WHEN** 用户在阻断确认框点击「强制下发」
- **THEN** SHALL 以 `force=true` 重发原请求，成功后按 force 分支展示非阻断归属警告

#### Scenario: 取消则中止
- **WHEN** 用户在阻断确认框取消
- **THEN** SHALL 中止提交流程，不下发、不展示错误态
