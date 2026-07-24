---
name: gh-cli-monitor-gotcha
description: "监控 PR CI 前必读：本机 gh 老版本不支持 `gh pr checks --json`，监控脚本会静默死循环"
metadata: 
  node_type: memory
  type: project
  originSessionId: 1c015416-7746-4d30-8363-c35061c4e505
---

本机（leezesi/USMP 开发机）的 `gh` CLI 版本较老，**`gh pr checks --json` 报 `unknown flag: --json`**。

**Why:** 监控脚本里 `gh pr checks N --json ... || { sleep; continue; }` 会把这个永久性错误当瞬时错误无限重试——监控永不触发，看起来像"间隔设置有问题"（2026-07-18 #202/#204/#205 三次监控全因此失效，均由用户先发现 CI 结果）。

**How to apply:** 监控/等待 CI 用纯文本解析：`gh pr checks N 2>&1 | grep -qE 'pending|fail'` 之类；完成条件 = 输出非空且无 pending；失败检测 = grep fail。或先 `gh pr checks N --json x 2>&1 | grep -q "unknown flag"` 探测一次再选路径。永远先单跑一次命令验证可用，再放进循环。
