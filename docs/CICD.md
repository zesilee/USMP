# USMP CI/CD 手册

端到端流水线：**GitHub 托管 Runner 跑基础质量门禁**，**个人 Mac 自托管 Runner 跑部署 + 全栈浏览器 E2E + 常驻本地 staging**。

---

## 1. 拓扑

```
push / PR ──▶ GitHub 托管 Runner（云端）
│              compliance · commit-lint · pr-size · branch-name ·
│              sensitive-files · openspec · frontend-ci
│              全绿 = PR 可合并（分支保护，PR 合并门禁）
│
merge → main ─▶ Mac 自托管 Runner（标签 macos-staging）
                .github/workflows/e2e-staging.yml：
                docker compose build → up -d → 健康等待
                → 后端 API 冒烟 → 预热默认设备
                → Playwright 部署冒烟（真浏览器，chromium）
                → 保持常驻（restart: unless-stopped，不 teardown）
                访问：前端 http://localhost:3002 · 后端 http://localhost:8080/api/v1
```

**职责切分**：轻量、每次提交都跑、要干净隔离的 → GitHub 托管（私有仓有免费额度）；重量级、要真实容器、要真浏览器、要常驻的 → 你的 Mac。

**Runner 离线**（关机/断网）不影响日常：只有 `e2e-staging` 会转 pending 等待，GitHub 那 7 个门禁照常跑、PR 照常可评审合并。

---

## 2. 编排（docker-compose.yml）

| 服务 | 镜像 | 网络 | 端口 |
|------|------|------|------|
| `simulator` | `usmp-simulator`（`backend/Dockerfile.simulator`） | `usmp-net` 固定 IP **192.168.1.1** | 830（仅内网） |
| `backend` | `usmp-controller`（`backend/Dockerfile`） | `usmp-net` | 宿主 **8080** |
| `frontend` | `usmp-frontend`（`frontend/Dockerfile`） | `usmp-net` | 宿主 **3002** |

**为什么 simulator 固定 192.168.1.1**：后端把默认设备硬编码为 `192.168.1.1:830 admin/admin`（不可经环境变量配置）。把模拟器固定到该 IP，后端用种子设备即可直连 —— 无需改后端代码。网关显式设为 `.254`，因为 Docker 默认会把子网首地址 `.1` 分给网关，会与模拟器抢地址。

> **子网冲突回退**：`usmp-net` 用 `192.168.1.0/24`。Docker Desktop for Mac 的 bridge 运行在其 LinuxKit VM 内，与宿主 LAN 隔离，即便家用路由器也是 `192.168.1.1` 也不冲突。若极个别环境 `docker compose up` 报子网冲突：改用别的 `/24`（如 `10.77.0.0/24`）并把 simulator 固定 IP 改成该网段内地址，同时把后端种子设备 IP 一并调整（涉及后端代码，属 v2）。

---

## 3. 首次在 Mac 上装 Runner

### 3.1 前置
- **Docker Desktop**：安装后在 *Settings → General* 勾选 **Start Docker Desktop when you sign in**（保证重启后 staging 自动恢复）。
- **Node ≥ 20**：`brew install node`。

### 3.2 取注册 token
仓库 → **Settings → Actions → Runners → New self-hosted runner → macOS**，复制页面 `./config.sh --token XXXX` 里的 **token**（有效期约 1 小时）。

### 3.3 一键安装
```bash
scripts/setup-mac-runner.sh <RUNNER_TOKEN>
```
脚本会：检查 docker/node → 下载最新 macOS Runner 到 `~/actions-runner-usmp`（仓库外，不污染 git）→ 以标签 `macos-staging` 注册 → 装成登录自启服务并启动。

完成后仓库 *Settings → Actions → Runners* 应看到该 Runner 为 **Idle**。

### 3.4 管理 Runner 服务
```bash
cd ~/actions-runner-usmp
./svc.sh status     # 查看
./svc.sh stop       # 停
./svc.sh start      # 起
./svc.sh uninstall  # 卸载服务
```

---

## 4. 日常使用

- **触发**：把 PR 合并到 `main` 即自动触发 `e2e-staging`；也可在 *Actions → E2E Staging → Run workflow* 手动触发（`workflow_dispatch`）。
- **访问 staging**：浏览器打开 `http://localhost:3002`（前端）、`http://localhost:8080/api/v1/devices`（后端）。
- **本地复现 CI**（不经 GitHub）：
  ```bash
  make e2e-local      # 构建+起+健康等待+浏览器冒烟
  make staging-ps     # 看容器状态
  make staging-logs   # 跟日志
  make staging-down   # 停
  ```

---

## 5. E2E 门禁范围（v1）与 v2 待办

**v1（当前，保证诚实为绿）** —— `frontend/tests/staging-smoke.spec.ts`：
- 真浏览器访问已部署前端 → SPA 挂载成功（`#app` 有内容）、无致命控制台错误；
- 应用外壳导航（设备管理/概览/系统设置）渲染；
- 配合后端 API 冒烟（`/api/v1/devices` 返回 `success:true`）与三容器健康检查。

**为什么不用现有 `navigation` / `vlan` / `interfaces` / `e2e-demo` 规格**：经实测，它们断言的是当前实现并不产生的接口契约与设计稿文案，例如：
- `e2e-demo` 断言 `GET /config/:ip/vlans` 返回 `data.data.vlans` 数组 —— 但后端该端点返回的是**原始 XML 字节**（base64）；
- `e2e-demo` 断言 `/devices` 的 `data.data` 是数组 —— 实际是 `{devices, stats}` 对象；
- `navigation` 断言页面可见文案「交换机设备管理平台」—— 该文案只在 `<title>` 和设计稿 `design-v0.html` 里，真实 `Header.vue` 不渲染它；
- `vlan` 断言设备树里出现 `192.168.1.1` 及 `default`/`Management` VLAN 表格 —— 真实前端是 **CRD schema 驱动**（`biz.usmp.io/v1/businessvlans`），且模拟器默认空数据。

**v2（另立 OpenSpec change）** —— 让数据丰富的 E2E 真正变绿，需应用级改造：CRD `businessvlans` 数据注入链、前端 schema 渲染核对、重写上述虚构断言、模拟器启动注入演示 running-config。工作量与不确定性较大，不塞进本次 CI/CD 交付。

---

## 6. 故障排查

| 现象 | 排查 |
|------|------|
| `e2e-staging` 一直 pending | Runner 离线：`cd ~/actions-runner-usmp && ./svc.sh status`；Docker Desktop 是否在跑 |
| `docker compose build` 失败 | 确认 Docker Desktop 运行；`go.mod` 的 go 版本与 `backend/Dockerfile` 的 `golang:` 标签需一致 |
| 健康等待超时 | `make staging-logs`；后端健康探针命中 `:8080/api/v1/yang/modules`，前端命中 `:3002/healthz` |
| `up` 报子网/网关冲突 | 见 §2 子网冲突回退 |
| Playwright 报缺库 | macOS 原生无需系统库；`npx playwright install chromium` 即可（Linux 才需 `--with-deps`） |
| staging 打开设备离线 | 预热是 best-effort；手动 `curl -X POST http://localhost:8080/api/v1/devices -H 'Content-Type: application/json' -d '{"ip":"192.168.1.1","port":830,"username":"admin","password":"admin"}'` |

---

## 7. 安全备注

- 仓库须保持 **PRIVATE**。自托管 Runner + 公开仓库 = fork PR 可在你 Mac 上执行任意代码（RCE）。转公开前必须先摘除自托管 Runner。
- Runner 安装目录在仓库外（`~/actions-runner-usmp`），不影响工作树、不进 git。
