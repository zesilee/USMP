# 验证（Verification）命令

对当前代码库状态进行全面验证。

## 项目结构适配

本项目已实现前后端分离，验证时必须先定位 Git 仓库根目录，再进入对应目录，避免当前工作目录已在子目录时路径错误：
- **后端**：`$(git rev-parse --show-toplevel)/backend` - Go 代码
- **前端**：`$(git rev-parse --show-toplevel)/frontend` - Vue3 + TypeScript

---

## 指令（Instructions）

请按以下确切顺序执行验证：

### 1. **后端构建检查（Backend Build Check）**
```bash
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT/backend"
go build ./...
```
- 如果构建失败，报告错误并停止（STOP）

### 2. **前端构建检查（Frontend Build Check）**
```bash
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT/frontend"
npm run build
```
- 如果构建失败，报告错误并停止（STOP）

### 3. **后端测试套件（Backend Test Suite）**
```bash
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT/backend"
go test ./... -short -v  # 快速测试（跳过集成测试）
go test ./... -v          # 完整测试（含集成测试）
```
- 报告通过/失败的数量

### 4. **前端测试套件（Frontend Test Suite）**
```bash
REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT/frontend"
npm run test
```
- 报告通过/失败的数量
- 报告覆盖率百分比

### 5. **Console.log 审计**
```bash
REPO_ROOT=$(git rev-parse --show-toplevel)
grep -RIn --exclude-dir=node_modules --exclude-dir=dist --exclude-dir=.git "console\.log" "$REPO_ROOT/backend" "$REPO_ROOT/frontend/src" || true
```
- 如果没有匹配结果，报告 `OK`
- 如果有匹配结果，报告其所在位置

### 6. **Git 状态（Git Status）**
- 显示未提交的更改
- 显示自上次提交以来修改的文件

---

## 输出（Output）

生成一份简洁的验证报告：

```
VERIFICATION: [PASS/FAIL]

Backend Build:  [OK/FAIL]
Frontend Build: [OK/FAIL]
Backend Tests:  [X passed, 0 failed]
Frontend Tests: [X/Y passed, Z% coverage]
Logs:           [OK/X console.logs]

Ready for PR: [YES/NO]
```

如果存在任何关键问题，请列出这些问题并给出修复建议。

## 参数（Arguments）

`$ARGUMENTS` 可以是：
- `quick` - 仅执行后端 + 前端构建
- `full` - 执行所有检查（默认）
- `pre-commit` - 执行构建 + 单元测试
- `pre-pr` - 执行完整检查 + E2E 测试
