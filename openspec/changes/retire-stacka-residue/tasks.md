# retire-stacka-residue — 任务

> 三 PR：①制品+文档对齐（插入型）→ ②纯删除（豁免档）→ ③sync+archive。②须待①合入后 rebase，保证②的 PR diff insertions ≤ 50。

## 1. PR① — 立项与文档对齐

- [x] 1.1 change 四件制品（proposal/design/specs delta/tasks）+ `openspec validate` 通过
- [x] 1.2 CLAUDE.md §5.6 B0 行标注「已物理删除」（backend/test/{e2e,integration} 载体清零，层定位不变）
- [x] 1.3 backend/README.md 摘除 NativeDeviceConfig 文档索引行及已删目录引用
- [x] 1.4 提交、push、PR ①，CI 全绿合入（merge-on-green 已授权）

## 2. PR② — 纯删除

- [x] 2.1 rebase 合入后的 main；确认删除前引用面仍为零（grep 复核）
- [x] 2.2 删 `backend/test/e2e/` 与 `backend/test/integration/` 整目录
- [x] 2.3 删 `backend/deploy/` 与 `backend/config/` 整目录
- [x] 2.4 删 NDC 闭包：`nativedeviceconfig_types.go`、`zz_generated.deepcopy.go`、`types_common.go`、`docs/crd/nativedeviceconfig.md`
- [x] 2.5 backend/Makefile 裁剪 Stack A target（design D4 边界），`make -C backend help` 仍可用
- [x] 2.6 验证：`go build ./...` + `go test ./... -race` 全绿；`grep -r "NativeDeviceConfig\|test/e2e\|test/integration" backend --include="*.go"` 零命中；每 commit ≤500 或纯删除 ≤6000
- [ ] 2.7 提交、push、PR ②（insertions ≤ 50 走豁免档），CI 全绿合入

## 3. PR③ — 收尾

- [ ] 3.1 sync：system-architecture SC-01 delta 合入主 spec；business-crd LEGACY 横幅补记 NDC 载体已删
- [ ] 3.2 archive change + 记忆更新（arch-optimization-roadmap D1 勾销、test-server 泄漏坑注记 integration 腿已删）
- [ ] 3.3 PR ③ 合入后清理 worktree（§6.3）
