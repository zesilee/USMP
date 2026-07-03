# 仓库级开箱即用配置设计

> 日期: 2026-07-03
> 状态: Approved

## 目标

克隆后执行一次 `make setup`（或 `./scripts/bootstrap.sh`），自动激活全部开发流程配置：Git Hooks、OpenSpec、Superpowers、编辑器格式、依赖安装。无需额外手动配置。

## 核心问题

克隆后如果忘了 `make setup`，整个拦截体系不生效。

## 方案

手动一次 bootstrap——README 醒目提示 + `scripts/bootstrap.sh` 一键脚本 + `make setup` 调用。

## 交付文件

| 文件 | 作用 |
|------|------|
| `scripts/bootstrap.sh` | 一键激活：hooksPath + 依赖 + OpenSpec 配置 + 验证 |
| `.githooks/post-checkout` | 切换分支后自动验证环境（非首次 clone） |
| `.editorconfig` | 编辑器格式统一 |
| `openspec/.openspec.yaml` | OpenSpec 项目配置固化 |
| `Makefile` 更新 | `make setup` 调用 bootstrap |
| `CLAUDE.md` 更新 | §11 新增 TM10 |
| `README.md` 更新 | 醒目 bootstrap 提示 |

## Bootstrap 脚本逻辑

1. 设置 `git config core.hooksPath .githooks`
2. `chmod +x .githooks/*`
3. `cd backend && go mod download`
4. `cd frontend && npm install`
5. `cd backend && go test ./... -count=1 -timeout=120s`
6. 验证 `make hook-verify`
7. 输出就绪状态

## 自审

- Placeholder: 无 ✅
- Consistency: 与现有 Makefile setup 目标一致 ✅
- Scope: 聚焦于开箱即用配置 ✅
- Ambiguity: bootstrap 是唯一手动步骤 ✅
