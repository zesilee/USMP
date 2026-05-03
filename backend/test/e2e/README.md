# USMP E2E 端到端测试

基于 Kind Kubernetes 集群的完整端到端集成测试，验证整个 CRD + Controller 系统的功能。

## 目录结构

```
test/e2e/
├── config/
│   ├── kind-cluster.yaml       # Kind 集群配置
│   └── netconf-simulator.yaml  # NETCONF 模拟器部署
├── patches/
│   └── controller-patch.yaml   # 控制器部署补丁
├── kustomization.yaml          # Kustomize 配置
├── run_e2e.sh                 # 自动化测试脚本
├── e2e_suite_test.go          # Ginkgo 测试套件
├── businessswitch_test.go      # BusinessSwitch CRD 测试
├── businessvlan_test.go        # BusinessVlan CRD 测试
└── README.md                   # 本文档
```

## 快速开始

### 前提条件

- Docker (20.10+)
- Kind (0.20+)
- kubectl (1.28+)
- kustomize (可选，kubectl 内置)
- Go (1.21+)

### 一键运行完整 E2E 测试

```bash
# 进入后端目录
cd backend

# 运行完整的端到端测试（自动创建集群、部署、测试、清理）
make test-e2e
```

### 分步运行

```bash
# 1. 创建 Kind 测试集群
make e2e-cluster

# 2. 部署 CRD 和 NETCONF 模拟器
make e2e-deploy

# 3. 构建并加载控制器镜像（可选）
make docker-build
make kind-load-image

# 4. 查看集群状态
make e2e-status

# 5. 运行 Ginkgo 测试
cd test/e2e
go test -v -tags=e2e ./...

# 6. 测试完成后清理
make e2e-clean
```

## 测试说明

### 测试覆盖范围

E2E 测试覆盖以下功能场景：

#### BusinessSwitch CRD 测试
- ✅ 交换机资源创建和删除
- ✅ Spec 字段验证（DeviceIP, Vendor, Port 等）
- ✅ Status 状态更新（Pending/Syncing/Synced）
- ✅ 多设备批量管理
- ✅ Spec 更新验证

#### BusinessVlan CRD 测试
- ✅ VLAN 配置创建和删除
- ✅ VLAN ID 和名称验证
- ✅ 端口成员配置
- ✅ 管理员状态（Up/Down）
- ✅ Super/Sub VLAN 类型支持
- ✅ 批量 VLAN 创建

#### BusinessInterface CRD 测试
- ✅ 接口配置创建和删除
- ✅ Access/Trunk 模式切换
- ✅ VLAN 成员配置
- ✅ 接口状态同步

#### BusinessRoute CRD 测试
- ✅ 静态路由配置
- ✅ 下一跳和出接口验证
- ✅ 路由优先级配置

#### NativeDeviceConfig CRD 测试
- ✅ 原生配置透传
- ✅ CLI/XML/YANG 格式支持
- ✅ 配置执行状态跟踪

### 测试环境

测试使用以下组件：

1. **Kind Kubernetes 集群**
   - 单控制平面 + 单工作节点
   - 端口映射：8080→80, 8443→443, 30830→830

2. **NETCONF 模拟器**
   - 模拟华为交换机 NETCONF 接口
   - 支持 get-config/edit-config/commit 操作
   - 配置持久化存储

3. **控制器 Manager**
   - 运行所有 CRD 的 Controller
   - 与模拟器 NETCONF 通信

## 测试命令

### 快速测试

```bash
# 快速测试（不重建镜像，保留集群）
make e2e-quick

# 只运行 BusinessSwitch 相关测试
cd test/e2e
go test -v -tags=e2e ./... -run BusinessSwitch

# 只运行 VLAN 相关测试
go test -v -tags=e2e ./... -run BusinessVlan
```

### 调试模式

```bash
# 运行测试但保留集群用于调试
SKIP_CLEANUP=1 ./test/e2e/run_e2e.sh

# 查看控制器日志
make e2e-logs

# 查看模拟器日志
make e2e-simulator-logs

# 查看所有资源
make e2e-status

# 手动执行 kubectl 命令
kubectl --context=kind-usmp-e2e -n usmp-e2e get pods
```

## 测试用例编写规范

### 新增 CRD 测试用例

1. 创建测试文件 `{crd_name}_test.go`
2. 使用 Ginkgo BDD 风格编写测试
3. 覆盖以下场景：
   - 资源创建和删除
   - Spec 字段验证
   - Status 状态更新
   - 异常场景测试

### 测试模板

```go
//go:build e2e
// +build e2e

package e2e

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	bizv1 "github.com/leezesi/usmp/backend/api/v1"
)

var _ = Describe("YourCRD E2E Test", func() {
	const (
		namespace = "usmp-e2e-test"
		timeout   = time.Second * 30
		interval  = time.Second * 1
	)

	Context("创建 YourCRD 资源", func() {
		It("应该成功创建", func() {
			// 测试代码
		})
	})
})
```

## 故障排查

### Kind 集群创建失败

```bash
# 检查 Docker 是否运行
docker info

# 清理旧集群
kind delete cluster --name usmp-e2e

# 重新创建
make e2e-cluster
```

### Pod 启动失败

```bash
# 查看 Pod 状态
make e2e-status

# 查看详细事件
kubectl --context=kind-usmp-e2e -n usmp-e2e describe pod <pod-name>

# 查看日志
make e2e-logs
```

### NETCONF 连接失败

```bash
# 检查模拟器 Pod 状态
kubectl --context=kind-usmp-e2e -n usmp-e2e get pods -l app=netconf-simulator

# 检查模拟器日志
make e2e-simulator-logs

# 端口转发测试
kubectl --context=kind-usmp-e2e -n usmp-e2e port-forward svc/netconf-simulator 830:830
```

### 测试超时

```bash
# 增加超时时间
# 修改测试文件中的 timeout 常量
const (
    timeout   = time.Second * 60  # 从 30 增加到 60
    interval  = time.Second * 2
)
```

## CI/CD 集成

在 GitHub Actions 或 GitLab CI 中使用：

```yaml
name: E2E Test
on: [push, pull_request]
jobs:
  e2e-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      - name: Setup Kind
        uses: helm/kind-action@v1.8.0
        with:
          cluster_name: usmp-e2e
          config: test/e2e/config/kind-cluster.yaml
      
      - name: Run E2E Tests
        run: cd backend && make test-e2e
```

## 扩展测试

### 添加新的 CRD 测试

1. 在 `test/e2e/` 目录创建 `{crd}_test.go`
2. 参考 `businessvlan_test.go` 编写测试用例
3. 运行测试验证：`go test -v -tags=e2e ./... -run YourCRD`

### 添加新的集成场景

1. 创建独立的测试文件
2. 使用 `envtest` 或真实 Kind 集群
3. 集成 NETCONF 模拟器进行真实配置下发

### 性能测试

在 E2E 环境中进行性能测试：

```bash
# 创建大量 CRD 实例测试控制器性能
for i in {1..100}; do
  kubectl apply -f - <<EOF
apiVersion: biz.usmp.io/v1
kind: BusinessVlan
metadata:
  name: perf-vlan-$i
  namespace: usmp-e2e
spec:
  vlanID: $i
  deviceID: switch-demo-01
EOF
done

# 观察控制器同步速度
kubectl --context=kind-usmp-e2e -n usmp-e2e get businessvlans -w
```

## 注意事项

1. **资源限制**：E2E 测试会消耗较多 CPU 和内存，建议在资源充足的机器上运行
2. **网络访问**：需要能够拉取 Docker 镜像（alpine, golang 等）
3. **测试时间**：完整测试运行约 5-10 分钟
4. **隔离性**：每个测试套件使用独立的命名空间，避免互相影响
5. **清理**：测试失败时可能残留资源，建议手动执行 `make e2e-clean`

## 获取帮助

```bash
# 查看可用的 make 命令
make help

# 查看测试脚本帮助
./test/e2e/run_e2e.sh --help
```
