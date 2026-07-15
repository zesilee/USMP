# device-protocol (delta)

## ADDED Requirements

### Requirement: DP-08 confirmed-commit 原语（跨设备 2PC 支撑）

NETCONF 客户端 SHALL 在 DP-04（candidate→commit）之上提供 confirmed-commit 原语：`CommitConfirmed(ctx, timeout)` SHALL 发送带 `<confirmed/>`+`<confirm-timeout>` 的 commit；`ConfirmCommit(ctx)` SHALL 发送确认 commit 使配置转正；超时未确认时设备侧自动回滚 SHALL 被视为预期行为并可被上层感知（Get 回读验证）。设备不支持 :confirmed-commit capability 时 SHALL 返回明确错误供上层降级为普通 commit（呈现为非事务下发）。

#### Scenario: confirmed-commit 后确认转正
- **WHEN** `CommitConfirmed(30s)` 成功后在超时内调用 `ConfirmCommit`
- **THEN** 配置 SHALL 保持在 running，不发生回滚

#### Scenario: 超时未确认自动回滚
- **WHEN** `CommitConfirmed(1s)` 成功后不发确认
- **THEN** 超时后设备 running SHALL 回滚到提交前状态，回读可验证

#### Scenario: 能力缺失明确报错
- **WHEN** 设备 hello 未宣告 :confirmed-commit capability
- **THEN** `CommitConfirmed` SHALL 返回能力缺失错误，不发送 RPC
