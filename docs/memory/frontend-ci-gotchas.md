---
name: frontend-ci-gotchas
description: 前端 CI 坑：vitest4 需 Node≥20.12（本机 Node18 跑不了）；仓库 Actions 设为 local_only 会让所有工作流 startup_failure
metadata: 
  node_type: memory
  type: project
  originSessionId: 885fb078-4483-407e-b202-c54d39217185
---

前端（`frontend/`）与 CI 的几个非显而易见的坑：

1. **vitest 4 / rolldown 需 Node ≥ 20.12（本机是 Node 18 / npm 9.2）** —— 本地 `npm run test` 报 `node:util does not provide an export named 'styleText'`，**不是测试失败**，是 Node 太旧。CI 固定 Node 22。要本地跑单测得先升级 Node（`npm ci` 本身在 Node18 可用，只是 vitest 跑不起来）。

2. **仓库 Actions 权限设为「仅本地 actions」(`allowed_actions: local_only`) 会让所有工作流秒挂** —— 症状：`gh run list` 里每个 workflow 都是 `startup_failure` / `0s`（连 checkout 都没跑）；PR 的 statusCheckRollup 空、mergeState=BLOCKED，看起来像「CI 卡住」。根因：local_only 连 GitHub 官方 `actions/checkout`、`setup-node`、`setup-go` 都拦。修复：`allowed_actions: selected` + `github_owned_allowed: true`（放开官方、仍拦第三方）。改仓库 Actions 设置是 admin 操作，Claude 的 auto 模式会拦——需用户自己跑 `gh api -X PUT /repos/OWNER/REPO/actions/permissions ...` 或在 UI 改，改后 close/reopen PR 重新触发。

3. **前端 lockfile 曾与 package.json 不同步（已修，PR #26）** —— 之前 committed `package-lock.json` 缺 `esbuild@0.28.1` 等致 `npm ci` 硬失败；已 `npm install` 同步并提交，现 `npm ci` 通过。若再漂移，`cd frontend && npm install` 同步 lockfile 再提交即可。

前端无 ESLint / 无 `lint` 脚本；`frontend-ci.yml` 的 Lint 步用 `npm run lint --if-present`（现 no-op，加了 lint 脚本自动生效）。Playwright E2E 不进 CI（省 Actions 分钟），本地 `npm run e2e`。

相关：[[dual-stack-migration]]、[[openspec-cli]]。
