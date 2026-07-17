.PHONY: all build test test-unit test-integration test-e2e test-all clean help e2e-cluster e2e-deploy e2e-clean test-e2e-go test-e2e-go-verbose test-e2e-kind test-all-full e2e-kind-setup install-ginkgo install-envtest docker-build docker-build-dev kind-load-image e2e-status e2e-logs e2e-simulator-logs kind-kubeconfig kind-export-kubeconfig e2e-test-info kind-up kind-cluster kind-delete kind-clean kind-deploy kind-redeploy kind-status kind-logs kind-frontend-logs kind-simulator-logs kind-load-images docker-build-frontend kind-shell kind-port-forward

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

## test-integration: 运行集成测试（需要先启动服务器）
test-integration:
	@echo "Running integration tests..."
	$(GO) test $(GOFLAGS) -tags=integration ./test/integration/...
	@echo "Integration tests completed"

## test-all: 运行所有测试（单元 + 集成）
test-all: test-unit test-integration
	@echo "All tests completed"

## test-vlan: 只运行 VLAN 相关集成测试
test-vlan:
	@echo "Running VLAN integration tests..."
	$(GO) test $(GOFLAGS) -tags=integration ./test/integration/... -run TestE2E_VLAN
	@echo "VLAN tests completed"

## test-device: 只运行设备相关集成测试
test-device:
	@echo "Running device integration tests..."
	$(GO) test $(GOFLAGS) -tags=integration ./test/integration/... -run TestE2E_Device
	@echo "Device tests completed"

## coverage: 生成测试覆盖率报告
coverage:
	@echo "Generating test coverage report..."
	$(GO) test $(GOFLAGS) -tags=integration ./... -coverprofile=coverage.out
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
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {split($$1, a, ":"); printf "  \033[36m%-15s\033[0m %s\n", a[1], $$2}' $(MAKEFILE_LIST) | sort

## integration-setup: 集成测试环境准备
integration-setup: build
	@echo "Setting up integration test environment..."
	@echo "Make sure to run 'make run' in a separate terminal before running integration tests"
	@echo ""
	@echo "Quick start:"
	@echo "  Terminal 1: make run"
	@echo "  Terminal 2: make test-integration"
	@echo ""

## check-server: 检查服务器是否运行
check-server:
	@curl -s http://localhost:8080/api/v1/yang/modules > /dev/null 2>&1 && echo "✓ Server is running" || echo "✗ Server is NOT running - please run 'make run' first"

# =========================
# E2E 测试相关命令
# =========================

## e2e-cluster: 创建 E2E 测试用 Kind 集群
e2e-cluster:
	@echo "Creating Kind cluster for E2E testing..."
	@kind create cluster --name usmp-e2e --config ./test/e2e/config/kind-cluster.yaml
	@echo "Kind cluster created successfully"

## e2e-clean: 删除 E2E 测试用 Kind 集群
e2e-clean:
	@echo "Cleaning up E2E test cluster..."
	@kind delete cluster --name usmp-e2e
	@echo "E2E test cluster cleaned up"

## e2e-deploy: 部署所有资源到 E2E 集群
e2e-deploy:
	@echo "Deploying resources to E2E cluster..."
	@kubectl --context=kind-usmp-e2e create namespace usmp-e2e --dry-run=client -o yaml | kubectl --context=kind-usmp-e2e apply -f -
	@kubectl --context=kind-usmp-e2e apply -f config/crd/bases/
	@kubectl --context=kind-usmp-e2e apply -f test/e2e/config/netconf-simulator.yaml
	@echo "Waiting for NETCONF simulator to be ready..."
	@kubectl --context=kind-usmp-e2e -n usmp-e2e wait --for=condition=Ready pods -l app=netconf-simulator --timeout=120s
	@echo "All resources deployed successfully"

## test-e2e: 运行完整的端到端测试（使用脚本）
test-e2e:
	@echo "Running end-to-end tests..."
	@./test/e2e/run_e2e.sh
	@echo "E2E tests completed"

## e2e-quick: 快速运行 E2E 集成测试（不构建镜像）
e2e-quick:
	@echo "Running quick E2E tests (skipping image build)..."
	@./test/e2e/run_e2e.sh --skip-image --skip-cleanup
	@echo "Quick E2E tests completed"

## e2e-status: 显示 E2E 集群状态
e2e-status:
	@echo "=== E2E Test Cluster Status ==="
	@echo ""
	@echo "Cluster info:"
	@kubectl --context=kind-usmp-e2e cluster-info 2>/dev/null || echo "Cluster not found or not accessible"
	@echo ""
	@echo "Namespace usmp-e2e resources:"
	@kubectl --context=kind-usmp-e2e -n usmp-e2e get all 2>/dev/null || echo "Namespace usmp-e2e not found"
	@echo ""
	@echo "CRD Instances:"
	@kubectl --context=kind-usmp-e2e -n usmp-e2e get businessswitches,businessvlans,businessinterfaces,businessroutes,nativedeviceconfigs 2>/dev/null || echo "No CRD instances found"

## e2e-logs: 查看 E2E 控制器日志
e2e-logs:
	@echo "Controller manager logs:"
	@kubectl --context=kind-usmp-e2e -n usmp-e2e logs -l control-plane=controller-manager --tail=50 -f 2>/dev/null || echo "No controller pod found or log error"

## e2e-simulator-logs: 查看 NETCONF 模拟器日志
e2e-simulator-logs:
	@echo "NETCONF Simulator logs:"
	@kubectl --context=kind-usmp-e2e -n usmp-e2e logs -l app=netconf-simulator --tail=50 -f 2>/dev/null || echo "No simulator pod found or log error"

## docker-build: 构建控制器 Docker 镜像
docker-build:
	@echo "Building Docker image..."
	docker build -t usmp-controller:e2e-test -f Dockerfile .
	@echo "Docker image built successfully"

## kind-load-image: 加载镜像到 Kind 集群
kind-load-image:
	@echo "Loading Docker image to Kind cluster..."
	kind load docker-image usmp-controller:e2e-test --name usmp-e2e
	@echo "Docker image loaded to Kind cluster"

docker-build-dev: ## 构建控制器开发环境 Docker 镜像
	@echo "Building development Docker image..."
	docker build -t usmp-controller:latest -f Dockerfile .
	@echo "Development Docker image built successfully"

## install-ginkgo: 安装 Ginkgo 测试框架
install-ginkgo:
	@echo "Installing Ginkgo testing framework..."
	go install github.com/onsi/ginkgo/v2/ginkgo@latest
	@echo "Ginkgo installed successfully"

## test-e2e-go: 使用 Ginkgo 运行 Go E2E 测试（envtest 环境，快速）
test-e2e-go:
	@echo "Running Go E2E tests with envtest..."
	go test -tags=e2e -v ./test/e2e/... -ginkgo.v
	@echo "Go E2E tests completed"

## test-e2e-go-verbose: 运行详细的 Go E2E 测试
test-e2e-go-verbose:
	@echo "Running verbose Go E2E tests..."
	go test -tags=e2e -v ./test/e2e/... -ginkgo.v -ginkgo.fail-fast
	@echo "Go E2E tests completed"

## test-e2e-kind: 使用 Kind 集群运行完整 E2E 测试（真实环境，较慢）
test-e2e-kind:
	@echo "Running full E2E tests with Kind cluster..."
	@./test/e2e/run_e2e.sh
	@echo "Kind E2E tests completed"

## test-all-full: 运行所有测试（单元 + 集成 + E2E）
test-all-full: test-unit test-integration test-e2e-go
	@echo "All tests (unit, integration, E2E) completed successfully"

## e2e-kind-setup: 只设置 Kind 测试集群（不运行测试）
e2e-kind-setup: e2e-cluster e2e-deploy
	@echo "Kind E2E cluster setup completed"
	@echo "Run 'make e2e-status' to check cluster status"
	@echo "Run 'make test-e2e-kind' to run tests"

## kind-kubeconfig: 获取 Kind 集群的 kubeconfig
kind-kubeconfig:
	@echo "Kind cluster kubeconfig:"
	@kind get kubeconfig --name usmp-e2e

## kind-export-kubeconfig: 导出 Kind 集群 kubeconfig 到文件
kind-export-kubeconfig:
	@echo "Exporting Kind kubeconfig to ~/.kube/usmp-e2e-config..."
	@kind get kubeconfig --name usmp-e2e > ~/.kube/usmp-e2e-config
	@echo "Usage: export KUBECONFIG=~/.kube/usmp-e2e-config"

## install-envtest: 安装 envtest 二进制文件（运行 Go E2E 测试需要）
install-envtest:
	@echo "Installing envtest tools..."
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	setup-envtest use 1.29.x! --bin-dir ./bin/k8s
	@echo "envtest installed successfully"

## e2e-test-info: 显示 E2E 测试说明
e2e-test-info:
	@echo "=== USMP E2E Testing Guide ==="
	@echo ""
	@echo "Two types of E2E tests are available:"
	@echo ""
	@echo "1. Go E2E Tests (envtest - fast, in-memory cluster):"
	@echo "   make test-e2e-go          # Run all Go E2E tests"
	@echo "   make test-e2e-go-verbose  # Run with verbose output"
	@echo ""
	@echo "   Requirements:"
	@echo "   - Run 'make install-envtest' first to install envtest binaries"
	@echo ""
	@echo "2. Kind Cluster E2E Tests (full Kubernetes cluster):"
	@echo "   make test-e2e-kind        # Run full E2E test with Kind cluster"
	@echo "   make e2e-kind-setup       # Setup Kind cluster without running tests"
	@echo "   make e2e-clean            # Delete Kind cluster"
	@echo ""
	@echo "   Requirements:"
	@echo "   - kind (installed)"
	@echo "   - docker (installed)"
	@echo "   - kubectl (installed)"
	@echo ""
	@echo "Debug Commands:"
	@echo "   make e2e-status           # Show cluster status"
	@echo "   make e2e-logs             # View controller logs"
	@echo "   make e2e-simulator-logs   # View NETCONF simulator logs"
	@echo ""
	@echo "Kind Kubeconfig:"
	@echo "   export KUBECONFIG=\$$(kind get kubeconfig --name usmp-e2e)"
	@echo "   OR use: kubectl --context=kind-usmp-e2e"

# =========================
# Kind 开发环境相关命令
# =========================

## deploy: 一键部署并校验服务（推荐使用）
deploy: one-click-deploy

## one-click-deploy: 一键部署完整环境，包含前置检查、部署和校验（推荐使用）
one-click-deploy:
	@echo "🚀 开始 USMP Kind 一键部署..."
	@./deploy/scripts/one-click-deploy.sh

## kind-verify: 校验已部署服务的状态
kind-verify:
	@echo "🔍 开始校验 USMP 部署状态..."
	@./deploy/scripts/verify-deployment.sh

kind-up: kind-cluster kind-load-images kind-deploy ## 创建集群并部署完整开发环境
	@echo ""
	@echo "✅ USMP Kind 开发环境部署完成！"
	@echo ""
	@echo "访问地址："
	@echo "  前端界面: http://localhost:30081"
	@echo "  后端 API: http://localhost:30080"
	@echo "  NETCONF 模拟器: localhost:30830"
	@echo ""
	@echo "查看状态: make kind-status"
	@echo "查看日志: make kind-logs"
	@echo "清理环境: make kind-clean"

kind-cluster: ## 创建 Kind 集群
	@echo "Creating Kind cluster for USMP development..."
	kind create cluster --name usmp-dev --config deploy/kind-cluster.yaml
	@echo "Kind cluster created successfully"

kind-delete: kind-clean ## 删除 Kind 集群 (别名)

kind-clean: ## 删除 Kind 集群
	@echo "Cleaning up Kind cluster..."
	kind delete cluster --name usmp-dev
	@echo "Kind cluster cleaned up"

kind-deploy: ## 部署/更新所有组件到 Kind 集群
	@echo "Deploying resources to Kind cluster..."
	kubectl --context=kind-usmp-dev apply -k deploy/manifests/
	@echo "Waiting for pods to be ready..."
	kubectl --context=kind-usmp-dev -n usmp-system wait --for=condition=Ready pods --all --timeout=120s || true
	@echo "Deployment completed"

kind-redeploy: ## 重新部署（先删除再应用）
	@echo "Redeploying all resources..."
	kubectl --context=kind-usmp-dev delete -k deploy/manifests/ || true
	kubectl --context=kind-usmp-dev apply -k deploy/manifests/
	@echo "Redeployment completed"

kind-status: ## 查看 Kind 集群状态
	@echo "=== USMP Kind Development Cluster Status ==="
	@echo ""
	@echo "Cluster info:"
	@kubectl --context=kind-usmp-dev cluster-info 2>/dev/null || echo "Cluster not found or not accessible"
	@echo ""
	@echo "Namespace usmp-system resources:"
	@kubectl --context=kind-usmp-dev -n usmp-system get all 2>/dev/null || echo "Namespace usmp-system not found"
	@echo ""
	@echo "CRD Instances:"
	@kubectl --context=kind-usmp-dev -n usmp-system get businessswitches,businessvlans,businessinterfaces,businessroutes,nativedeviceconfigs 2>/dev/null || echo "No CRD instances found"

kind-logs: ## 查看 Controller 日志
	@echo "Controller manager logs:"
	@kubectl --context=kind-usmp-dev -n usmp-system logs -l control-plane=controller-manager --tail=50 -f 2>/dev/null || echo "No controller pod found or log error"

kind-frontend-logs: ## 查看前端日志
	@echo "Frontend logs:"
	@kubectl --context=kind-usmp-dev -n usmp-system logs -l app=usmp-frontend --tail=50 -f 2>/dev/null || echo "No frontend pod found or log error"

kind-simulator-logs: ## 查看模拟器日志
	@echo "NETCONF Simulator logs:"
	@kubectl --context=kind-usmp-dev -n usmp-system logs -l app=netconf-simulator --tail=50 -f 2>/dev/null || echo "No simulator pod found or log error"

kind-load-images: docker-build-dev docker-build-frontend ## 加载本地镜像到 Kind 集群
	@echo "Loading Docker images to Kind cluster..."
	kind load docker-image usmp-controller:latest --name usmp-dev
	kind load docker-image usmp-frontend:latest --name usmp-dev
	@echo "Docker images loaded to Kind cluster"

docker-build-frontend: ## 构建前端 Docker 镜像
	@echo "Building frontend Docker image..."
	cd ../frontend && docker build -t usmp-frontend:latest .
	@echo "Frontend Docker image built successfully"

kind-shell: ## 在 Controller Pod 中打开 Shell
	@echo "Opening shell in controller pod..."
	@kubectl --context=kind-usmp-dev -n usmp-system exec -it deployment/usmp-controller -- sh 2>/dev/null || echo "Failed to open shell"

kind-port-forward: ## 端口转发（用于本地调试）
	@echo "Starting port forwarding..."
	@echo "API: 8080 -> 30080, Frontend: 30081 -> 30081"
	kubectl --context=kind-usmp-dev -n usmp-system port-forward service/usmp-controller 8080:8080 &
	kubectl --context=kind-usmp-dev -n usmp-system port-forward service/usmp-frontend 30081:80 &
	@echo "Port forwarding started. Press Ctrl+C to stop."
	wait
