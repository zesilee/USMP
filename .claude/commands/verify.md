# 验证（Verification）命令

对当前代码库状态进行全面验证。

## 项目结构适配

本项目前后端分离，验证时先定位 Git 仓库根目录，再进入对应目录：
- **后端**：`$(git rev-parse --show-toplevel)/backend` — Go 1.22
- **前端**：`$(git rev-parse --show-toplevel)/frontend` — React 19 + TypeScript（openinula 运行时 + EviewUI/antd 适配层）

---

## 指令（Instructions）

按以下顺序执行：

### 1. 后端构建检查
```bash
cd "$(git rev-parse --show-toplevel)/backend" && go build ./...
```
失败则报告错误并停止。

### 2. 前端构建检查
```bash
cd "$(git rev-parse --show-toplevel)/frontend" && npm run typecheck && npm run build
```
失败则报告错误并停止。

### 3. 后端测试套件
```bash
cd "$(git rev-parse --show-toplevel)/backend"
go test ./... -short   # 快速（跳过集成测试）
go test ./... -race    # 完整（含集成测试与竞态检测）
```
报告通过/失败数量。

### 4. 前端测试套件
```bash
cd "$(git rev-parse --show-toplevel)/frontend"
npm run test           # F1/F2 happy-dom 单测（含覆盖率棘轮阈值）
```
报告通过/失败数量与覆盖率；涉及 F3 场景另跑 `npm run test:browser`。

### 5. E2E 冒烟（pre-pr 档，或含 frontend/ 改动时）
```bash
make e2e-local    # 起 docker 全栈 → Playwright staging smoke（§6.2 门禁，pre-push 亦调用）
```

### 6. Git 状态
显示未提交变更与自上次提交以来修改的文件。

---

## 输出（Output）

```
VERIFICATION: [PASS/FAIL]

Backend Build:  [OK/FAIL]
Frontend Build: [OK/FAIL]  (typecheck + vite build)
Backend Tests:  [X passed, 0 failed] (-race)
Frontend Tests: [X/Y passed, Z% coverage]
E2E Smoke:      [OK/FAIL/SKIPPED]

Ready for PR: [YES/NO]
```

存在关键问题时列出并给修复建议。

## 参数（Arguments）

`$ARGUMENTS` 可以是：
- `quick` — 仅执行后端 + 前端构建
- `full` — 执行 1-4 与 6（默认）
- `pre-commit` — 构建 + 快速单测（对应 pre-commit 钩子口径）
- `pre-pr` — 全部检查 + `make e2e-local`
