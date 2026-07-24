---
name: test-governance-military-rules
description: 测试分层军规 + 覆盖率棘轮基线值 + 两个待办；改测试/加用例前必读
metadata: 
  node_type: memory
  type: project
  originSessionId: bbca4548-3ed5-478b-9e42-e33edd11430e
---

测试分层与职责已军规化并机械生效（PR #98，2026-07-06 合入 main `85f3e91`）。权威规范：**CLAUDE.md §5.6 + frontend/TESTING.md**。

**核心 = 「改动类型→必补测试层」映射**（此前只有 YANG 一条 T02b）：后端 B1单元/B2集成(netconfsim)/B3 API；前端 F1逻辑/F2组件/F3真浏览器/F4 staging-smoke。新增军规 T05(测试设计先行) T06(缺层禁止合并) T07(Bug必先回归) T08(覆盖率不下降) T09(本地门禁对称)。何时上 F3 真浏览器：el-select/teleport/嵌套 list 增删改（happy-dom 测不准的）。

**覆盖率棘轮（只准升不准降，加用例后要同步上调）**：
- 后端 `backend/.coverage-baseline` = **71.0**（compliance.yml 低于即 fail）。注意：CI -race 下覆盖有 ±0.1 抖动，**上调时留 0.1 余量**；且本地跑覆盖率**不含 envtest 集成测试**（无 KUBEBUILDER_ASSETS），绝对值与 CI 有差，别拿本地值直接上调。另：pr-size/commit-msg 有纯删除豁免（insertions≤50 上限 6000，2026-07-17 用户批准）。
- 前端 `vitest.config.ts` thresholds = **statements 84 / branches 78 / functions 77 / lines 84**（frontend-ci 跑 `npm run test:coverage`）。
- **⚠️ 前端本地覆盖率会被跑着的 staging 后端灌水**（2026-07-24 schema-harness 实测踩坑）：本机 :8080 有 staging 时，测试里 fetch localhost:8080 的**成功回调真的执行**，多覆盖数个函数——funcs 本地 78.18 vs CI 77.44（差 0.74）。据此把 funcs 提到 78 → CI 红 → 退回 77。**教训：上调前端阈值前先 `make staging-down`，或直接信 CI 数、留足 margin，别拿本地灌水值上调。**
- 补测后**记得把基线/阈值上调**到新水平，否则棘轮不收紧。

**本地门禁**（`make setup` 激活）：pre-commit 前端变更跑 `npm run test`（`USMP_SKIP_FE_TEST=1` 跳）；pre-push 全量 `go test -race` + 前端 e2e-smoke（`USMP_SKIP_E2E=1` 跳）。

**待办 follow-up**：
1. `test/browser/FieldRenderer.browser.test.ts` 只测 list `add`，欠 `remove/edit/group`（军规触发本 bug 排查）——军规生效后触碰该组件会被 T06+棘轮强制补。
2. `frontend/coverage/` 是历史误提交的生成物（67 文件、被跟踪），应取消跟踪+gitignore+pr-size 排除；本次因撞「每提交≤500 行」门禁未做，留独立小 PR。

注意：commit-msg 有**每提交 ≤500 行**限制（区别于 pr-size ≤1000）；改测试文件时勿把 coverage HTML 一起 `git add -A`。
