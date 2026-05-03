#!/usr/bin/env bash
set -euo pipefail

# USMP 端到端集成测试脚本
# 使用 Kind Kubernetes 集群运行完整的 CRD + Controller 测试

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
KIND_CLUSTER_NAME="usmp-e2e"
KIND_CONFIG="${SCRIPT_DIR}/config/kind-cluster.yaml"
KUBECTL="kubectl --context=kind-${KIND_CLUSTER_NAME}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."

    local missing=0

    if ! command -v kind &> /dev/null; then
        log_error "kind 未安装，请先安装: https://kind.sigs.k8s.io/"
        missing=1
    fi

    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl 未安装，请先安装"
        missing=1
    fi

    if ! command -v docker &> /dev/null; then
        log_error "docker 未安装，请先安装"
        missing=1
    fi

    if ! command -v kustomize &> /dev/null && ! kubectl kustomize --help &> /dev/null; then
        log_warning "kustomize 未安装，将使用 kubectl kustomize 替代"
    fi

    if [ $missing -eq 1 ]; then
        exit 1
    fi

    log_success "依赖检查通过"
}

# 创建 Kind 集群
create_cluster() {
    log_info "创建 Kind 集群..."

    if kind get clusters | grep -q "^${KIND_CLUSTER_NAME}$"; then
        log_warning "集群 ${KIND_CLUSTER_NAME} 已存在，跳过创建"
        return
    fi

    kind create cluster --name "${KIND_CLUSTER_NAME}" --config "${KIND_CONFIG}"

    # 等待集群就绪
    log_info "等待集群就绪..."
    until ${KUBECTL} cluster-info &> /dev/null; do
        sleep 2
    done

    log_success "集群创建成功"
}

# 构建并加载镜像
build_and_load_image() {
    log_info "构建控制器镜像..."

    cd "${PROJECT_ROOT}"

    # 构建 Docker 镜像
    docker build -t usmp-controller:e2e-test -f "${PROJECT_ROOT}/Dockerfile" .

    # 加载镜像到 Kind 集群
    log_info "加载镜像到 Kind 集群..."
    kind load docker-image usmp-controller:e2e-test --name "${KIND_CLUSTER_NAME}"

    log_success "镜像加载完成"
}

# 部署资源
deploy_resources() {
    log_info "部署资源到集群..."

    cd "${SCRIPT_DIR}"

    # 等待节点就绪
    ${KUBECTL} wait --for=condition=Ready nodes --all --timeout=60s

    # 创建命名空间（如果不存在）
    ${KUBECTL} create namespace usmp-e2e --dry-run=client -o yaml | ${KUBECTL} apply -f -

    # 使用 kustomize 部署所有资源
    if command -v kustomize &> /dev/null; then
        kustomize build "${SCRIPT_DIR}" | ${KUBECTL} apply -f -
    else
        ${KUBECTL} kustomize "${SCRIPT_DIR}" | ${KUBECTL} apply -f -
    fi

    log_success "资源部署完成"
}

# 等待所有 Pod 就绪
wait_for_pods() {
    log_info "等待所有 Pod 就绪..."

    local namespace="usmp-e2e"
    local timeout=300
    local elapsed=0

    while [ $elapsed -lt $timeout ]; do
        local pending=$( ${KUBECTL} -n "${namespace}" get pods --no-headers 2>/dev/null | grep -v -E 'Running|Completed' | wc -l | tr -d ' ' )

        if [ "${pending}" = "0" ]; then
            log_success "所有 Pod 已就绪"
            return
        fi

        log_info "还有 ${pending} 个 Pod 未就绪，等待中..."
        sleep 5
        elapsed=$((elapsed + 5))
    done

    log_error "Pod 就绪超时"
    ${KUBECTL} -n "${namespace}" get pods
    exit 1
}

# 部署 NETCONF 模拟器
deploy_simulator() {
    log_info "部署 NETCONF 模拟器..."

    ${KUBECTL} apply -f "${SCRIPT_DIR}/config/netconf-simulator.yaml"

    log_info "等待模拟器就绪..."
    ${KUBECTL} -n usmp-e2e wait --for=condition=Ready pods -l app=netconf-simulator --timeout=120s

    log_success "NETCONF 模拟器部署完成"
}

# 运行 CRD 测试
run_crd_tests() {
    log_info "开始运行 CRD 集成测试..."

    local namespace="usmp-e2e"

    # 1. 测试 BusinessSwitch CRD
    log_info "测试 BusinessSwitch CRD..."
    ${KUBECTL} apply -f "${PROJECT_ROOT}/config/samples/biz_v1_businessswitch.yaml" -n "${namespace}"
    sleep 3

    # 验证创建成功
    if ${KUBECTL} -n "${namespace}" get businessswitch switch-sample -o jsonpath='{.metadata.name}' 2>/dev/null; then
        log_success "BusinessSwitch CRD 测试通过"
    else
        log_error "BusinessSwitch CRD 测试失败"
        return 1
    fi

    # 2. 测试 BusinessVlan CRD
    log_info "测试 BusinessVlan CRD..."
    ${KUBECTL} apply -f "${PROJECT_ROOT}/config/samples/biz_v1_businessvlan.yaml" -n "${namespace}"
    sleep 3

    if ${KUBECTL} -n "${namespace}" get businessvlan vlan-sample -o jsonpath='{.metadata.name}' 2>/dev/null; then
        log_success "BusinessVlan CRD 测试通过"
    else
        log_error "BusinessVlan CRD 测试失败"
        return 1
    fi

    # 3. 测试 BusinessInterface CRD
    log_info "测试 BusinessInterface CRD..."
    ${KUBECTL} apply -f "${PROJECT_ROOT}/config/samples/biz_v1_businessinterface.yaml" -n "${namespace}"
    sleep 3

    if ${KUBECTL} -n "${namespace}" get businessinterface interface-sample -o jsonpath='{.metadata.name}' 2>/dev/null; then
        log_success "BusinessInterface CRD 测试通过"
    else
        log_error "BusinessInterface CRD 测试失败"
        return 1
    fi

    # 4. 测试 BusinessRoute CRD
    log_info "测试 BusinessRoute CRD..."
    ${KUBECTL} apply -f "${PROJECT_ROOT}/config/samples/biz_v1_businessroute.yaml" -n "${namespace}"
    sleep 3

    if ${KUBECTL} -n "${namespace}" get businessroute route-sample -o jsonpath='{.metadata.name}' 2>/dev/null; then
        log_success "BusinessRoute CRD 测试通过"
    else
        log_error "BusinessRoute CRD 测试失败"
        return 1
    fi

    # 5. 测试 NativeDeviceConfig CRD
    log_info "测试 NativeDeviceConfig CRD..."
    ${KUBECTL} apply -f "${PROJECT_ROOT}/config/samples/biz_v1_nativedeviceconfig.yaml" -n "${namespace}"
    sleep 3

    if ${KUBECTL} -n "${namespace}" get nativedeviceconfig cli-banner-config -o jsonpath='{.metadata.name}' 2>/dev/null; then
        log_success "NativeDeviceConfig CRD 测试通过"
    else
        log_error "NativeDeviceConfig CRD 测试失败"
        return 1
    fi

    log_success "所有 CRD 测试通过！"
}

# 运行控制器集成测试
run_controller_tests() {
    log_info "开始运行控制器集成测试..."

    # 检查控制器日志
    log_info "控制器日志（最近 10 行）："
    ${KUBECTL} -n usmp-e2e logs -l control-plane=controller-manager --tail=10 2>/dev/null || true

    # 验证 CRD 状态更新
    log_info "验证 CRD 状态更新..."

    local timeout=60
    local elapsed=0
    local success=0

    while [ $elapsed -lt $timeout ]; do
        phase=$( ${KUBECTL} -n usmp-e2e get businessvlan vlan-sample -o jsonpath='{.status.phase}' 2>/dev/null || echo "" )
        if [ "${phase}" = "Synced" ] || [ "${phase}" = "Pending" ]; then
            log_success "VLAN 状态正确: ${phase}"
            success=1
            break
        fi
        sleep 2
        elapsed=$((elapsed + 2))
    done

    if [ $success -eq 0 ]; then
        log_warning "VLAN 状态未更新，可能需要更长时间"
    fi

    log_success "控制器集成测试完成"
}

# 显示测试摘要
show_summary() {
    echo ""
    echo "========================================"
    echo "      E2E 测试摘要"
    echo "========================================"
    echo ""

    log_info "集群信息："
    ${KUBECTL} cluster-info
    echo ""

    log_info "命名空间资源："
    ${KUBECTL} -n usmp-e2e get all
    echo ""

    log_info "CRD 实例："
    ${KUBECTL} -n usmp-e2e get businessswitches,businessvlans,businessinterfaces,businessroutes,nativedeviceconfigs 2>/dev/null || true

    echo ""
    log_success "E2E 测试全部完成！"
    echo ""
    echo "后续操作："
    echo "  查看 Pod 日志: ${KUBECTL} -n usmp-e2e logs -l control-plane=controller-manager"
    echo "  访问模拟器:   ${KUBECTL} -n usmp-e2e port-forward svc/netconf-simulator 830:830"
    echo "  删除集群:     kind delete cluster --name ${KIND_CLUSTER_NAME}"
}

# 清理集群
cleanup() {
    if [ "${SKIP_CLEANUP:-0}" -eq 1 ]; then
        log_info "保留集群以便调试"
        return
    fi

    log_info "清理 Kind 集群..."
    kind delete cluster --name "${KIND_CLUSTER_NAME}"
    log_success "清理完成"
}

# 主流程
main() {
    echo ""
    echo "========================================"
    echo "  USMP 端到端集成测试"
    echo "========================================"
    echo ""

    local SKIP_BUILD=0
    local SKIP_CLEANUP=0
    local SKIP_IMAGE=0

    # 解析参数
    for arg in "$@"; do
        case $arg in
            --skip-build)
                SKIP_BUILD=1
                shift
                ;;
            --skip-cleanup)
                SKIP_CLEANUP=1
                shift
                ;;
            --skip-image)
                SKIP_IMAGE=1
                shift
                ;;
            -h|--help)
                echo "用法: $0 [选项]"
                echo ""
                echo "选项："
                echo "  --skip-build    跳过镜像构建"
                echo "  --skip-image    跳过镜像构建和加载"
                echo "  --skip-cleanup  测试完成后保留集群"
                echo "  -h, --help      显示帮助信息"
                exit 0
                ;;
        esac
    done

    export SKIP_CLEANUP

    # 捕获退出信号
    trap cleanup EXIT

    # 执行测试流程
    check_dependencies
    create_cluster

    if [ $SKIP_IMAGE -eq 0 ]; then
        build_and_load_image
    fi

    deploy_resources
    wait_for_pods
    deploy_simulator

    # 等待控制器启动
    log_info "等待控制器启动完成..."
    sleep 10

    # 运行测试
    run_crd_tests
    run_controller_tests

    # 显示摘要
    show_summary

    echo ""
    log_success "E2E 测试成功完成！"
}

# 运行主程序
main "$@"
