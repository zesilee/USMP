# Tasks — yang-codegen-split-files

> **启动前置（TM03）**：`huawei-network-instance-config` change 必须先合入 main。本 change apply 以其扩充后的 `internal/generated/huawei/all.gen.go` 为拆分基线。**在前置未满足前，本文件仅为已批准的设计与计划，禁止进入 apply/worktree。**

## 0. 前置门禁（apply 启动检查）

- [x] 确认 `huawei-network-instance-config` 已合入 main（PR#148/#149/#150，delta 已 sync、change 已归档 2026-07-13）
- [x] 从最新 main 创建 worktree（`worktree-yang-codegen-split-files`），基线 `go test ./...` 全绿（2026-07-15）
- [x] 确认无其他 worktree 正在改 `internal/generated/huawei/` 或 `scripts/gen-yang.sh`（`git worktree list` 仅主仓库）

## 1. genfix 多文件测试先行（T05 红灯）

- [x] B1：`genfix` 对无 `ySchema` 文件输入 → 内容不变（no-op）——`TestFixFileNoSchemaNoop`
- [x] B1：`genfix` 对 enum 文件 → 仅 `PortType_50|100GE`→`_OR_`，YANG 原值映射串 `"50|100GE"` 不变——`TestFixFileSetIdempotent` 逐文件断言
- [x] B1：`genfix` 对含 `var ySchema` 文件 → blob 确定性重排、语义等价——`TestFixSchemaBlobStandaloneFile`（独立 schema.go、无 struct 上下文，design D4 显式锁定）
- [x] B1：文件集幂等——同组文件跑两次，第二次零变更——`TestFixFileSetIdempotent`
- [x] 红灯确认 → **实际为「锁定回归」绿灯基线**：现有 genfix `for os.Args[1:]` 已多文件就绪，三条新用例首跑即绿，作为拆分布局契约锁定（2026-07-15）

## 2. 管线双模式（gen.conf + gen-yang.sh）

- [x] `gen.conf` 解析新增可选 `split_count`（沿用现有 `key=value` 解析，正整数校验 fail-fast）
- [x] `gen-yang.sh` 分支：`split_count` 设置 → `-output_dir` + `-structs_split_files_count`；未设置 → 保持 `-output_file`
- [x] 后处理改 glob：genfix 多文件入参一次调用 + **`go tool goimports -w`**（apply 期新发现：ygot `-output_dir` 给每文件写同一份 import 块，未用 import 致编译失败，spike 未覆盖编译；goimports 以 go.mod `tool` 指令锁版 v0.39.0，沿用 swag 先例，兼做 gofmt 格式化）
- [x] 拆分分支生成前清理旧产物（`all.gen.go` + 陈留 `structs-*.go`/`enum*.go`/`union.go`/`schema.go`），幂等，不动 doc.go/gen.conf
- [x] `make gen-yang VENDOR=huawei`（拆分路径）与 `make gen-yang VENDOR=openconfig`（单文件路径，重生成零 diff）均验证

## 3. huawei 包重生成（拆分落地）

- [x] `gen.conf` 设 `split_count=8`（实测最大文件 6,523 行 ≤6.5k 目标，无需微调）
- [x] `make gen-yang VENDOR=huawei` 重生成 → 删 all.gen.go(59,414 行) + 新增 structs-0..7/enum/enum_map/union/schema 12 文件（总 59,745 行，+0.6% 仅包头）
- [x] openconfig 不设 `split_count`，确认其 `all.gen.go` 零 diff

## 4. 等价性与确定性验证（绿灯）

- [x] `go build ./...` + `go vet` 通过（下游零改动实证）
- [x] 连续两次 `make gen-yang VENDOR=huawei` → 全部 12 文件 md5 一致（CG-01 确定性；CI 同构环境由 CG-03 regen-and-diff 复验）
- [x] `huawei.Schema()` round-trip 成功；SchemaTree 键集(1,082 entry)+ΛEnum 全量 dump 拆分前后 1,791 行逐字节一致（一次性对拍程序，不入库）
- [x] 全量 `go test ./...` 35 包全绿——证 struct/Validate 语义未变
- [x] Validate off-by-one 结案：严格正则 `func \(t \*\w+\) Validate\(` 拆分前后 **819 == 819 精确一致**；spike 的 1636/1637 为宽松 grep（连带命中调用点/注释）计数噪声，非语义差

## 5. 门禁复验（零改动确认）

- [x] 本地 pre-commit（regen-and-diff 对称）通过——生成物 commit 触发全量 `make gen-yang` + diff 空校验实跑通过
- [ ] CI CG-03（compliance regen-and-diff）通过——路径前缀 `backend/internal/generated/` 匹配已静态审计覆盖新文件名，待 PR 实跑
- [ ] CI pr-size 通过——`:(exclude)backend/internal/generated/**` 已静态审计生效，待 PR 实跑
- [x] 门禁路径审计：pre-commit/commit-msg/compliance/pr-size 全部目录前缀匹配，新文件名零改动覆盖

## 6. 代码评审 + 提交（§6.2 / T04）

- [x] `go-code-review-check` 通过（0 严重 0 中危，2 低危备注不阻止）
- [x] 覆盖率不下降（T08）：总覆盖率 61.9% == 基线 61.9（genfix 包 80.4%），基线无需上调
- [x] 提交 What/Why/How 三段式；分 commit：97745bb 管线双模式+genfix 测试（181 行逻辑）、生成物重排独立 commit
- [x] ≤500 行/commit：逻辑 commit 181 行；生成物 churn +59,745/-59,414 不计逻辑行（PR 描述标注）

## 7. 完成分支（§6.3）

- [ ] `go test ./...` 全绿
- [ ] `superpowers:finishing-a-development-branch` → push + PR（选项 B）
- [ ] PR ≤3000 行逻辑（生成物排除后）

## 8. sync / archive

- [ ] CI 全绿 + 自审清单（TM05）通过 → 合入 main
- [ ] `/opsx:sync`：delta（CG-01/CG-02 MODIFIED）合入主 spec `yang-codegen-pipeline`
- [ ] `/opsx:archive`：change 归档
