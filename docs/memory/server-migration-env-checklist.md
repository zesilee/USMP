---
name: server-migration-env-checklist
description: 换机器/新环境跑 USMP 前必读——迁移会丢的四样东西（gh 二进制、node_modules 半旧、Playwright 双套浏览器、hooksPath 绝对路径）与逐项修法
metadata: 
  node_type: memory
  type: project
  originSessionId: 377703a8-eeab-458f-a0ae-4054ee1c4ecd
  modified: 2026-07-23T07:04:42.855Z
---

2026-07-23 从旧服务器迁移到新机器（Ubuntu 6.8 / 2 核 / 3.8G / root），全量体检结论：
代码、Go modcache（2.1G）、YANG 源（snd/ce6866p-yang 143 文件）、envtest 资产、docker 镜像都跟过来了；
**丢的是"装在系统里、不在 git 里"的东西**。逐项：

1. **`gh` 二进制丢失但 `~/.config/gh/hosts.yml` 还在** — 最隐蔽的致命项。
   `~/.gitconfig` 把 github 凭据 helper 指向 `!/usr/bin/gh auth git-credential`，
   gh 一没 → **push 直接失败**，PR 流程（R13/R14/TM01）整个断。
   修：`apt-get install -y gh`（Ubuntu 源里是 2.45.0），装完 token 自动复活，无需重新登录。
   现象容易误判成"网络问题"：`git ls-remote` 只读能过（公开仓库），只有 push 才炸。

2. **`node_modules` 是旧机器拷来的、比 package.json 旧** — `npm ls` 看着正常，
   但缺后加的包（本次缺 `vue-i18n`）→ **55 个前端测试文件 100% 报 "Failed to resolve import"**。
   修：`npm ci`（勿用 npm install，要对齐 lockfile）。

3. **Playwright 浏览器要装两套**（[[frontend-contract-gen]] 的"双版本歧义"具体化）：
   仓库里 `@playwright/test` 1.59.1（F4 staging-smoke 用）与顶层 `playwright` 1.61.1
   （`@vitest/browser-playwright` → F3 browser mode 用）各自锁不同 chromium build（1217 vs 1228）。
   `npx playwright install chromium` 只装到 `.bin/playwright` 指向的那一套（本次装到 1217），
   F3 依旧报 `Executable doesn't exist at .../chromium_headless_shell-1228`。
   修：另一套用 `node node_modules/playwright/cli.js install chromium` 显式装。

4. **`core.hooksPath` 是绝对路径** → 钩子照常触发，但 `make hook-verify` 只认字面量
   `.githooks`，会误报"❌ Git Hooks 未激活"。修：`make hook-install` 归一化成相对路径。
   （worktree 下的解析行为仍见 [[worktree-hooks-gotcha]]。）

**非迁移引起、但会持续看到的噪音**：staging 前端容器长期 `unhealthy` —— healthcheck 用
busybox wget 打 `http://localhost:80/healthz`，容器 /etc/hosts 里 `::1 localhost` 排在前面，
nginx 只听 IPv4 → connection refused。打 `127.0.0.1` 立即 healthy。纯误报：
`scripts/e2e-smoke.sh` 是从宿主 curl 映射端口探活，不看容器 health，故不阻断 e2e。

**迁移后全绿基线（本次实测）**：后端 `go test ./...` 全过、`make lint` 过、
前端单测 55 文件/394 用例过、F3 6 文件/10 用例过、F4 staging-smoke 19/19 过、
`openspec validate --all --strict` 32/32 过。冷 `go build ./...` ≈1m46s（2 核，慢但可用）。

**待用户确认**：仓库 GitHub 上现为 **PUBLIC**，与 [[cicd-self-hosted]] 记的"已转 PRIVATE"矛盾。
