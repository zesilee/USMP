.PHONY: all build run test-unit coverage lint clean help

# 项目配置
BINARY_NAME=usmp-backend
BUILD_DIR=bin

# Go 配置
GO=go
GOFLAGS=-v

# 默认目标
all: build

## build: 编译后端服务
build:
	@echo "Building backend..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Build completed: $(BUILD_DIR)/$(BINARY_NAME)"

## run: 启动后端服务
run:
	@echo "Starting USMP backend server..."
	$(GO) run main.go

## test-unit: 运行单元测试
test-unit:
	@echo "Running unit tests..."
	$(GO) test $(GOFLAGS) ./internal/... ./pkg/...
	@echo "Unit tests completed"

## coverage: 生成测试覆盖率报告
coverage:
	@echo "Generating test coverage report..."
	$(GO) test $(GOFLAGS) ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

## lint: 运行代码检查
lint:
	@echo "Running linter..."
	@golangci-lint run ./... 2>/dev/null || echo "golangci-lint not installed, skipping"

## clean: 清理构建产物
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) coverage.out coverage.html
	@echo "Clean completed"

## help: 显示帮助信息
help:
	@echo "USMP Backend - Available targets:"
	@echo ""
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'

# Stack A 时代的 e2e/kind/deploy/docker target 已随 backend/{test/e2e,test/integration,
# deploy,config} 目录退役删除（change retire-stacka-residue）。现行链：
#   - 集成测试：包内 *_integration_test.go + simulator/netconfsim（B2）
#   - E2E/部署：顶层 make e2e-local / scripts/kind-deploy.sh / deploy/
