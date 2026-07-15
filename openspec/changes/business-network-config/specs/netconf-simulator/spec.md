# netconf-simulator (delta)

## ADDED Requirements

### Requirement: NS-07 confirmed-commit 仿真

模拟器 SHALL 在 NS-02（running/candidate + commit/discard-changes）之上仿真 confirmed-commit：收到带 `<confirmed/>` 的 commit SHALL 将 candidate 提升为 running 并启动确认计时器（记录提交前 running 快照）；超时前收到确认 commit SHALL 取消计时器使配置转正；超时未确认 SHALL 将 running 回滚到快照。hello SHALL 宣告 :confirmed-commit capability（可经 ScenarioConfig 关闭以测试能力缺失路径）。

#### Scenario: 确认转正
- **WHEN** confirmed-commit 后超时内收到确认 commit
- **THEN** running SHALL 保持新配置，计时器取消

#### Scenario: 超时自动回滚
- **WHEN** confirmed-commit 后无确认直至超时
- **THEN** running SHALL 回滚到提交前快照，get-config 可断言

#### Scenario: 能力开关
- **WHEN** ScenarioConfig 关闭 confirmed-commit capability
- **THEN** hello SHALL 不宣告该 capability，供客户端能力缺失负路径测试
