---
id: add-e2e-cicd-self-hosted
title: 端到端 CI/CD —— 自托管 Runner 部署流水线 + 常驻本地 staging
status: completed
priority: high
branch: worktree-add-e2e-cicd-self-hosted
worktree: .claude/worktrees/add-e2e-cicd-self-hosted
change: add-e2e-cicd-self-hosted
updated: 2026-07-05
merged_pr: 53
merge_commit: ca67bbc
---

## 目标

补齐项目「有 CI 无 CD」缺口：GitHub 托管跑基础质量门禁不变；个人 Mac 自托管 Runner（标签 `macos-staging`）在 merge→main 后 `docker compose build→up→健康→API冒烟→预热设备→Playwright 浏览器冒烟→常驻`，成为随时可访问的本地 staging（前端 localhost:3002 / 后端 localhost:8080/api/v1）。前提：仓库已转 PRIVATE。

## 进度（截至 2026-07-05）

- [x] **实现完成，PR #53 已开，7 个 GitHub 门禁全绿**（commit `251af2c`、`533f4d3`）
  - [x] 编排/镜像：`docker-compose.yml`（simulator@192.168.1.1 + backend:8080 + frontend:3002，restart:unless-stopped）、`backend/Dockerfile.simulator`；修 `backend/Dockerfile`（go1.21→1.26、健康探针改命中 `:8080/api/v1/yang/modules`）
  - [x] 流水线：`.github/workflows/e2e-staging.yml`（push→main + workflow_dispatch，self-hosted macOS）、`frontend/tests/staging-smoke.spec.ts`（真浏览器实测 2 passed）
  - [x] 运维：`scripts/setup-mac-runner.sh`、`docs/CICD.md`、`Makefile`（staging-up/down/ps/logs、e2e-local）
  - [x] 三镜像 docker build 全过；backend↔simulator SSH 链路实跑连通；过程修掉 3 硬 bug（/simulator 写成目录、compose 网关抢 .1、Dockerfile go 版本/探针失效）
- [x] **合并 PR #53**（squash → `ca67bbc`，2026-07-05T07:12:38Z）
- [ ] **用户本地基建（我无法代做，非阻塞）**：Mac 装 Docker Desktop(开机自启)+Node≥20 → 取 Runner token → 跑 `scripts/setup-mac-runner.sh <token>` → 确认 e2e-staging 绿 + localhost:3002 可访问
- [x] **合并后清理 worktree** + 归档任务（本次会话完成）
- [ ] **v2（另立 OpenSpec change）**：让数据丰富 E2E 真绿

## 上下文恢复提示

- **v1 门禁范围**：只用 `frontend/tests/staging-smoke.spec.ts`（SPA 挂载 + 外壳导航「设备管理/概览/系统设置」+ 无控制台错误）+ 后端 API 冒烟 + 三容器健康。理由：现有 `navigation/vlan/interfaces/e2e-demo` 规格实测断言的是**当前实现不产生的契约**——`/config/:ip/vlans` 返回原始 XML 字节（非 `data.data.vlans` 数组）、`/devices` 的 `data.data` 是对象非数组、「交换机设备管理平台」只在 `<title>`+`design-v0.html`（真实 Header.vue 不渲染）、vlan 表格需 CRD `businessvlans` 数据 + 模拟器默认空数据。
- **关键接线（改前必读）**：后端硬编码默认设备 `192.168.1.1:830 admin/admin` 不可经 env 配；compose 把 simulator 固定该 IP（网关设 .254 避让 docker 默认占 .1）。`config_handler.GetConfig` 只传 `{IP}` 无协议/凭据，靠连接池缓存，需先 POST 设备预热（工作流已含 best-effort 预热步）。
- **v2 复活清单**：CRD `businessvlans` 数据注入链 + 前端 schema 渲染核对 + 重写上述虚构断言 + 模拟器 `cmd/netconf-simulator` 启动注入 running-config（≥4 VLAN 含 default/Management，Huawei `HuaweiVlan_Vlan` XML 格式，见 `internal/controller/vlan/reconciler_integration_test.go`）。牵扯双栈半迁移复杂度，工作量大。
- 相关记忆：`[[cicd-self-hosted]]`、`[[dual-stack-migration]]`、`[[frontend-ci-gotchas]]`。

## 恢复指令

1. 新 session：`git pull origin main`（确认 PR #53 是否已合并）。
2. `/task resume add-e2e-cicd-self-hosted`。
3. 若 PR 已合并：清理 worktree + `/task archive add-e2e-cicd-self-hosted`，随后可 `/opsx:propose` 起草 v2（数据丰富 E2E）。
4. 若未合并：提醒用户合并 + 按 `docs/CICD.md` §3 在 Mac 上装 Runner。
