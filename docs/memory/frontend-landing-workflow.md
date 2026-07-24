---
name: frontend-landing-workflow
description: 前端落地迭代（PR-B 起）的工作方式约定——独立 agent 检视 + 自助合并 + 风险台账
metadata: 
  node_type: memory
  type: feedback
  originSessionId: 0ec4b9ae-3cb9-4a36-9d3b-8524c8c49f29
---

用户 2026-07-05 为前端设计落地迭代（见 [[frontend-redesign]]）定的工作方式：

- **每个 PR 的设计文档、测试设计、开发都必须用独立 agent 检视**；检视无问题才继续开发/合并。
- **CI/CD 通过后由我自行 `gh pr merge` 合并**，不必等用户手动合。
- **迭代中的关键风险记录到记忆**（见 [[frontend-landing-risklog]]），用户次日据此规划后续迭代。
- **认真对待测试设计**（高质量交付的硬要求）：正常/异常/边界/并发/幂等/负路径都要有防线；后端触发 netconf 模拟网元集成测试。
- **这是一次很长的迭代**：上下文压缩时务必保留关键信息（进度、风险台账、设计决策、PR 顺序）。

**Why**：用户不实时盯屏，要用自动化 + 独立检视替代人工评审，保证跨会话/长迭代的质量与可追溯。
**How to apply**：每个 PR 收尾前 spawn 一个 review agent 检视 diff（对照 CLAUDE.md 红线 + 测试完备性）；改完再 push；CI 绿则自助 squash merge；风险随手记进 risklog。
