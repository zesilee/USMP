---
name: backend-ci-flaky-tests
description: compliance -race flaky 现状：TestDelayingQueueAddAfter 已根治(#227)；剩 client 包 -race 偶发超时/netconfsim，重触发即可
metadata: 
  node_type: memory
  type: project
  originSessionId: 885fb078-4483-407e-b202-c54d39217185
---

`compliance.yml`「Test + Lint + Coverage」跑 `go test ./... -race -timeout=300s`，CI 负载下偶发失败（非代码问题）：

- ~~`pkg/yang-runtime/queue` 的 `TestDelayingQueueAddAfter`~~ **已根治（PR #227，2026-07-28）**：根因是真并发 bug——process() 空睡无唤醒信号导致短延迟项被拖延 + Len() 无锁读竞态。修法=注入 clock（FakeClock 确定性测试）+ newItem 唤醒信号 + Len 加锁。`-race -count=200` 零抖动。**别再对这个测试「重跑别管」——它已确定。**
- 仍偶发：`pkg/yang-runtime/client` 的 `-race` 整包**超时**（301s>300s，netconf 并发集成测试在负载下慢）；netconfsim/actor 集成测试偶发。这些是环境/负载 flaky，非代码。
- **更正（2026-07-30）**：client 包超时里有一类**不是**负载 flaky——SelfHeal 测试 Fatalf 后 cleanup 对半死 driver 二次 Close 踩 scrapligo 死锁，单测卡满 5 分钟拖爆整包（PR #235 首轮）。已根治：Close 有界化 + 回归测试钉死，见 [[scrapligo-concurrency-pitfalls]] 第 2 条。此后 client 包再超时才可按负载 flaky 重触发处理。

**症状**：compliance 有两个 run（push + pull_request 事件），常一个 FAIL 一个 SUCCESS，mergeState BLOCKED。日志里 `--- FAIL:`/`test timed out` 是环境 flaky，非本次改动包。

**处置**：确认失败是 client 超时/netconfsim 且非本 PR 触碰的包后，`gh run rerun <id> --failed` 重跑（本会话 #227/#228 实测重跑即过）；push-event run 限制时用空提交（`git commit --allow-empty` + push）。

**根治（部分）**：queue 时序测试已去时钟依赖（FakeClock，见上）。client 超时可提 `-timeout` 或拆重型集成测试到 B2，属测试健壮性债，非阻塞。

相关：[[arch-optimization-roadmap]]。
