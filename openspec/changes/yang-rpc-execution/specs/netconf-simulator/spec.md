## ADDED Requirements

### Requirement: NS-09 custom rpc 分发与校验

模拟网元 SHALL 识别非 get/get-config/edit-config 的模块 custom rpc，校验其 input（mandatory 存在、leafref 目标存在于当前 running 树），记录调用 `(rpc 名, input 值)` 供测试断言，并返回 `<ok/>` 或注入的结果/错误。未识别的 rpc SHALL 仍返回 ok（NS 既有降级不变）。

#### Scenario: 执行 rpc 被识别、校验、记录

- **WHEN** 收到 `reset-if-counters-by-name`（if-name = running 树中存在的接口）
- **THEN** 模拟网元 SHALL 校验通过、记录该调用、返回 `<ok/>`
- **AND** 测试 SHALL 能读取到该调用记录以断言端到端

#### Scenario: leafref 目标不存在 → rpc-error

- **WHEN** 收到 rpc，其 leafref input 指向 running 树中不存在的目标
- **THEN** 模拟网元 SHALL 返回 `<rpc-error>`（供负路径集成测试）

#### Scenario: 未识别 rpc 降级

- **WHEN** 收到无对应处理的 rpc
- **THEN** 模拟网元 SHALL 返回 ok（既有降级语义不变）
