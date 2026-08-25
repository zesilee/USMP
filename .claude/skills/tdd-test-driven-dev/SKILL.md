---
name: tdd-test-driven-dev
description: 所有改动测试先行（T01-T09 军规载体）：先产出测试设计再写实现，按 CLAUDE.md §5.6 选层补齐，覆盖正常、异常、边界、并发场景
---

# USMP TDD 军规执行技能

## 一、激活时机
1. 任何新功能、Bug 修复、重构，在编写业务代码**之前**激活（T01/T05）。
2. 用户需求包含「测试用例」「TDD」「单元测试」「异常测试」等关键词时激活。

## 二、权威来源（本技能只做执行指针，不复制正文）
- 测试军规 T01-T09 与测试分层职责：**CLAUDE.md §5.5-§5.6**（含「改动类型→必补层」映射表）
- 前端分层权威：**frontend/TESTING.md**（F1 happy-dom / F2 组件 / F3 真浏览器 / F4 E2E）
- 新增 YANG 模型接入设备配置：**必触发 `yang-config-test-design` 技能**（T02b，完备测试矩阵）
- 集成测试模板：`netconf-sim-integration-test` 技能
- 覆盖率棘轮：`backend/.coverage-baseline` 与前端 vitest thresholds 为下限，补测后同步上调（T08）

## 三、执行流程（红绿循环）
1. **测试设计先行（T05）**：列出本改动的用例清单（正常/异常/边界/并发/负路径），对照 §5.6 选层，缺层=未完成（T06）。
2. **红灯**：先写测试，运行确认其因正确原因失败。**Bug 修复必须先写复现该 Bug 的回归测试**（T07）。
3. **绿灯**：写最小实现使测试通过。
4. **重构**：保持全绿。
5. **验证**：后端 `go test ./... -race`；前端 `npm run test`（happy-dom）、涉及 Select 弹层/嵌套 list 交互时 `npm run test:browser`（F3）。

## 四、Go 测试写法要点（真实仓库口径）
- 表格驱动 + `t.Run` 子测试；并发用例必须在 `-race` 下通过（R09）。
- 并发测试用 `sync.WaitGroup` 收拢协程；**禁止用 `time.Sleep` 凑同步（Magic Sleep 反模式）、禁止在子协程里调用 `t.Fatalf`**（`t.Errorf` 可以，但失败信号建议经 channel 回主协程断言）。
- 集成测试命名 `*_integration_test.go`，开头 `if testing.Short() { t.Skip(...) }`（T03）。
- 现成范例：`backend/internal/cache/`、`backend/internal/api/` 下的 `*_test.go` 与 `*_integration_test.go`。

## 五、门禁
- 测试未过禁止 commit：pre-commit 本地拦截（后端变更包测试 + 前端 happy-dom 单测）+ CI 兜底（R15/T09）。
- 代码评审不通过禁止提交（T04，`go-code-review-check` 技能）。
