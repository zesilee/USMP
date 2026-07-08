<!-- tasks.md 模板 —— TDD 分阶段任务清单。见 CLAUDE.md §5.3。 -->

# <change-name> — tasks（<一句话>）

> 依赖：proposal/design/specs。TDD（R06 先测后码）。单 commit ≤500 行、PR ≤1000。
> §5.3 渐进迁移：双写 → 切读 → 删旧路径，禁一次性重写。每阶段 `go test ./... -race` 绿。

## 1. <阶段名>

- [ ] 1.1 先写测试：<正常/异常/并发>
- [ ] 1.2 实现：<...>
- [ ] 1.3 `go build ./...` + 变更包测试绿

## 2. <阶段名>

- [ ] 2.1 ...

## N. 收尾与验收

- [ ] N.1 全量 `go test ./... -race` + `go vet` 绿
- [ ] N.2 更新主 spec / 新增 delta spec（sync 阶段）
- [ ] N.3 满足相关红线（R01/R03/R06/R08/R09…）
