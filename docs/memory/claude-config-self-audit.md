---
name: claude-config-self-audit
description: /self-audit 配置自审计首轮执行台账：两钩子输入协议 bug 修复、superpowers 摘除、技能瘦身 21→14、一个待拍板的 W01 冲突
metadata:
  type: project
---

# .claude 配置自审计（2026-08-25，PR#414/#415）

碰 `.claude/` 配置、钩子、技能、CLAUDE.md §6/§7 前必读。

**已交付**：`/self-audit` 命令（`.claude/commands/self-audit.md`，仅手动触发）+ 首轮报告 `docs/reviews/self-audit-2026-08-25.md`（32 条目全批准执行）。PR#414 纯删除档（-2815：外来模板簇 plan/tdd-backend/tdd-frontend/planner/tdd-guide、孤儿钩子×4、yang-ygot-generate、5 个 openspec-* 副本、766 行 tdd-workflow）；PR#415 修正档（12 技能重写对齐现役栈、钩子修复、superpowers 摘除）。技能 21→14，agents 目录清零。

**钩子协议两坑（最有价值教训）**：
1. Claude Code 钩子载荷走 **stdin JSON**（tool_name/tool_input/tool_response），settings.json 里传 `"$TOOL_INPUT"` 参数或读环境变量都拿不到——两个钩子因此静默失效约两月（post-task-sync 日志 31 条全空值跳过，L2 拦截全放行）。
2. PreToolUse **exit 2 才阻断**，exit 1 只是非阻断报错。修复已实测（违规 exit 2 / 放行 exit 0 / TaskCreate 落盘）；两脚本尾部有固化自测命令，改钩子必跑。

**L2 拦截误伤模式**：钩子是对整条复合命令做行级正则，`git push` 之后同一行任何位置出现 `main` 字样（哪怕在 echo 文案里）都会拦——复合命令要拆开跑，或避免拦截关键词入行。

**superpowers 已摘除（方案 B）**：插件从未在本机安装，CLAUDE.md §7.3 改为「通用工作流实践」表（验证先于宣称完成、根因优先等内化保留）；worktree 走原生 EnterWorktree。别再引用 `superpowers:*` 技能名。

**待拍板冲突（下轮处理）**：L2 钩子修好后 W01「拦 `git checkout main`」变成真实生效，与 §6.3 完成分支选项 A（本地合并需切 main）打架；`git switch main` 不在拦截模式内，是当前合法逃生口。另：openspec CLI 再跑 init 可能重新生成已删的 openspec-* 技能副本，届时再删即可。

**配置清理拆 PR 模式**：本地 commit-msg 卡单 commit ≤500 行、CI 卡 PR 1000/3000/6000 三档（纯删除=新增≤50 行才享 6000）——删除与重写必须拆两个基于 main 的并行 PR，重写档凑 >20 文件走 3000 基建档。

相关：[[branch-sync-and-bulk-delete-gotchas]]、[[merge-authorization]]、[[gh-cli-monitor-gotcha]]（本机 gh 老版本连 `pr update-branch` 也没有，用 `gh api -X PUT .../update-branch`）。
