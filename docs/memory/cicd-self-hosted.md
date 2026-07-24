---
name: cicd-self-hosted
description: CI/CD 架构 —— GitHub 托管跑基础门禁 + Mac 自托管 Runner 跑部署/E2E/常驻 staging；含 E2E v2 债
metadata: 
  node_type: memory
  type: project
  originSessionId: bbca3dbd-b096-4965-8ef1-825f0e549811
---

2026-07-05 起，仓库补齐 CD（PR #53）。**仓库已从 PUBLIC 转为 PRIVATE**（自托管 Runner 前提：公开仓 + 自托管 = fork PR 本机 RCE）。

**架构**：轻量基础门禁留 GitHub 托管（compliance/commit-lint/pr-size/branch-name/sensitive-files/openspec/frontend-ci，PR 合并门禁）；重量级部署 + 全栈浏览器 E2E 上个人 Mac 自托管 Runner（标签 `macos-staging`）。`.github/workflows/e2e-staging.yml` 在 merge→main 触发：`docker compose build→up→健康→API冒烟→预热设备→Playwright staging-smoke→常驻`（前端 localhost:3002 / 后端 localhost:8080）。Runner 离线只让该检查 pending，不阻塞日常提交。

**2026-07-05 首次实战跑绿**（run 28735250327，success 9m3s，10 步全 ✓）。落地踩坑与结论（下次复用）：
- **Runner 领取任务失败**（`job was not acquired by Runner even after multiple attempts`，~3min 超时）：GitHub 显示 `online` 只代表监听连接在，Mac 休眠/服务模式异常时不接活。定位法：`./svc.sh stop` 后前台 `./run.sh`（前台能秒领任务即坐实是服务/休眠问题）。长驻需给 Mac「插电防休眠」。
- **Build images 极慢/挂**（一次 `nginx frontend: unexpected EOF`、一次挂 25min）：根因是**基础镜像 `nginx:alpine` 从 Docker Hub 拉取慢**（非 npm；frontend 用 `registry.npmmirror.com` 那步 OK）。解法：Mac 上手动 `docker pull nginx:alpine` 预热本地缓存。
- **不会重复下载镜像**：工作流 `docker compose build` 无 `--pull`/`--no-cache`，compose 未设 `pull_policy`（默认缺失才拉）→ 本地已有基础镜像即命中缓存跳过下载。改这点无需动配置。
- 首绿后 fix PR #54（setup-mac-runner.sh 非 UTF-8 locale 下 `${VAR}` 花括号修复，squash `9d1b468`）已合并。

**纯文件变更不触发重型 CI（PR #75/#76/#77）**：前端类工作流早有 `paths:` 过滤;补齐两处重型——① `e2e-staging` 加 `paths-ignore`（`**.md`/`.claude/**`/`docs/**`/`.gitignore`/`openspec/**`），docs 合并不触发 Mac 部署（非必需检查，零风险）。② `compliance`（必需检查，后端全量 `go test -race` ~4-5min）用 **GitHub「必需检查+同名哑绿」标准解法**：`compliance.yml` 加 `paths`（backend/** + compliance workflow 自身）只在后端代码跑;新增 `compliance-skip.yml`（`paths-ignore` 互补）在非后端变更时以**同名 job `Test + Lint + Coverage`** 秒级报绿。**坑**：必需检查直接加 paths 会永远 pending 卡合并,故必须有同名哑绿兜底;分支保护的必需检查名须与 job 名（`Test + Lint + Coverage`）一致。已用纯文档 PR #77 端到端验证（docs 变更 3-4s 哑绿、不触发 e2e、不空跑后端测试）。

**关键接线**（改架构前必读）：后端硬编码默认设备 `192.168.1.1:830 admin/admin` 不可经 env 配；compose 把 simulator 固定到该 IP（网关设 .254 避让）。`config_handler.GetConfig` 只传 `{IP}` 无协议/凭据，靠连接池缓存，需先 POST 设备预热。详见 `docs/CICD.md`。

**E2E v2 债（另立 OpenSpec change）**：现有 `navigation/vlan/interfaces/e2e-demo` 规格实测断言的是当前实现不产生的契约——`/config/:ip/vlans` 返回原始 XML（非 `data.data.vlans` 数组）、`/devices` 的 `data.data` 是对象非数组、「交换机设备管理平台」只在 `<title>`+`design-v0.html`（真实 Header.vue 不渲染）、vlan 表格需 CRD `businessvlans` 数据 + 模拟器默认空数据。复活它们=应用级改造（CRD 数据注入链 + 重写虚构断言 + 模拟器启动注入 running-config）。v1 门禁只用 `frontend/tests/staging-smoke.spec.ts`（SPA 挂载+外壳导航，实测绿）。关联 [[dual-stack-migration]] [[frontend-ci-gotchas]]。

**2026-07-05 债务收口一处 + 新增运维模式（PR #55, squash c6c0de0）**：
- **虚构契约债**不止在旧 E2E 规格，也潜伏在**运行时代码**：`stores/device.ts` 曾对接虚构的 `GET /api/devices` + `res.data.devices`（真实是 `/api/v1/devices` + `res.data.data.devices`），导致设备管理页恒空——同项目 `DeviceTree.vue`/`components/Dashboard.vue` 早已用对，唯此 store 漏网。修法：对齐真实契约 + `normalizeDevice`（`online:bool` 与旧 `status:'online'` 双兼容，缺失字段 ip 兜底）。**下次遇「页面空/数据不显示」先查该组件取数路径是否 `res.data.data.xxx`**。
- **staging-smoke 已加真设备断言**：`设备管理页应列出种子设备 192.168.1.1`（实测绿），是该 BUG 回归防线，也是把 device 数据从 E2E v2 债里抢救出的第一条真断言。
- **e2e-staging checkout 间歇性网络抖动**（与 [[backend-ci-flaky-tests]] 同属 flaky 一族）：`actions/checkout@v4` 的 `git fetch` 偶发 `Error in the HTTP2 framing layer` / `Failed to connect to github.com port 443`，3 次内建重试后仍挂 exit 128，build/deploy/E2E 全 skip。**这是 Mac→GitHub 网络抖动，非代码问题；`gh run rerun <id>` 重跑即过**（本次两次都是重跑一次转绿）。根治可考虑 checkout 换 SSH 或加 step 级重试。
- **分支级预验证法**：e2e-staging 支持 `workflow_dispatch`，可 `gh workflow run e2e-staging.yml --ref <branch>` 在**合并前**用自托管 Runner 跑真部署验证（会把该分支临时部署成常驻 staging，合并后 main push 再自动刷回）。
