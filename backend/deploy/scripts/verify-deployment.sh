#!/bin/bash
# USMP Kind 部署校验脚本 - 用于独立校验部署状态
set +e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

CLUSTER_NAME="usmp-dev"
NAMESPACE="usmp-system"
KUBE_CONTEXT="kind-${CLUSTER_NAME}"

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

print_header "USMP 部署状态校验"

all_passed=true
total_checks=0
passed_checks=0

check() {
    total_checks=$((total_checks + 1))
    if eval "$1"; then
        passed_checks=$((passed_checks + 1))
        return 0
    else
        return 1
    fi
}

# 1. 集群状态检查
echo ""
print_info "1. 集群状态检查"
if check "kubectl --context=${KUBE_CONTEXT} cluster-info > /dev/null 2>&1"; then
    print_success "Kubernetes 集群运行正常"
else
    print_error "Kubernetes 集群异常或不可访问"
    all_passed=false
fi

# 2. Namespace 检查
echo ""
print_info "2. Namespace 检查"
if check "kubectl --context=${KUBE_CONTEXT} get namespace ${NAMESPACE} > /dev/null 2>&1"; then
    print_success "Namespace ${NAMESPACE} 存在"
else
    print_error "Namespace ${NAMESPACE} 不存在"
    all_passed=false
fi

# 3. CRD 检查
echo ""
print_info "3. CRD 检查"
required_crds=(
    "nativedeviceconfigs.core.usmp.io"
)

for crd in "${required_crds[@]}"; do
    if check "kubectl --context=${KUBE_CONTEXT} get crd $crd > /dev/null 2>&1"; then
        print_success "CRD ${crd} 已注册"
    else
        print_error "CRD ${crd} 未注册"
        all_passed=false
    fi
done

# 4. Controller 检查
echo ""
print_info "4. Controller 检查"

# Deployment 检查
if check "kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get deployment usmp-controller > /dev/null 2>&1"; then
    print_success "Deployment usmp-controller 存在"
else
    print_error "Deployment usmp-controller 不存在"
    all_passed=false
fi

# ReplicaSet 检查
replica_ready=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get deployment usmp-controller -o jsonpath='{.status.readyReplicas}' 2>/dev/null)
replica_desired=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get deployment usmp-controller -o jsonpath='{.spec.replicas}' 2>/dev/null)
if [ "$replica_ready" = "$replica_desired" ] && [ -n "$replica_ready" ]; then
    print_success "所有 Controller 副本就绪 (${replica_ready}/${replica_desired})"
else
    print_warning "Controller 副本未全部就绪 (${replica_ready:-0}/${replica_desired:-1})"
fi

# Pod 状态检查
controller_pods=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get pods -l control-plane=controller-manager -o jsonpath='{.items[*].status.phase}' 2>/dev/null)
if [[ $controller_pods == *"Running"* ]]; then
    print_success "Controller Pod 处于 Running 状态"
else
    print_error "Controller Pod 未就绪，当前状态: ${controller_pods:-N/A}"
    all_passed=false
fi

# Service 检查
if check "kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get service usmp-controller > /dev/null 2>&1"; then
    nodeport=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get service usmp-controller -o jsonpath='{.spec.ports[?(@.name=="api")].nodePort}')
    if [ -n "$nodeport" ]; then
        print_success "Controller Service 存在，NodePort: ${nodeport}"
    else
        print_warning "Controller Service 存在但未找到 api 端口"
    fi
else
    print_error "Controller Service 不存在"
    all_passed=false
fi

# 5. 前端检查
echo ""
print_info "5. 前端检查"

# Deployment 检查
if check "kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get deployment usmp-frontend > /dev/null 2>&1"; then
    print_success "Deployment usmp-frontend 存在"
else
    print_error "Deployment usmp-frontend 不存在"
    all_passed=false
fi

# ReplicaSet 检查
replica_ready=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get deployment usmp-frontend -o jsonpath='{.status.readyReplicas}' 2>/dev/null)
replica_desired=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get deployment usmp-frontend -o jsonpath='{.spec.replicas}' 2>/dev/null)
if [ "$replica_ready" = "$replica_desired" ] && [ -n "$replica_ready" ]; then
    print_success "所有前端副本就绪 (${replica_ready}/${replica_desired})"
else
    print_warning "前端副本未全部就绪 (${replica_ready:-0}/${replica_desired:-1})"
fi

# Pod 状态检查
frontend_pods=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get pods -l app=usmp-frontend -o jsonpath='{.items[*].status.phase}' 2>/dev/null)
if [[ $frontend_pods == *"Running"* ]]; then
    print_success "前端 Pod 处于 Running 状态"
else
    print_error "前端 Pod 未就绪，当前状态: ${frontend_pods:-N/A}"
    all_passed=false
fi

# Service 检查
if check "kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get service usmp-frontend > /dev/null 2>&1"; then
    nodeport=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get service usmp-frontend -o jsonpath='{.spec.ports[0].nodePort}')
    print_success "前端 Service 存在，NodePort: ${nodeport}"
else
    print_error "前端 Service 不存在"
    all_passed=false
fi

# 6. NETCONF 模拟器检查
echo ""
print_info "6. NETCONF 模拟器检查"

# Deployment 检查
if check "kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get deployment netconf-simulator > /dev/null 2>&1"; then
    print_success "Deployment netconf-simulator 存在"
else
    print_error "Deployment netconf-simulator 不存在"
    all_passed=false
fi

# Pod 状态检查
simulator_pods=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get pods -l app=netconf-simulator -o jsonpath='{.items[*].status.phase}' 2>/dev/null)
if [[ $simulator_pods == *"Running"* ]]; then
    print_success "NETCONF 模拟器 Pod 处于 Running 状态"
else
    print_error "NETCONF 模拟器 Pod 未就绪，当前状态: ${simulator_pods:-N/A}"
    all_passed=false
fi

# Service 检查
if check "kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get service netconf-simulator > /dev/null 2>&1"; then
    nodeport=$(kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get service netconf-simulator -o jsonpath='{.spec.ports[0].nodePort}')
    print_success "NETCONF 模拟器 Service 存在，NodePort: ${nodeport}"
else
    print_error "NETCONF 模拟器 Service 不存在"
    all_passed=false
fi

# 7. RBAC 检查
echo ""
print_info "7. RBAC 检查"

# 前端 ServiceAccount 检查
if check "kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get serviceaccount usmp-frontend > /dev/null 2>&1"; then
    print_success "前端 ServiceAccount 存在"
else
    print_error "前端 ServiceAccount 不存在"
    all_passed=false
fi

# Controller ServiceAccount 检查
if check "kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get serviceaccount usmp-controller > /dev/null 2>&1"; then
    print_success "Controller ServiceAccount 存在"
else
    print_error "Controller ServiceAccount 不存在"
    all_passed=false
fi

# 8. Pod 资源概览
echo ""
print_info "8. Pod 资源概览"
kubectl --context=${KUBE_CONTEXT} -n ${NAMESPACE} get pods -o wide 2>/dev/null || print_warning "无法获取 Pod 信息"

# 9. 服务访问地址
echo ""
print_info "9. 服务访问地址"
echo -e "  前端界面:    ${YELLOW}http://localhost:30081${NC}"
echo -e "  后端 API:    ${YELLOW}http://localhost:30080${NC}"
echo -e "  NETCONF 模拟器: ${YELLOW}localhost:30830${NC}"

# 校验结果汇总
echo ""
print_header "校验结果汇总"
echo ""
echo -e "总检查项: ${total_checks}"
echo -e "通过:     ${GREEN}${passed_checks}${NC}"
echo -e "失败:     ${RED}$((total_checks - passed_checks))${NC}"
echo ""

if $all_passed; then
    print_success "🎉 所有服务状态校验通过！"
    exit 0
else
    print_warning "⚠️ 部分检查未通过，请检查上述错误信息"
    echo ""
    echo "调试命令："
    echo "  查看所有 Pod 状态: make kind-status"
    echo "  查看 Controller 日志: make kind-logs"
    echo "  查看前端日志: make kind-frontend-logs"
    echo "  查看模拟器日志: make kind-simulator-logs"
    exit 1
fi
