# USMP Makefile — 标准开发命令
# 用法: make <target>

.PHONY: setup bootstrap test lint compliance hook-install hook-verify help \
	staging-up staging-down staging-logs staging-ps e2e-local

# 默认目标
help: ## 显示所有可用目标
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# ──────────────────────────────────────────────
# 一键安装（克隆后必执行）
# ──────────────────────────────────────────────
setup: bootstrap ## 一键安装: hooks + 依赖 + 基线测试 + 验证（克隆后必执行）
	@echo ""

bootstrap: ## 激活全流程配置（hooks + 依赖 + 测试 + 验证）
	@./scripts/bootstrap.sh

# ──────────────────────────────────────────────
# Git Hooks
# ──────────────────────────────────────────────
hook-install: ## 安装 Git Hooks (L3 拦截层)
	@git config core.hooksPath .githooks
	@chmod +x .githooks/pre-commit .githooks/commit-msg .githooks/pre-push .githooks/post-checkout 2>/dev/null || true
	@echo "✅ Git Hooks 已激活: .githooks/ → core.hooksPath"

hook-verify: ## 验证 Git Hooks 是否激活
	@HOOKS_PATH=$$(git config core.hooksPath 2>/dev/null || echo ""); \
	if [ "$$HOOKS_PATH" = ".githooks" ]; then \
		echo "✅ Git Hooks 已激活 (.githooks/)"; \
		echo "  pre-commit:     $$([ -x .githooks/pre-commit ] && echo '✅ 可执行' || echo '❌ 不可执行')"; \
		echo "  commit-msg:     $$([ -x .githooks/commit-msg ] && echo '✅ 可执行' || echo '❌ 不可执行')"; \
		echo "  pre-push:       $$([ -x .githooks/pre-push ] && echo '✅ 可执行' || echo '❌ 不可执行')"; \
		echo "  post-checkout:  $$([ -x .githooks/post-checkout ] && echo '✅ 可执行' || echo '❌ 不可执行')"; \
	else \
		echo "❌ Git Hooks 未激活，运行: make setup"; \
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

# ──────────────────────────────────────────────
# 本地 Staging（docker-compose）—— 复现 e2e-staging 工作流
# 详见 docs/CICD.md。需要 Docker（Mac 用 Docker Desktop）。
# ──────────────────────────────────────────────
staging-up: ## 构建并起本地 staging（simulator+backend+frontend，常驻）
	docker compose up -d --build --remove-orphans
	@echo "✅ staging 已启动 → 前端 http://localhost:3002  后端 http://localhost:8080/api/v1"

staging-down: ## 停止并移除本地 staging
	docker compose down

staging-ps: ## 查看 staging 容器状态
	docker compose ps

staging-logs: ## 跟随 staging 日志
	docker compose logs -f --tail=100

e2e-local: ## 本地复现 CI：起 staging → 健康等待 → 浏览器冒烟
	docker compose up -d --build --remove-orphans
	@echo "waiting for backend/frontend health..."
	@for i in $$(seq 1 40); do curl -fsS -o /dev/null http://localhost:8080/api/v1/yang/modules && break || sleep 3; done
	@for i in $$(seq 1 20); do curl -fsS -o /dev/null http://localhost:3002/healthz && break || sleep 3; done
	cd frontend && npm ci --prefer-offline --no-audit --fund=false && \
		npx playwright install chromium && \
		npx playwright test tests/staging-smoke.spec.ts --project=chromium --reporter=list
	@echo "✅ e2e-local 通过（staging 仍在运行，make staging-down 可停止）"
