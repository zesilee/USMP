---
name: merge-authorization
description: 合入授权：用户明确授权 CI 全绿的 PR 可直接自助 merge，无需逐次确认
metadata: 
  node_type: memory
  type: feedback
  originSessionId: 1c015416-7746-4d30-8363-c35061c4e505
---

用户拍板（2026-07-18，snd 融合期间）：「后续 CI 绿了都可以直接合」。

**Why:** 个人项目、无他人审批，PR 门禁（CI required checks + 自审清单）已是唯一质量闸；逐次询问合入徒增等待。

**How to apply:** PR 的全部 required checks 通过后直接 `gh pr merge --merge`，随后照常做 spec sync/归档/worktree 清理；不再用 AskUserQuestion 确认合入。若 PR 含破坏性契约变更或用户未拍板的范围扩张，仍先确认。与 [[frontend-landing-workflow]] 的「CI过后自助merge」一致，此条为显式授权记录。
