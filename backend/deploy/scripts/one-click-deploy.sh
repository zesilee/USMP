#!/bin/bash
# USMP Kind 一键部署脚本 - 包含完整的服务状态校验
set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

CLUSTER_NAME="usmp-dev"
NAMESPACE="usmp-system"
KUBE_CONTEXT="kind-${CLUSTER_NAME}"

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅  $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌  $1${NC}"
}

print_header() {
    echo ""
    echo -e "${BLUE}=======================================${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}=======================================${NC}"
}

# 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_error "命令 $1 不存在，请先安装"
        exit 1
    fi
}

# 1. 前置环境检查
prerequisite_check() {
    print_header "1/6 - 前置环境检查"

    print_info "检查必要命令..."
    check_command kind
    check_command kubectl
    check_command docker

    print_info "检查 Docker 运行状态..."
    if ! docker info > /dev/null 2>&1; then
        print_error "Docker 未运行，请先启动 Docker"
        exit 1
    fi
    print_success "Docker 运行正常"

    print_success "前置环境检查通过"
}

# 2. 创建 Kind 集群
create_cluster() {
    print_header "2/6 - 创建 Kind 集群"

    if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
        print_warning "Kind 集群 ${CLUSTER_NAME} 已存在，跳过创建"
    else
        print_info "创建 Kind 集群 ${CLUSTER_NAME}..."
        kind create cluster --name ${CLUSTER_NAME} --config deploy/kind-cluster.yaml
        print_success "Kind 集群创建完成"
    fi

    # 等待 API Server 可用
    print_info "等待 Kubernetes API Server 就绪..."
    sleep 5

    if ! kubectl --context=${KUBE_CONTEXT} cluster-info > /dev/null 2>&1; then
        print_error "Kubernetes API Server 不可用"
        exit 1
    fi
    print_success "Kubernetes API Server 就绪"
}

# 3. 加载 Docker 镜像到 Kind 集群
load_images() {
    print_header "3/6 - 加载 Docker 镜像"

    # 构建镜像
    print_info "构建 Controller 镜像..."
    make docker-build-dev > /dev/null

    print_info "构建前端镜像..."
    make docker-build-frontend > /dev/null

    # 加载镜像
    print_info "加载 Controller 镜像到 Kind 集群..."
    kind load docker-image usmp-controller:latest --name ${CLUSTER_NAME}

    print_info "加载前端镜像到 Kind 集群..."
    kind load docker-image usmp-frontend:latest --name ${CLUSTER_NAME}

    print_success "Docker 镜像加载完成"
}

# 4. 部署所有组件
deploy_components() {
    print_header "4/6 - 部署组件"

    print_info "部署所有 Kubernetes 资源..."
    kubectl --context=${KUBE_CONTEXT} apply -k deploy/manifests/

    print_success "所有组件部署完成"
}

# 5. 等待 Pod 就绪
wait_for_pods() {
    print_header "5/6 - 等待 Pod 就绪"

    local timeout=120
    local interval=5
    local elapsed=0

    print_info "等待所有 Pod 就绪（超时时间: ${timeout}秒）..."

    while [ $elapsed -lt $timeout ]; do
        local pending_pods=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get pods 2>/dev/null | grep -v "Running" | grep -v "Completed" | grep -v "NAME" | wc -l)

        if [ $pending_pods -eq 0 ]; then
            print_success "所有 Pod 已就绪"
            return 0
        fi

        echo -n "."
        sleep $interval
        elapsed=$((elapsed + interval))
    done

    echo ""
    print_warning "部分 Pod 可能未完全就绪，继续检查..."
}

# 6. 完整的服务状态校验
validate_services() {
    print_header "6/6 - 服务状态校验"

    local all_passed=true

    # ============================================
    # 6.1 集群状态检查
    # ============================================
    print_info "6.1 集群状态检查..."
    if kubectl --context=${KUBE_CONTEXT} cluster-info > /dev/null 2>&1; then
        print_success "Kubernetes 集群运行正常"
    else
        print_error "Kubernetes 集群异常"
        all_passed=false
    fi

    # ============================================
    # 6.2 Namespace 检查
    # ============================================
    print_info "6.2 Namespace 检查..."
    if kubectl --context=${KUBE_CONTEXT} get namespace ${NAMESPACE} > /dev/null 2>&1; then
        print_success "Namespace ${NAMESPACE} 存在"
    else
        print_error "Namespace ${NAMESPACE} 不存在"
        all_passed=false
    fi

    # ============================================
    # 6.3 CRD 检查
    # ============================================
    print_info "6.3 CRD 检查..."
    local required_crds=(
        "nativedeviceconfigs.core.usmp.io"
    )

    for crd in "${required_crds[@]}"; do
        if kubectl --context=${KUBE_CONTEXT} get crd $crd > /dev/null 2>&1; then
            print_success "CRD ${crd} 已注册"
        else
            print_error "CRD ${crd} 未注册"
            all_passed=false
        fi
    done

    # ============================================
    # 6.4 Controller 检查
    # ============================================
    print_info "6.4 Controller 检查..."

    # Deployment 检查
    if kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get deployment usmp-controller > /dev/null 2>&1; then
        print_success "Deployment usmp-controller 存在"
    else
        print_error "Deployment usmp-controller 不存在"
        all_passed=false
    fi

    # Pod 状态检查
    local controller_pods=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get pods -l control-plane=controller-manager -o jsonpath='{.items[*].status.phase}' 2>/dev/null)
    if [[ $controller_pods == *"Running"* ]]; then
        print_success "Controller Pod 处于 Running 状态"
    else
        print_error "Controller Pod 未就绪，当前状态: ${controller_pods:-N/A}"
        all_passed=false
    fi

    # Service 检查
    if kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get service usmp-controller > /dev/null 2>&1; then
        local nodeport=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get service usmp-controller -o jsonpath='{.spec.ports[?(@.name=="api")].nodePort}')
        if [ "$nodeport" == "30080" ]; then
            print_success "Controller Service NodePort 正确: ${nodeport}"
        else
            print_warning "Controller Service NodePort: ${nodeport} (期望: 30080)"
        fi
    else
        print_error "Controller Service 不存在"
        all_passed=false
    fi

    # ============================================
    # 6.5 前端检查
    # ============================================
    print_info "6.5 前端检查..."

    # Deployment 检查
    if kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get deployment usmp-frontend > /dev/null 2>&1; then
        print_success "Deployment usmp-frontend 存在"
    else
        print_error "Deployment usmp-frontend 不存在"
        all_passed=false
    fi

    # Pod 状态检查
    local frontend_pods=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get pods -l app=usmp-frontend -o jsonpath='{.items[*].status.phase}' 2>/dev/null)
    if [[ $frontend_pods == *"Running"* ]]; then
        print_success "前端 Pod 处于 Running 状态"
    else
        print_error "前端 Pod 未就绪，当前状态: ${frontend_pods:-N/A}"
        all_passed=false
    fi

    # Service 检查
    if kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get service usmp-frontend > /dev/null 2>&1; then
        local nodeport=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get service usmp-frontend -o jsonpath='{.spec.ports[0].nodePort}')
        if [ "$nodeport" == "30081" ]; then
            print_success "前端 Service NodePort 正确: ${nodeport}"
        else
            print_warning "前端 Service NodePort: ${nodeport} (期望: 30081)"
        fi
    else
        print_error "前端 Service 不存在"
        all_passed=false
    fi

    # ============================================
    # 6.6 NETCONF 模拟器检查
    # ============================================
    print_info "6.6 NETCONF 模拟器检查..."

    # Deployment 检查
    if kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get deployment netconf-simulator > /dev/null 2>&1; then
        print_success "Deployment netconf-simulator 存在"
    else
        print_error "Deployment netconf-simulator 不存在"
        all_passed=false
    fi

    # Pod 状态检查
    local simulator_pods=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get pods -l app=netconf-simulator -o jsonpath='{.items[*].status.phase}' 2>/dev/null)
    if [[ $simulator_pods == *"Running"* ]]; then
        print_success "NETCONF 模拟器 Pod 处于 Running 状态"
    else
        print_error "NETCONF 模拟器 Pod 未就绪，当前状态: ${simulator_pods:-N/A}"
        all_passed=false
    fi

    # Service 检查
    if kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get service netconf-simulator > /dev/null 2>&1; then
        local nodeport=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get service netconf-simulator -o jsonpath='{.spec.ports[0].nodePort}')
        if [ "$nodeport" == "30830" ]; then
            print_success "NETCONF 模拟器 Service NodePort 正确: ${nodeport}"
        else
            print_warning "NETCONF 模拟器 Service NodePort: ${nodeport} (期望: 30830)"
        fi
    else
        print_error "NETCONF 模拟器 Service 不存在"
        all_passed=false
    fi

    # ============================================
    # 6.7 RBAC 检查
    # ============================================
    print_info "6.7 RBAC 检查..."

    # 前端 ServiceAccount 检查
    if kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get serviceaccount usmp-frontend > /dev/null 2>&1; then
        print_success "前端 ServiceAccount 存在"
    else
        print_error "前端 ServiceAccount 不存在"
        all_passed=false
    fi

    # Controller ServiceAccount 检查
    if kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get serviceaccount usmp-controller > /dev/null 2>&1; then
        print_success "Controller ServiceAccount 存在"
    else
        print_error "Controller ServiceAccount 不存在"
        all_passed=false
    fi

    # ============================================
    # 校验结果
    # ============================================
    echo ""
    if $all_passed; then
        print_success "所有服务状态校验通过！"
        return 0
    else
        print_warning "部分检查未通过，请检查日志"
        return 1
    fi
}

# 显示部署完成信息
show_deployment_info() {
    print_header "部署完成"

    echo ""
    echo -e "${GREEN}=========================================${NC}"
    echo -e "${GREEN}  🎉 USMP Kind 开发环境部署完成！${NC}"
    echo -e "${GREEN}=========================================${NC}"
    echo ""
    echo "📋 访问地址："
    echo "   前端界面:    http://localhost:30081"
    echo "   后端 API:    http://localhost:30080"
    echo "   NETCONF 模拟器: localhost:30830"
    echo ""
    echo "🔧 常用命令："
    echo "   查看状态:    make kind-status"
    echo "   查看日志:    make kind-logs"
    echo "   前端日志:    make kind-frontend-logs"
    echo "   模拟器日志:  make kind-simulator-logs"
    echo "   清理环境:    make kind-clean"
    echo "   重新校验:    make kind-verify"
    echo ""
    echo "📝 Kubeconfig 设置："
    echo "   export KUBECONFIG=\$(kind get kubeconfig --name usmp-dev)"
    echo "   或使用: kubectl --context=kind-usmp-dev"
    echo ""
}

# 主函数
main() {
    # 切换到脚本所在目录
    cd "$(dirname "$0")/../.."

    prerequisite_check
    create_cluster
    load_images
    deploy_components
    wait_for_pods
    if validate_services; then
        show_deployment_info
        exit 0
    else
        echo ""
        print_warning "部署完成但部分校验失败，请检查集群状态"
        echo "运行 'make kind-status' 查看详细信息"
        exit 1
    fi
}

# 运行主函数
main
