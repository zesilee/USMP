---
name: frontend-contract-gen
description: API 契约生成管线（swag→openapi→TS）+ 前端测试能力分层改造进度（1a done / 1b 1c 待做）
metadata: 
  node_type: memory
  type: project
  originSessionId: e83cdcca-14a0-4971-8bbb-0da5ae835af1
---

前端「测试能力提升」三层改造，起于「设备页恒空」复盘（虚构契约 bug，见 [[cicd-self-hosted]]）。根因两面：本地测试链路断 + 前端从无类型检查/契约漂移。

**已完成**：
- **第0层（PR #56, 13c4169）**：锁 Node 22（`.nvmrc`+engines+Dockerfile），修通本地测试链路。之前本机 Node18<vitest4 要求的 20.12，单测本地跑不了 → 全压 CI。本地 `nvm use && npm ci && npm test` = 93 测试 ~9s。
- **第1层a（PR #57, f5cc08e）**：API 契约生成管线 + device 域切生成类型 + CI 漂移门禁。

**契约生成管线**（`make gen-contract`）：`go tool swag init`(注解→Swagger2.0) → `swagger2openapi@7.0.8`(转3.x) → `openapi-typescript@7`(→`frontend/src/types/api.gen.ts`)。后端 swag 注解是唯一真源；`contract-drift.yml` 门禁按注解重生成并比对提交的 api.gen.ts，漂移即红。
- **坑1**：swag 只出 Swagger **2.0**，openapi-typescript v7 拒收 → 中间**必须**加 swagger2openapi 转 3.x。
- **坑2**：本仓 git（2.43）对 `git diff --exit-status` 报 `invalid option`（exit 129）→ 门禁改用 `git status --porcelain` 判空。
- **确定性已验证**：CI(Go1.26.2) 重生成与本地字节一致，漂移门禁可作真防线。
- swag 用 `Response{data=DTO}` 组合语法覆盖 `interface{}`，无需改 response.go；handler 由 `gin.H` 改返回类型化 DTO（JSON 字节不变，go test + e2e 双证）。

- **第1层c（PR #58, d5b393c）✅**：启用 typecheck 门禁 + 清零 34 处存量类型错。前端**原本无 tsconfig、从不做类型检查**（虚构契约存活的深层土壤）。加 tsconfig+vue-tsc+`typecheck` 脚本，frontend-ci 加 Typecheck 步骤（src/ 零类型错为合并门禁）。**契约强制力至此闭环**：写 `res.data.devices` → 编译错误、阻断合并。清错要点（复用）：
  - `ConfigPage.vue`：`ref<ReturnType<useConfigPage>>` 的 `UnwrapRef` 递归解包内部 ref，使 `.value` 访问类型+运行时双错位 → 改 **`shallowRef`** 一发修 26 处（亦修潜在运行时 bug）。
  - `DynamicForm.vue` 漏 `computed` import（真 bug）；DeviceTree/Dashboard 设备列表切生成类型 `DeviceStatusDTO`+空值守卫；`useK8sCRD` `creationTimestamp` 改可选、`update` 参数改 `Partial`。
  - tsconfig `exclude: [test, tests]`（测试文件不纳入门禁）。三层封死漂移：类型生成→漂移门禁→typecheck 门禁。

- **第1层b（PR #59, fe3b50f）✅**：config/yang 端点接入生成契约。yang 已类型化仅补注解；config `gin.H`→DTO（ConfigGetData/ConfigSetData/ReconcileInfo）。契约现覆盖全 REST API 面。
- **体积门禁修复（PR #60, 27f931f）✅**：pr-size.yml + `.githooks/commit-msg` 的行数统计加 pathspec 排除 `**/package-lock.json`、`**/*.gen.ts`、`**/go.sum`、`backend/docs/openapi/**`。**加重型 devDep 前必做**——否则 lockfile 增千行会误判超 500/800 限制。改后端记得 `make hook-install` 重装本地钩子（core.hooksPath=.githooks）。
- **第2层（PR #61, db94b70）✅**：Vitest Browser Mode 真 Chromium 组件测试。`vitest.browser.config.ts`（`@vitest/browser`+`@vitest/browser-playwright` 的 `playwright()` 工厂，非字符串 provider）+ `test:browser` 脚本 + `test/browser/`（默认 happy-dom 套件 exclude 之）+ CI `frontend-browser-tests`。**坑**：项目有 `@playwright/test`+`playwright` 两版 → CI 装 chromium 用 `node node_modules/playwright/cli.js install --with-deps chromium`（直调匹配版本的包 CLI，避免 npx 版本歧义）。
- **第4层（PR #62, 1e4f33c）✅**：Storybook（`@storybook/vue3-vite`）给 YANG 动态渲染组件（DynamicForm/DynamicTable/StatusBadge）建隔离展示。`.storybook/main.ts`（关遥测）+`preview.ts`（全局注册 Element Plus）+ CSF3 stories + CI `frontend-storybook` 构建门禁。`storybook-static/` gitignore。

- **第3层（PR #63, 3fbfc2b）✅**：本地全栈热循环 `make dev`（`scripts/dev.sh`）：go build 后端(:8080) + vite dev 前端(:3000, HMR)，免 docker、Ctrl-C 同停、:8080 端口预检 + 就绪等待 + trap 清理。设备 192.168.1.1 本地不可达属预期（硬编码不可配），在线端到端仍走 `make staging-up`。

**全部完成（8 个 PR，#55–#63）**。测试金字塔：契约类型生成(1a/1b·全API面)→漂移门禁+typecheck门禁(1a/1c)→本地全栈热循环 make dev(层3)→happy-dom 单测(既有93)→真浏览器组件测试 test:browser(层2)→Storybook 隔离(层4)→e2e-staging 真部署(既有)。配套：体积门禁排除生成物(#60)。全程本地 Node22 秒级验证。

**约束**（改这块必读）：本地钩子 ≤500 行/commit、PR ≤800 行 → 生成物(api.gen.ts ~220)+lockfile 易超，需拆提交/拆 PR。便携 Node22 在 `scratchpad/node22/`（本机无 nvm）。
