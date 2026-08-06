## 1. 准备（worktree + 红灯）

- [x] 1.1 EnterWorktree 创建 `xmlcodec-multikey-decode` 隔离环境，跑基线测试确认全绿
- [x] 1.2 T05/T07 红灯先行：在 `xmlcodec/decode_test.go` 新增多键解码表格用例（复现 devm `physical-entity` 三键含 enum 的 `multi-key lists unsupported`）——确认红

## 2. B1 单元（表格驱动 + race）

- [x] 2.1 `entryKey` 多键分支实现（D1/D2：keyType 为 struct 时按 path tag 从 ΛListKeyMap 填充复合 key struct），红灯转绿
- [x] 2.2 补齐多键用例矩阵并实现缺键回退（D3）：嵌套多键列表（decodeField Map 分支）/ 部分键叶缺失宽容保留 / 键类型不可转换负路径（命名 list 的明确错误）/ 空回读边界
- [x] 2.3 单键零回归 + 并发：既有 golden/往返用例全绿；多键解码用例过 `-race`
- [x] 2.4 delete 通道契约守护：断言 `delete.go` 多键仍返回明确不支持错误（XC-03 范围注记落测试）

## 3. B2 集成（模拟网元端到端）

- [x] 3.1 集成测试：netconfsim 种子含多键列表回读形态（devm physical-entitys 真实结构），经 `decodeRunningConfig` 链路断言前端可渲染行数 > 0 且键字段真值正确（`testing.Short()` 跳过）

## 4. 收尾

- [x] 4.1 `go test ./...` 全绿 + 覆盖率不低于 `backend/.coverage-baseline`（T08，补测后按需上调）
- [x] 4.2 `go-code-review-check` 独立 agent 检视通过（T04）
- [x] 4.3 What/Why/How 三段式提交（测试与实现按红绿节奏原子提交）
- [x] 4.4 完成分支：push + PR，CI 全绿后自助 merge（合入授权），随 PR 提交记忆更新（MEM04 单独 commit）
