---
name: backend-ci-flaky-tests
description: compliance CI 的 -race 时序测试偶发 flaky（TestDelayingQueueAddAfter 等）；同码另一 run 常通过，重触发即可
metadata: 
  node_type: memory
  type: project
  originSessionId: 885fb078-4483-407e-b202-c54d39217185
---

`compliance.yml`「Test + Lint + Coverage」跑 `go test ./... -race -timeout=120s`，其中若干**时序敏感测试**在 CI 负载下偶发失败（非代码问题）：

- `pkg/yang-runtime/queue` 的 `TestDelayingQueueAddAfter`（DelayingQueue AddAfter 时序断言）——已实测 flaky（PR #40）。
- netconfsim/actor 集成测试在 -race 下偶发（PR #36 一次纯文档 PR 也挂）。

**症状**：compliance 有两个 run（push + pull_request 事件），常一个 FAILURE 一个 SUCCESS/IN_PROGRESS，mergeState BLOCKED。日志里 `--- FAIL:` 是时序测试，非本次改动包。

**处置**：确认失败测试是时序 flaky 且非本 PR 触碰的包后，**空提交重触发**（`git commit --allow-empty` + push，新 SHA 起干净 run），勿花时间调试。`gh run rerun --failed` 偶报「workflow file may be broken」（push-event run 限制），故用空提交更稳。

**根治（未做）**：给这些时序测试加容差/去时钟依赖，或 CI 加 `-count` 重试。属测试健壮性债，非阻塞。

相关：[[arch-optimization-roadmap]]。
