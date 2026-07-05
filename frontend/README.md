# USMP 前端

Vue3 + Element Plus + Pinia。YANG 模型驱动动态渲染（R05）。

## 环境要求

Node **≥ 20.12**（vitest 4 的底线），仓库锁定 **22**（见 `.nvmrc`，与 `frontend-ci` / Docker 构建一致）。

```bash
nvm use          # 读 .nvmrc → 切到 Node 22（或 fnm/volta 等价命令）
npm ci           # 严格按 lockfile 安装，可复现
```

> 用 Node 18 会踩坑：vitest 4 的 rolldown native binding 装不上，单测一行都跑不了。这是「设备页恒空」那次只能靠 CI 验证的根因——本地测试链路是断的。

## 本地测试循环（秒级，别再 push 上 CI 才验证）

| 命令 | 用途 | 速度 |
|------|------|------|
| `npm test` | 全量单测（vitest run，20 文件 93 测试） | ~9s |
| `npm run test:watch` | 改哪测哪，热重跑 | 即时 |
| `npm run test:ui` | vitest 图形界面 | — |
| `npm run test:coverage` | 覆盖率报告 | — |

E2E（Playwright，需先起后端+前端）：

```bash
npm run e2e          # 无头跑
npm run e2e:ui       # 图形调试 + trace
```

Playwright `baseURL=http://localhost:3002`（对齐已部署栈）。本地可先 `npm run dev`（:3000，`/api` 已代理到 :8080），或 `docker compose up` 起全栈。

## 本地全栈热循环（免 docker）

```bash
make dev     # 仓库根目录：go build 后端(:8080) + vite dev 前端(:3000, HMR)，Ctrl-C 同停
```

秒级迭代前端/API，无需 docker 重建。设备 `192.168.1.1` 本地不可达 → 展示「离线」（页面渲染、路由、动态表单、YANG 契约、绝大多数 API 照常）。需要「设备在线」的端到端（配置读写对账、E2E）用 docker 路径：`make staging-up` / `make e2e-local`（正确对齐 simulator→192.168.1.1:830）。

## 组件隔离开发 / 真浏览器测试

```bash
npm run storybook       # YANG 动态渲染组件（DynamicForm/DynamicTable/StatusBadge）隔离展示
npm run test:browser    # 真 Chromium 组件测试（Vitest Browser Mode）
```

## 分层策略

- **底层**：vitest 单测/组件测试 —— 本地秒级，覆盖逻辑与渲染。
- **顶层**：Playwright `tests/staging-smoke.spec.ts` —— 自托管 Mac Runner 上对真部署栈跑真浏览器，作合并门禁（见 `docs/CICD.md`）。

> 契约类 bug（前端调错端点/读错字段）单测 mock 容易「自我实现」而漏测，治本方案见后端 OpenAPI 契约生成（规划中）。
