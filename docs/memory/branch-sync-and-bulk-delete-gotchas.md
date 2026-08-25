---
name: branch-sync-and-bulk-delete-gotchas
description: 大清理/大删除 PR 与分支同步的三个门禁坑（2026-08-25 清理会话实录）
metadata:
  type: project
---

2026-08-25 仓库清理（PR#410-#413，原 #409 拆分）踩坑实录：

1. **PR 分支同步 main 禁用本地 `git merge origin/main`**：默认 merge commit 消息
   （"Merge remote-tracking branch..."）过不了 commit-msg 钩子（要求 type 前缀 +
   What/Why/How），会留下半截 merge 状态连锁污染后续 checkout。正确姿势：
   `gh api -X PUT repos/<owner>/<repo>/pulls/<n>/update-branch`（服务端更新，
   等价页面 Update branch 按钮）。分支保护要求 head 与 main 同步，多 PR 只能
   串行 update→等CI→merge。
2. **大删除也要拆 PR**：pr-size 纯删除档（insertions≤50）上限也只有 6000 行；
   超了就按文件集不相交拆成多个**直接基于 main** 的并行 PR（cherry-pick 分发
   commit），绝不堆叠（[[list-server-pagination]] 连坐教训）。commit 级门禁另
   有单 commit ≤500（纯删除 ≤6000），改写大文件可拆「纯删除批+纯新增批」两个
   commit 过闸。
3. **大 pull 之后本地必修**：`cd frontend && npm ci`（换栈后 node_modules 半旧
   →单测全灭）+ `make gen-schema-ir`（schema.ir.gz 构建期产物不入库，缺了后端
   编译即挂）。与 [[server-migration-env-checklist]] 同族。

**Why**: 这三个坑每个都造成过一轮返工（#409 整个作废重拆、合入循环假成功——
`gh pr merge` 失败被 `| tail` 吞掉退出码仍打印 "merged"）。
**How to apply**: 大删除清理开工前先按第 2 条规划 PR 切分；写自动化循环时
禁止 `cmd | tail` 后直接当成功，显式查退出码。
