# USMP Makefile — 标准开发命令
# 用法: make <target>

.PHONY: setup test lint compliance hook-install hook-verify help

# 默认目标
help: ## 显示所有可用目标
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ──────────────────────────────────────────────
# 一键安装
# ──────────────────────────────────────────────
setup: hook-install ## 一键安装: git hooks + 依赖 + 基线测试
	@echo "→ 安装 Go 依赖..."
	cd backend && go mod download
	@echo "→ 安装前端依赖..."
	cd frontend && npm install 2>/dev/null || echo "  (前端依赖跳过)"
	@echo "→ 运行基线测试..."
	cd backend && go test ./... -count=1 -timeout=120s
	@echo "✅ 安装完成！所有拦截层已激活。"

# ──────────────────────────────────────────────
# Git Hooks
# ──────────────────────────────────────────────
hook-install: ## 安装 Git Hooks (L3 拦截层)
	@git config core.hooksPath .githooks
	@chmod +x .githooks/pre-commit .githooks/commit-msg .githooks/pre-push 2>/dev/null || true
	@echo "✅ Git Hooks 已激活: .githooks/ → core.hooksPath"

hook-verify: ## 验证 Git Hooks 是否激活
	@HOOKS_PATH=$$(git config core.hooksPath 2>/dev/null || echo ""); \
	if [ "$$HOOKS_PATH" = ".githooks" ]; then \
		echo "✅ Git Hooks 已激活 (.githooks/)"; \
		echo "  pre-commit: $$([ -x .githooks/pre-commit ] && echo '✅ 可执行' || echo '❌ 不可执行')"; \
		echo "  commit-msg: $$([ -x .githooks/commit-msg ] && echo '✅ 可执行' || echo '❌ 不可执行')"; \
		echo "  pre-push:   $$([ -x .githooks/pre-push ] && echo '✅ 可执行' || echo '❌ 不可执行')"; \
	else \
		echo "❌ Git Hooks 未激活，运行: make hook-install"; \
	fi

# ──────────────────────────────────────────────
# 测试 & 检查
# ──────────────────────────────────────────────
test: ## 全量测试
	cd backend && go test ./... -race -count=1 -timeout=120s

lint: ## Go vet + Go fmt 检查
	cd backend && go vet ./...
	@echo "✅ go vet 通过"
	@CHANGED=$$(cd backend && git diff --name-only --diff-filter=ACMR HEAD 2>/dev/null | grep '\.go$$' || true); \
	if [ -n "$$CHANGED" ]; then \
		UNFORMATTED=$$(cd backend && gofmt -l $$CHANGED 2>/dev/null); \
		if [ -n "$$UNFORMATTED" ]; then \
			echo "❌ 未格式化文件:"; echo "$$UNFORMATTED"; exit 1; \
		fi; \
	fi
	@echo "✅ go fmt 通过"

compliance: lint test ## 完整合规检查 (lint + test)
	@echo "✅ 合规检查全部通过"
