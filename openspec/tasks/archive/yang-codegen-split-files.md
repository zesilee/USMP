---
id: yang-codegen-split-files
title: 生成物按文件拆分——消除 all.gen.go 单文件巨物，同包多文件、零语义变更
status: completed
priority: medium
branch: worktree-yang-codegen-split-files
worktree: .claude/worktrees/yang-codegen-split-files
change: yang-codegen-split-files（已归档 archive/2026-07-15-yang-codegen-split-files；delta 已 sync 主 spec）
blocked_by: (已解除 2026-07-15：huawei-network-instance-config 已合入 main，PR#148/#149/#150，已归档)
pr: https://github.com/zesilee/USMP/pull/162
updated: 2026-07-15
origin: 用户拍板 2026-07-14；由「ygot 技术栈自研可行性」讨论收敛而来——结论是不自研 ygot，改做同包生成物拆分治单文件膨胀
---

## 目标

`backend/internal/generated/huawei/all.gen.go`（59,414 行单文件）随全量 YANG 模型集成必然膨胀。**在不改语义、不改下游、不改运行期的前提下**，按文件拆小到 ≤~6.5k 行/文件。**不自研 ygot、不动 gzip blob 语义**——纯构建期物理重排。

## 当前状态（✅ 全部完成，2026-07-15，PR #162）

- ✅ explore/propose 制品齐全并 validate 通过（CG-01/CG-02 MODIFIED delta，R17 spec-first）。
- ✅ blocker 解除：huawei-network-instance-config 已合入 main（PR#148/#149/#150）并归档。
- ✅ **apply 阶段 0–6 全部完成**（worktree `worktree-yang-codegen-split-files`）：
  - 管线双模式 + genfix 多文件契约测试（commit 97745bb，181 行逻辑）；生成物拆分 12 文件独立 commit（最大 6,523 行）。
  - 等价性实证：SchemaTree+枚举全量对拍 1,791 行逐字节一致、Validate 819==819（off-by-one 结案：宽松 grep 噪声）、连续两次生成 md5 一致、`go test ./... -race` 35 包全绿、覆盖率 61.9%==基线。
  - **apply 新发现（spike 盲区）**：ygot `-output_dir` 每文件写同一份 import 块 → 未用 import 编译失败；解法 = 拆分模式以 `go tool goimports` 收尾（go.mod tool 指令锁版 v0.39.0，沿用 swag 先例）。
  - 评审通过（0 严重 0 中危）；本地 pre-commit regen-and-diff 实跑通过。
- ✅ 阶段 7–8 收官：PR #162 十项 CI check 全 pass；`/opsx:sync` delta 合入主 spec（validate 24 项全过）+ `/opsx:archive` 归档 `archive/2026-07-15-yang-codegen-split-files`（沿 AFPOL PR#160 先例随同一 PR 收尾）。
- 后续独立事项（本任务不含）：blob 外置 `//go:embed schema.gz`（省 13%，边际收益）推迟至独立 change 评估。

## 去风险实证结论（2026-07-14 spike，已验证，勿重跑）

| 项 | 结论 |
|----|------|
| 拆分可行 | `-output_dir -structs_split_files_count=8` → 12 文件，最大 6.5k 行 |
| 语义等价 | `type` 数 1082==1082；下游零改动（同包 `huawei`、同 import 路径） |
| 无膨胀 | 总行 59,733 vs 59,414（+0.5% 仅包头） |
| blob 隔离 | 自动落独立 `schema.go`（struct/blob diff 从此分离） |
| **确定性** | 独立两次生成 `diff -rq` 字节完全一致（CG-01/CG-03 前提） |
| genfix 复用 | 已支持 `<file>...` 多文件入参，无 blob 文件 no-op，几乎不用改 |
| 门禁复用 | pr-size（`:(exclude)backend/internal/generated/**`）+ CG-03 regen（`git diff --exit-code backend/internal/generated/`）均路径匹配，零改动 |

## 方案要点（详见 design.md）

- `gen.conf` 加可选 `split_count`；`gen-yang.sh` 双模式（设置→`-output_dir`+拆分；未设置→`-output_file` 单文件，向后兼容）。**huawei 设 8、openconfig 不设**。
- 后处理改「输出目录 glob」逐文件 genfix+gofmt；拆分前清理旧产物（`git rm all.gen.go` + 陈留 `structs-*.go` 幂等清理）。
- 不动 blob 语义、不改 `-include_schema`、不改运行期、不改门禁 YAML、不拆 openconfig。
- blob 外置 `//go:embed schema.gz`（省 13%）**推迟**到独立 change（边际收益）。

## 上下文恢复提示

- **纠正常见误解**：文件大主因是 struct+Validate（87%），不是 gzip blob（13%）。blob 是运行期承重墙（部署无 `.yang`，`drivers/huawei.go` + `schema/entry.go` 唯一 schema 源），删不得。
- 相关记忆：[[snd-driver-registry]]（gen-yang 管线/加模块流程）、[[test-governance-military-rules]]（§5.6 选层、覆盖率棘轮）。
- 相关 spec：`yang-codegen-pipeline`（CG-01/CG-02/CG-03，本 change 改前两者）。
- 历史先例：`archive/2026-07-09-snd-ygot-pipeline`（管线参数化 + genfix + regen-and-diff 门禁的由来；本 change 沿用其 explore-spike 去风险方法论）。
- 遗留待查（apply 期）：spike 中 `Validate` grep 计数 1636 vs 单文件 1637 off-by-one——疑 grep 命中差异（`type` 数精确一致佐证无语义差），apply 期以编译+round-trip+全量测试证实。

## 恢复指令

1. 新会话：`/task resume yang-codegen-split-files`。
2. **先检查 blocker**：`git log main` 确认 `huawei-network-instance-config` 已合入。**未合入则保持 blocked，不启动 apply。**
3. blocker 解除后：`EnterWorktree`（从最新 main）→ 跑基线 `go test ./...` → 按 `openspec/changes/yang-codegen-split-files/tasks.md` 阶段 1→8 执行（genfix 测试先行 T05 红灯 → 双模式管线 → huawei 重生成 → 等价/确定性验证 → 门禁复验 → review → 完成分支 → sync/archive）。
4. 每阶段完成照例回写本文件 + `/task sync`。
