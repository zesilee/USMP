---
name: worktree-hooks-gotcha
description: worktree 里改 .githooks 后钩子仍跑旧版？相对 core.hooksPath 在 linked worktree 解析到主仓库副本；改钩子的提交要用 git -c core.hooksPath=$PWD/.githooks
metadata: 
  node_type: memory
  type: project
  originSessionId: 276ae15a-4bc7-4bdc-9d70-60d9d460636e
---

在 linked worktree（`.claude/worktrees/*`）中修改 `.githooks/` 后，`git commit/push` 执行的仍是**主仓库工作树**的旧钩子——相对路径 `core.hooksPath=.githooks` 不解析到当前 worktree（2026-07-09 snd-ygot-pipeline 实测：worktree 已是新版 regen-and-diff 钩子，git 却跑了主仓库的旧冻结版）。

**Why:** 改门禁钩子的 PR（如 R04 regen-and-diff 化）在 worktree 里会被自己要退役的旧钩子拦死，看起来像门禁死锁，其实是执行了错误副本。

**How to apply:** 提交/推送时显式指绝对路径运行本 worktree 的新钩子：`git -c core.hooksPath="$PWD/.githooks" commit/push ...`（这是运行更新版门禁，不是 `--no-verify` 绕过）。合入 main 后主仓库钩子随 checkout 更新，问题自然消失。

相关：R04/体积门禁的 regen-and-diff 口径见 [[snd-driver-registry]]、openspec/specs/yang-codegen-pipeline/spec.md（CG-03）。另一坑：commit-msg 的 ≤500 行与 pr-size 各自维护排除清单（.githooks/commit-msg 与 .github/workflows/pr-size.yml），加新生成物类型要两处同步。
