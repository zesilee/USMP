# retire-idle-scaffolds — 任务

## 1. PR① — 立项

- [x] 1.1 四件制品 + validate；提交 push PR，CI 绿合入

## 2. PR② — 实现与删除（TDD）

- [x] 2.1 【测试先行】factory 负路径红灯：Protocol=GNMI → 显式未实现错误；AUTO+9339 → 同；AUTO 其余端口行为不变
- [x] 2.2 删 client/gnmi.go + source/gnmi_sub.go；factory 分支改显式错误——绿灯；go mod tidy
- [x] 2.3 删 pkg/yang-runtime/plugin 整包 + Manager 接口方法/字段；受影响单测清理
- [x] 2.4 CLAUDE.md §1 gNMI 标注规划、§4 C1/C4 改述；Settings.vue 端口提示标注规划（F2 断言同步）
- [x] 2.5 验证：go test ./... -race 全绿、覆盖率不降（死代码删除应回升，棘轮酌情上调）、前端单测/typecheck；含 frontend 改动跑 e2e smoke
- [x] 2.6 sync（YR-07/DP-02/DP-06 入主 spec + purpose 缺口注记清理）+ archive + 记忆更新（roadmap D3/D5 勾销清零）+ worktree 清理
