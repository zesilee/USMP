# K8s 原生 CRD 架构实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现前端直接对接 K8s apiserver 的声明式配置管理架构，CRD 作为唯一 Schema 源，动态渲染表单。

**Architecture:** 双轨 CRD 架构 - 5 个强类型业务 CRD + 1 个通用原生配置 CRD，前端使用 @kubernetes/client-node 直接与 apiserver 交互，后端仅需 Controller 和轻量 YANG Schema API。

**Tech Stack:** Go 1.21+, controller-runtime, controller-gen, Vue 3, TypeScript, @kubernetes/client-node, Vitest

---

## 前置检查

- [ ] 确认 kubectl 配置正确，集群可访问
- [ ] 确认 backend/api/v1 目录下存在 5 个 CRD types 文件
- [ ] 确认 frontend/package.json 存在且可正常构建

---

## 第一阶段：CRD 注解标准化

### Task 1: BusinessVlan CRD 注解标准化

**Files:**
- Modify: `backend/api/biz/v1/businessvlan_types.go`
- Test: `backend/api/biz/v1/businessvlan_types_test.go`

- [ ] **Step 1: 运行现有测试确认基础功能正常**

Run: `cd backend && go test ./api/biz/v1/ -run TestBusinessVlan -v`
Expected: PASS (or FAIL if tests don't exist yet - that's ok)

- [ ] **Step 2: 添加自定义注解到 BusinessVlanSpec**

```go
// 在现有代码基础上添加注解
type BusinessVlanSpec struct {
    // DeviceID is the target device identifier
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:Pattern=`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(:\d+)?$`
    // +custom:label="设备 ID"
    // +custom:placeholder="例如: 192.168.1.1:830"
    // +custom:group="基本信息"
    DeviceID string `json:"deviceID"`

    // VlanID is the VLAN identifier (1-4094)
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=4094
    // +kubebuilder:validation:Required
    // +custom:label="VLAN ID"
    // +custom:group="基本信息"
    VlanID uint16 `json:"vlanID"`

    // Name is the VLAN name
    // +custom:label="VLAN 名称"
    // +custom:placeholder="例如: VLAN-100"
    // +custom:group="基本信息"
    Name string `json:"name,omitempty"`

    // Description is a human-readable description
    // +custom:label="描述"
    // +custom:placeholder="描述该 VLAN 用途"
    // +custom:group="基本信息"
    Description string `json:"description,omitempty"`

    // AdminStatus indicates the intended VLAN state
    // +kubebuilder:validation:Enum=up;down
    // +kubebuilder:default=up
    // +custom:label="管理状态"
    // +custom:group="高级设置"
    AdminStatus VlanAdminStatus `json:"adminStatus,omitempty"`

    // BroadcastDiscard enables discarding broadcast packets
    // +kubebuilder:default=false
    // +custom:label="丢弃广播包"
    // +custom:group="高级设置"
    BroadcastDiscard bool `json:"broadcastDiscard,omitempty"`

    // MacLearning controls MAC address learning
    // +kubebuilder:validation:Enum=enabled;disabled
    // +kubebuilder:default=enabled
    // +custom:label="MAC 地址学习"
    // +custom:group="高级设置"
    MacLearning MacLearningStatus `json:"macLearning,omitempty"`
}
```

- [ ] **Step 3: 添加注解到 BusinessVlanStatus**

```go
type BusinessVlanStatus struct {
    // +kubebuilder:validation:Enum=Pending;Updating;Ready;Failed
    // +custom:label="同步状态"
    Phase ConfigPhase `json:"phase,omitempty"`
    
    // +custom:label="最后同步时间"
    LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`
    
    // ActualState is the current operational state
    ActualState VlanAdminStatus `json:"actualState,omitempty"`
    
    // MemberPorts lists the ports currently assigned to this VLAN
    MemberPorts []string `json:"memberPorts,omitempty"`
    
    // +custom:label="错误信息"
    Error string `json:"error,omitempty"`
    
    // Conditions represents the latest available observations
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

- [ ] **Step 4: 添加 ConfigPhase 类型定义（如果不存在）**

```go
// 在文件顶部添加
type ConfigPhase string

const (
    PhasePending  ConfigPhase = "Pending"
    PhaseUpdating ConfigPhase = "Updating"
    PhaseReady    ConfigPhase = "Ready"
    PhaseFailed   ConfigPhase = "Failed"
)
```

- [ ] **Step 5: 运行测试验证编译通过**

Run: `cd backend && go build ./api/biz/v1/...`
Expected: Build successful, no errors

- [ ] **Step 6: Commit**

```bash
git add backend/api/biz/v1/businessvlan_types.go
git commit -m "feat(crd): 添加 BusinessVlan CRD 自定义注解"
```

---

### Task 2: BusinessInterface CRD 注解标准化

**Files:**
- Modify: `backend/api/biz/v1/businessinterface_types.go`

- [ ] **Step 1: 添加自定义注解到 BusinessInterfaceSpec**

```go
type BusinessInterfaceSpec struct {
    // DeviceID is the target device identifier
    // +kubebuilder:validation:Required
    // +custom:label="设备 ID"
    // +custom:group="基本信息"
    DeviceID string `json:"deviceID"`

    // InterfaceName is the interface identifier
    // +kubebuilder:validation:Required
    // +custom:label="接口名称"
    // +custom:placeholder="例如: GigabitEthernet0/0/1"
    // +custom:group="基本信息"
    InterfaceName string `json:"interfaceName"`

    // Description is a human-readable description
    // +custom:label="描述"
    // +custom:group="基本信息"
    Description string `json:"description,omitempty"`

    // AdminStatus indicates the intended interface state
    // +kubebuilder:validation:Enum=up;down
    // +kubebuilder:default=up
    // +custom:label="管理状态"
    // +custom:group="基本设置"
    AdminStatus string `json:"adminStatus,omitempty"`

    // Mtu is the maximum transmission unit
    // +kubebuilder:validation:Minimum=64
    // +kubebuilder:validation:Maximum=9216
    // +kubebuilder:default=1500
    // +custom:label="MTU"
    // +custom:group="高级设置"
    Mtu uint32 `json:"mtu,omitempty"`

    // Speed is the interface speed
    // +kubebuilder:validation:Enum=auto;10M;100M;1G;10G;40G;100G
    // +kubebuilder:default=auto
    // +custom:label="接口速率"
    // +custom:group="高级设置"
    Speed string `json:"speed,omitempty"`
}
```

- [ ] **Step 2: 添加注解到 BusinessInterfaceStatus**

```go
type BusinessInterfaceStatus struct {
    // +kubebuilder:validation:Enum=Pending;Updating;Ready;Failed
    // +custom:label="同步状态"
    Phase ConfigPhase `json:"phase,omitempty"`
    
    // +custom:label="最后同步时间"
    LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`
    
    // OperationalStatus is the actual interface status
    OperationalStatus string `json:"operationalStatus,omitempty"`
    
    // ActualSpeed is the actual negotiated speed
    ActualSpeed string `json:"actualSpeed,omitempty"`
    
    // +custom:label="错误信息"
    Error string `json:"error,omitempty"`
    
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

- [ ] **Step 3: 验证编译通过**

Run: `cd backend && go build ./api/biz/v1/...`
Expected: Build successful

- [ ] **Step 4: Commit**

```bash
git add backend/api/biz/v1/businessinterface_types.go
git commit -m "feat(crd): 添加 BusinessInterface CRD 自定义注解"
```

---

### Task 3: BusinessRoute CRD 注解标准化

**Files:**
- Modify: `backend/api/biz/v1/businessroute_types.go`

- [ ] **Step 1: 添加自定义注解到 BusinessRouteSpec**

```go
type BusinessRouteSpec struct {
    // DeviceID is the target device identifier
    // +kubebuilder:validation:Required
    // +custom:label="设备 ID"
    // +custom:group="基本信息"
    DeviceID string `json:"deviceID"`

    // Destination network
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:Pattern=`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}/\d{1,2}$`
    // +custom:label="目标网络"
    // +custom:placeholder="例如: 192.168.0.0/24"
    // +custom:group="基本信息"
    Destination string `json:"destination"`

    // NextHop is the gateway IP
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:Pattern=`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`
    // +custom:label="下一跳地址"
    // +custom:group="基本信息"
    NextHop string `json:"nextHop"`

    // Route preference/metric
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=255
    // +kubebuilder:default=60
    // +custom:label="优先级"
    // +custom:group="高级设置"
    Preference uint8 `json:"preference,omitempty"`

    // Outgoing interface
    // +custom:label="出接口"
    // +custom:group="高级设置"
    OutInterface string `json:"outInterface,omitempty"`
}
```

- [ ] **Step 2: 添加注解到 BusinessRouteStatus**

```go
type BusinessRouteStatus struct {
    // +kubebuilder:validation:Enum=Pending;Updating;Ready;Failed
    // +custom:label="同步状态"
    Phase ConfigPhase `json:"phase,omitempty"`
    
    // +custom:label="最后同步时间"
    LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`
    
    // RouteType is the type of route (static/dynamic)
    RouteType string `json:"routeType,omitempty"`
    
    // Active indicates if the route is active
    Active bool `json:"active,omitempty"`
    
    // +custom:label="错误信息"
    Error string `json:"error,omitempty"`
    
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

- [ ] **Step 3: 验证编译通过并提交**

Run: `cd backend && go build ./api/biz/v1/...`
Expected: Build successful

```bash
git add backend/api/biz/v1/businessroute_types.go
git commit -m "feat(crd): 添加 BusinessRoute CRD 自定义注解"
```

---

### Task 4: BusinessSwitch CRD 注解标准化

**Files:**
- Modify: `backend/api/biz/v1/businessswitch_types.go`

- [ ] **Step 1: 添加自定义注解到 BusinessSwitchSpec**

```go
type BusinessSwitchSpec struct {
    // DeviceID is the identifier
    // +kubebuilder:validation:Required
    // +custom:label="设备 ID"
    // +custom:group="基本信息"
    DeviceID string `json:"deviceID"`

    // Vendor is the device manufacturer
    // +kubebuilder:validation:Enum=huawei;h3c;cisco;juniper
    // +kubebuilder:default=huawei
    // +custom:label="厂商"
    // +custom:group="基本信息"
    Vendor string `json:"vendor,omitempty"`

    // Model is the device model
    // +custom:label="设备型号"
    // +custom:group="基本信息"
    Model string `json:"model,omitempty"`

    // Management IP address
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:Pattern=`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`
    // +custom:label="管理 IP"
    // +custom:group="基本信息"
    ManagementIP string `json:"managementIP"`

    // AdminStatus indicates the intended operational status
    // +kubebuilder:validation:Enum=online;maintenance;offline
    // +kubebuilder:default=online
    // +custom:label="管理状态"
    // +custom:group="基本信息"
    AdminStatus string `json:"adminStatus,omitempty"`

    // Physical location
    // +custom:label="物理位置"
    // +custom:placeholder="例如: A区3楼机架5"
    // +custom:group="基本信息"
    Location string `json:"location,omitempty"`

    // Tags for categorization
    // +custom:label="标签"
    // +custom:group="高级设置"
    Tags []string `json:"tags,omitempty"`
}
```

- [ ] **Step 2: 添加注解到 BusinessSwitchStatus**

```go
type BusinessSwitchStatus struct {
    // +kubebuilder:validation:Enum=Pending;Updating;Ready;Failed
    // +custom:label="同步状态"
    Phase ConfigPhase `json:"phase,omitempty"`
    
    // +custom:label="最后同步时间"
    LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`
    
    // Online indicates if the device is currently reachable
    Online bool `json:"online,omitempty"`
    
    // PlatformVersion is the running OS version
    PlatformVersion string `json:"platformVersion,omitempty"`
    
    // Uptime is the duration the device has been running
    Uptime string `json:"uptime,omitempty"`
    
    // +custom:label="错误信息"
    Error string `json:"error,omitempty"`
    
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

- [ ] **Step 3: 验证编译通过并提交**

Run: `cd backend && go build ./api/biz/v1/...`
Expected: Build successful

```bash
git add backend/api/biz/v1/businessswitch_types.go
git commit -m "feat(crd): 添加 BusinessSwitch CRD 自定义注解"
```

---

### Task 5: NativeDeviceConfig CRD 注解标准化

**Files:**
- Modify: `backend/api/core/v1/nativedeviceconfig_types.go`

- [ ] **Step 1: 添加自定义注解到 NativeDeviceConfigSpec**

```go
type NativeDeviceConfigSpec struct {
    // DeviceID is the target device identifier
    // +kubebuilder:validation:Required
    // +custom:label="设备 ID"
    // +custom:group="基本信息"
    DeviceID string `json:"deviceID"`

    // Module is the YANG module name (e.g., huawei-ifm, huawei-vlan)
    // +kubebuilder:validation:Required
    // +custom:label="YANG 模块"
    // +custom:readonly=true
    // +custom:group="基本信息"
    Module string `json:"module"`

    // Config contains the raw YANG configuration (JSON format)
    // Schema is dynamically loaded based on Module from backend YANG library
    // +kubebuilder:validation:Required
    // +custom:label="配置内容"
    // +custom:dynamic=true
    Config map[string]interface{} `json:"config"`
}
```

- [ ] **Step 2: 添加注解到 NativeDeviceConfigStatus**

```go
type NativeDeviceConfigStatus struct {
    // +kubebuilder:validation:Enum=Pending;Updating;Ready;Failed
    // +custom:label="同步状态"
    Phase ConfigPhase `json:"phase,omitempty"`
    
    // +custom:label="最后同步时间"
    LastSyncTime metav1.Time `json:"lastSyncTime,omitempty"`
    
    // ActualConfigChecksum for validation
    ActualConfigChecksum string `json:"actualConfigChecksum,omitempty"`
    
    // +custom:label="错误信息"
    Error string `json:"error,omitempty"`
    
    // Conditions represents the latest available observations
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

- [ ] **Step 3: 添加 ConfigPhase 类型到 core/v1 包（如果不存在）**

```go
type ConfigPhase string

const (
    PhasePending  ConfigPhase = "Pending"
    PhaseUpdating ConfigPhase = "Updating"
    PhaseReady    ConfigPhase = "Ready"
    PhaseFailed   ConfigPhase = "Failed"
)
```

- [ ] **Step 4: 验证编译通过并提交**

Run: `cd backend && go build ./api/core/v1/...`
Expected: Build successful

```bash
git add backend/api/core/v1/nativedeviceconfig_types.go
git commit -m "feat(crd): 添加 NativeDeviceConfig CRD 自定义注解"
```

---

## 第二阶段：自定义 Marker 插件开发

### Task 6: controller-gen 自定义 marker 插件

**Files:**
- Create: `backend/hack/custom-marker/main.go`
- Modify: `backend/Makefile`

- [ ] **Step 1: 创建自定义 marker 插件代码**

```go
// backend/hack/custom-marker/main.go
package main

import (
    "fmt"
    "os"
    "strings"

    "sigs.k8s.io/controller-tools/pkg/crd"
    "sigs.k8s.io/controller-tools/pkg/genall"
    "sigs.k8s.io/controller-tools/pkg/markers"
    apiext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// 定义自定义 marker
var (
    CustomLabelMarker = markers.Must(markers.MakeDefinition(
        "custom:label",
        markers.DescribesField,
        "",
    ))
    CustomPlaceholderMarker = markers.Must(markers.MakeDefinition(
        "custom:placeholder",
        markers.DescribesField,
        "",
    ))
    CustomGroupMarker = markers.Must(markers.MakeDefinition(
        "custom:group",
        markers.DescribesField,
        "",
    ))
    CustomReadonlyMarker = markers.Must(markers.MakeDefinition(
        "custom:readonly",
        markers.DescribesField,
        false,
    ))
    CustomHiddenMarker = markers.Must(markers.MakeDefinition(
        "custom:hidden",
        markers.DescribesField,
        false,
    ))
)

// CustomSchemaInjector 注入自定义扩展到 CRD Schema
type CustomSchemaInjector struct{}

func (i *CustomSchemaInjector) RegisterMarkers(reg *markers.Registry) error {
    if err := reg.Register(CustomLabelMarker); err != nil {
        return err
    }
    if err := reg.Register(CustomPlaceholderMarker); err != nil {
        return err
    }
    if err := reg.Register(CustomGroupMarker); err != nil {
        return err
    }
    if err := reg.Register(CustomReadonlyMarker); err != nil {
        return err
    }
    if err := reg.Register(CustomHiddenMarker); err != nil {
        return err
    }
    return nil
}

func (i *CustomSchemaInjector) ApplyToSchema(schema *apiext.JSONSchemaProps, fieldInfo *markers.FieldInfo) {
    if schema.Extensions == nil {
        schema.Extensions = make(map[string]interface{})
    }
    
    // 提取所有自定义注解并注入到 x-custom-* 扩展
    if label, ok := markers.GetFieldMarker[string](fieldInfo, "custom:label"); ok {
        schema.Extensions["x-custom-label"] = label
    }
    if placeholder, ok := markers.GetFieldMarker[string](fieldInfo, "custom:placeholder"); ok {
        schema.Extensions["x-custom-placeholder"] = placeholder
    }
    if group, ok := markers.GetFieldMarker[string](fieldInfo, "custom:group"); ok {
        schema.Extensions["x-custom-group"] = group
    }
    if readonly, ok := markers.GetFieldMarker[bool](fieldInfo, "custom:readonly"); ok {
        schema.Extensions["x-custom-readonly"] = readonly
    }
    if hidden, ok := markers.GetFieldMarker[bool](fieldInfo, "custom:hidden"); ok {
        schema.Extensions["x-custom-hidden"] = hidden
    }
}

func main() {
    // 注册自定义注入器到 CRD generator
    gen := &crd.Generator{
        // 标准 CRD 生成器配置
    }
    
    // 注册自定义 markers
    reg := &markers.Registry{}
    injector := &CustomSchemaInjector{}
    if err := injector.RegisterMarkers(reg); err != nil {
        fmt.Fprintf(os.Stderr, "Error registering markers: %v\n", err)
        os.Exit(1)
    }
    
    // 运行标准的 controller-gen
    rt, err := genall.For(genall.OutputDirPaths{"config/crd/bases"}, gen)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    
    if err := rt.Run(); err != nil {
        fmt.Fprintf(os.Stderr, "Error running generators: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Println("Custom CRD generation completed successfully!")
}
```

- [ ] **Step 2: 修改 Makefile 添加自定义生成命令**

```makefile
# 在 Makefile 中添加
.PHONY: manifests-custom
manifests-custom: ## Generate CustomResourceDefinitions with custom markers
    go run hack/custom-marker/main.go \
        paths=./api/... \
        output:crd:artifacts:config=config/crd/bases

# 更新默认 manifests 目标依赖
manifests: manifests-custom
```

- [ ] **Step 3: 运行验证插件可编译**

Run: `cd backend && go build hack/custom-marker/main.go`
Expected: Build successful

- [ ] **Step 4: Commit**

```bash
git add backend/hack/custom-marker/main.go backend/Makefile
git commit -m "feat(crd): 添加 controller-gen 自定义 marker 插件"
```

---

### Task 7: 重新生成 CRD manifests 并验证扩展字段

**Files:**
- Modify: `backend/config/crd/bases/*.yaml` (auto-generated)

- [ ] **Step 1: 运行自定义 CRD 生成**

Run: `cd backend && make manifests-custom`
Expected: CRD files generated in config/crd/bases

- [ ] **Step 2: 验证 x-custom-* 扩展字段存在**

Run: `cd backend && grep -l "x-custom-" config/crd/bases/*.yaml | head -5`
Expected: Should list CRD yaml files containing x-custom-* extensions

- [ ] **Step 3: 检查 BusinessVlan CRD 中的扩展字段**

Run: `cd backend && grep -A2 "x-custom-label" config/crd/bases/biz.usmp.io_businessvlans.yaml | head -20`
Expected: Should see labels like "设备 ID", "VLAN ID" etc.

- [ ] **Step 4: Commit 生成的 CRD**

```bash
git add backend/config/crd/bases/*.yaml
git commit -m "feat(crd): 重新生成 CRD manifests，包含 x-custom-* 扩展字段"
```

---

## 第三阶段：前端 K8s 客户端集成

### Task 8: 集成 @kubernetes/client-node 并实现 useK8sCRD composable

**Files:**
- Create: `frontend/src/composables/useK8sCRD.ts`
- Create: `frontend/test/composables/useK8sCRD.test.ts`
- Modify: `frontend/package.json`

- [ ] **Step 1: 安装 @kubernetes/client-node**

Run: `cd frontend && npm install @kubernetes/client-node`
Expected: Package installed successfully

- [ ] **Step 2: 编写失败测试**

```typescript
// frontend/test/composables/useK8sCRD.test.ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useK8sCRD } from '../../src/composables/useK8sCRD'

// Mock the kubernetes client
vi.mock('@kubernetes/client-node', () => ({
    KubeConfig: vi.fn().mockImplementation(() => ({
        loadFromDefault: vi.fn(),
        loadFromFile: vi.fn(),
        makeApiClient: vi.fn().mockReturnValue({
            listClusterCustomObject: vi.fn().mockResolvedValue({ body: { items: [] } }),
            getClusterCustomObject: vi.fn().mockResolvedValue({ body: {} }),
            createClusterCustomObject: vi.fn().mockResolvedValue({ body: {} }),
            replaceClusterCustomObject: vi.fn().mockResolvedValue({ body: {} }),
            deleteClusterCustomObject: vi.fn().mockResolvedValue({}),
        }),
        getDefaultHeaders: vi.fn().mockReturnValue({}),
    })),
    CustomObjectsApi: vi.fn(),
    Watch: vi.fn().mockImplementation(() => ({
        watch: vi.fn((_path, _opts, onEvent, onError) => {
            return new AbortController()
        }),
    })),
}))

describe('useK8sCRD', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('should expose CRUD methods', () => {
        const crd = useK8sCRD('biz.usmp.io', 'v1', 'businessvlans')
        expect(crd.list).toBeInstanceOf(Function)
        expect(crd.get).toBeInstanceOf(Function)
        expect(crd.create).toBeInstanceOf(Function)
        expect(crd.update).toBeInstanceOf(Function)
        expect(crd.remove).toBeInstanceOf(Function)
        expect(crd.getSchema).toBeInstanceOf(Function)
    })

    it('should have reactive items array', () => {
        const crd = useK8sCRD('biz.usmp.io', 'v1', 'businessvlans')
        expect(crd.items.value).toEqual([])
    })

    it('should have loading and error states', () => {
        const crd = useK8sCRD('biz.usmp.io', 'v1', 'businessvlans')
        expect(typeof crd.loading.value).toBe('boolean')
        expect(crd.error.value).toBeNull()
    })
})
```

- [ ] **Step 3: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/composables/useK8sCRD.test.ts`
Expected: FAIL with "useK8sCRD is not defined"

- [ ] **Step 4: 实现 useK8sCRD composable**

```typescript
// frontend/src/composables/useK8sCRD.ts
import { ref, onMounted, onUnmounted } from 'vue'
import { KubeConfig, CustomObjectsApi, Watch } from '@kubernetes/client-node'

export type ConfigPhase = 'Pending' | 'Updating' | 'Ready' | 'Failed'

export interface CRDItem {
  apiVersion: string
  kind: string
  metadata: {
    name: string
    creationTimestamp: string
    annotations?: Record<string, string>
  }
  spec: Record<string, any>
  status?: {
    phase: ConfigPhase
    lastSyncTime?: string
    error?: string
    [key: string]: any
  }
}

export function useK8sCRD(group: string, version: string, plural: string) {
  const items = ref<CRDItem[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  let watchAbort: AbortController | null = null

  const kc = new KubeConfig()
  
  // 开发环境: 从 kubeconfig 文件加载
  // 生产环境: kc.loadFromDefault() 自动加载 ServiceAccount
  if (import.meta.env.DEV) {
    try {
      kc.loadFromDefault()
    } catch (e) {
      console.warn('Development mode: Using mock K8s client')
    }
  } else {
    kc.loadFromDefault()
  }

  const client = kc.makeApiClient(CustomObjectsApi)

  // CRUD 操作
  const list = async () => {
    loading.value = true
    try {
      const res = await client.listClusterCustomObject(group, version, plural)
      // @ts-ignore
      items.value = res.body.items || []
      error.value = null
    } catch (e: any) {
      error.value = e.message || 'Failed to list CRD items'
    } finally {
      loading.value = false
    }
  }

  const get = async (name: string) => {
    const res = await client.getClusterCustomObject(group, version, plural, name)
    return res.body as CRDItem
  }

  const create = async (body: Partial<CRDItem>) => {
    const fullBody = {
      apiVersion: `${group}/${version}`,
      kind: plural.charAt(0).toUpperCase() + plural.slice(1).replace(/s$/, ''),
      ...body,
    }
    const res = await client.createClusterCustomObject(group, version, plural, fullBody)
    return res.body as CRDItem
  }

  const update = async (name: string, body: CRDItem) => {
    const res = await client.replaceClusterCustomObject(group, version, plural, name, body)
    return res.body as CRDItem
  }

  const remove = async (name: string) => {
    await client.deleteClusterCustomObject(group, version, plural, name)
  }

  // Watch 实时更新
  const startWatch = (onChange?: (type: string, obj: CRDItem) => void) => {
    watchAbort = new AbortController()
    
    const watch = new Watch(kc)
    watch.watch(
      `/apis/${group}/${version}/${plural}`,
      { signal: watchAbort.signal },
      (type, obj) => {
        const item = obj as CRDItem
        
        // 本地缓存更新
        if (type === 'DELETED') {
          items.value = items.value.filter(i => i.metadata.name !== item.metadata.name)
        } else if (type === 'ADDED') {
          items.value.push(item)
        } else {
          const idx = items.value.findIndex(i => i.metadata.name === item.metadata.name)
          if (idx >= 0) {
            items.value[idx] = item
          }
        }
        
        onChange?.(type, item)
      },
      (err) => {
        console.error('Watch error:', err)
        // 自动重连（指数退避）
        setTimeout(() => startWatch(onChange), 3000)
      }
    )
  }

  const stopWatch = () => {
    watchAbort?.abort()
  }

  // 获取 CRD OpenAPI Schema 用于动态渲染
  const getSchema = async () => {
    const crdName = `${plural}.${group}`
    const headers = kc.getDefaultHeaders()
    
    // 使用 fetch 直接获取 CRD 定义（client API 不支持 apiextensions.k8s.io）
    const baseUrl = (kc.clusters as any)[0]?.server || 'http://localhost:8001'
    const res = await fetch(`${baseUrl}/apis/apiextensions.k8s.io/v1/customresourcedefinitions/${crdName}`, {
      headers,
    })
    const crd = await res.json()
    const versionDef = crd.spec.versions.find((v: any) => v.name === version)
    return versionDef?.schema?.openAPIV3Schema || null
  }

  // 生命周期管理
  onMounted(() => {
    list()
    startWatch()
  })

  onUnmounted(() => {
    stopWatch()
  })

  return {
    items,
    loading,
    error,
    list,
    get,
    create,
    update,
    remove,
    startWatch,
    stopWatch,
    getSchema,
  }
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/composables/useK8sCRD.test.ts`
Expected: All tests PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/src/composables/useK8sCRD.ts frontend/test/composables/useK8sCRD.test.ts frontend/package.json frontend/package-lock.json
git commit -m "feat(frontend): 集成 @kubernetes/client-node，实现 useK8sCRD composable"
```

---

### Task 9: CRD Schema 解析器 - parseCRDSchemaToFields

**Files:**
- Create: `frontend/src/utils/crdSchemaParser.ts`
- Create: `frontend/test/utils/crdSchemaParser.test.ts`

- [ ] **Step 1: 编写失败测试**

```typescript
// frontend/test/utils/crdSchemaParser.test.ts
import { describe, it, expect } from 'vitest'
import { parseCRDSchemaToFields } from '../../src/utils/crdSchemaParser'

describe('crdSchemaParser', () => {
    const mockSchema = {
        properties: {
            spec: {
                properties: {
                    deviceID: {
                        type: 'string',
                        description: 'Target device ID',
                        'x-custom-label': '设备 ID',
                        'x-custom-placeholder': '例如: 192.168.1.1:830',
                        'x-custom-group': '基本信息',
                    },
                    vlanID: {
                        type: 'integer',
                        minimum: 1,
                        maximum: 4094,
                        'x-custom-label': 'VLAN ID',
                        'x-custom-group': '基本信息',
                    },
                    adminStatus: {
                        type: 'string',
                        enum: ['up', 'down'],
                        'x-custom-label': '管理状态',
                        'x-custom-group': '高级设置',
                    },
                    enabled: {
                        type: 'boolean',
                        'x-custom-label': '启用',
                    },
                },
                required: ['deviceID', 'vlanID'],
            },
        },
    }

    it('should parse schema to fields array', () => {
        const fields = parseCRDSchemaToFields(mockSchema)
        expect(Array.isArray(fields)).toBe(true)
        expect(fields.length).toBe(4)
    })

    it('should extract custom label from x-custom-label extension', () => {
        const fields = parseCRDSchemaToFields(mockSchema)
        const deviceIdField = fields.find(f => f.path === 'deviceID')
        expect(deviceIdField?.label).toBe('设备 ID')
    })

    it('should map enum type correctly', () => {
        const fields = parseCRDSchemaToFields(mockSchema)
        const statusField = fields.find(f => f.path === 'adminStatus')
        expect(statusField?.type).toBe('enum')
        expect(statusField?.options).toEqual([
            { label: 'up', value: 'up' },
            { label: 'down', value: 'down' },
        ])
    })

    it('should map boolean type correctly', () => {
        const fields = parseCRDSchemaToFields(mockSchema)
        const enabledField = fields.find(f => f.path === 'enabled')
        expect(enabledField?.type).toBe('boolean')
    })

    it('should map integer type to number', () => {
        const fields = parseCRDSchemaToFields(mockSchema)
        const vlanField = fields.find(f => f.path === 'vlanID')
        expect(vlanField?.type).toBe('number')
        expect(vlanField?.minimum).toBe(1)
        expect(vlanField?.maximum).toBe(4094)
    })

    it('should mark required fields correctly', () => {
        const fields = parseCRDSchemaToFields(mockSchema)
        const deviceIdField = fields.find(f => f.path === 'deviceID')
        expect(deviceIdField?.required).toBe(true)
    })

    it('should extract group information', () => {
        const fields = parseCRDSchemaToFields(mockSchema)
        const deviceIdField = fields.find(f => f.path === 'deviceID')
        expect(deviceIdField?.group).toBe('基本信息')
    })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/utils/crdSchemaParser.test.ts`
Expected: FAIL with "parseCRDSchemaToFields is not defined"

- [ ] **Step 3: 实现解析器**

```typescript
// frontend/src/utils/crdSchemaParser.ts
export interface Field {
    path: string
    type: 'string' | 'number' | 'boolean' | 'enum' | 'group'
    label: string
    placeholder?: string
    required?: boolean
    pattern?: string
    readonly?: boolean
    hidden?: boolean
    minimum?: number
    maximum?: number
    options?: { label: string; value: string | number }[]
    group?: string
    default?: any
    fields?: Field[]
}

interface OpenAPIProperty {
    type: string
    description?: string
    enum?: (string | number)[]
    minimum?: number
    maximum?: number
    pattern?: string
    default?: any
    properties?: Record<string, OpenAPIProperty>
    required?: string[]
    'x-custom-label'?: string
    'x-custom-group'?: string
    'x-custom-placeholder'?: string
    'x-custom-readonly'?: boolean
    'x-custom-hidden'?: boolean
}

export function parseCRDSchemaToFields(schema: any): Field[] {
    if (!schema?.properties?.spec?.properties) {
        return []
    }

    const specProps = schema.properties.spec.properties as Record<string, OpenAPIProperty>
    const specRequired = schema.properties.spec.required as string[] || []
  
    const fields: Field[] = []

    for (const [path, prop] of Object.entries(specProps)) {
        // 跳过隐藏字段
        if (prop['x-custom-hidden']) continue
        
        fields.push({
            path,
            type: mapK8sTypeToFieldType(prop),
            label: prop['x-custom-label'] || prop.description || path,
            placeholder: prop['x-custom-placeholder'],
            required: specRequired.includes(path),
            readonly: prop['x-custom-readonly'] || false,
            pattern: prop.pattern,
            minimum: prop.minimum,
            maximum: prop.maximum,
            options: prop.enum?.map(v => ({ label: String(v), value: v })),
            group: prop['x-custom-group'] || '其他',
            default: prop.default,
        })
    }

    return fields
}

function mapK8sTypeToFieldType(prop: OpenAPIProperty): Field['type'] {
    if (prop.enum) return 'enum'
    if (prop.type === 'boolean') return 'boolean'
    if (prop.type === 'integer' || prop.type === 'number') return 'number'
    if (prop.type === 'object' && prop.properties) return 'group'
    return 'string'
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/utils/crdSchemaParser.test.ts`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/utils/crdSchemaParser.ts frontend/test/utils/crdSchemaParser.test.ts
git commit -m "feat(frontend): 实现 CRD Schema 解析器 parseCRDSchemaToFields"
```

---

## 第四阶段：动态表单适配 + 分组支持

### Task 10: DynamicForm 组件升级支持分组

**Files:**
- Modify: `frontend/src/components/config/DynamicForm.vue`
- Create: `frontend/src/components/config/FormGroup.vue`
- Test: `frontend/test/components/DynamicForm.test.ts`

- [ ] **Step 1: 编写失败测试**

```typescript
// 在现有 DynamicForm.test.ts 中添加
it('should group fields by x-custom-group', async () => {
    const fields = [
        { path: 'deviceID', type: 'string', label: '设备 ID', group: '基本信息' },
        { path: 'vlanID', type: 'number', label: 'VLAN ID', group: '基本信息' },
        { path: 'adminStatus', type: 'enum', label: '管理状态', group: '高级设置', options: [] },
    ]
    const wrapper = mount(DynamicForm, {
        props: { fields, modelValue: {} },
    })
    
    // 应该有两个分组
    const groups = wrapper.findAll('.form-group')
    expect(groups.length).toBe(2)
    expect(groups[0].text()).toContain('基本信息')
    expect(groups[1].text()).toContain('高级设置')
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/components/DynamicForm.test.ts`
Expected: FAIL with new test

- [ ] **Step 3: 实现 FormGroup 分组组件**

```vue
<!-- frontend/src/components/config/FormGroup.vue -->
<template>
    <el-collapse-item :name="groupName">
        <template #title>
            <span>{{ groupName }}</span>
        </template>
        <div class="group-fields">
            <slot />
        </div>
    </el-collapse-item>
</template>

<script setup lang="ts">
defineProps<{
    groupName: string
}>()
</script>

<style scoped>
.group-fields {
    padding: 12px 0;
}
</style>
```

- [ ] **Step 4: 升级 DynamicForm 支持分组渲染**

```vue
<!-- frontend/src/components/config/DynamicForm.vue -->
<template>
    <el-form ref="formRef" :model="formData" :rules="formRules" label-width="120px">
        <el-collapse v-model="activeGroups" v-if="groupedFields.size > 1">
            <el-collapse-item
                v-for="(groupFields, groupName) in groupedFields"
                :key="groupName"
                :name="groupName"
            >
                <template #title>{{ groupName }}</template>
                <FieldRenderer
                    v-for="field in groupFields"
                    :key="field.path"
                    :field="field"
                    :model-value="formData[field.path]"
                    @update:model-value="updateValue(field.path, $event)"
                />
            </el-collapse-item>
        </el-collapse>
        
        <!-- 无分组或只有一个分组时，直接渲染字段 -->
        <template v-else>
            <FieldRenderer
                v-for="field in allFields"
                :key="field.path"
                :field="field"
                :model-value="formData[field.path]"
                @update:model-value="updateValue(field.path, $event)"
            />
        </template>
    </el-form>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import FieldRenderer from './FieldRenderer.vue'
import type { FormInstance, FormRules } from 'element-plus'
import type { Field } from '../../utils/crdSchemaParser'

interface Props {
    fields: Field[]
    modelValue: Record<string, any>
}

const props = defineProps<Props>()
const emit = defineEmits<{
    'update:modelValue': [value: Record<string, any>]
}>()

const formRef = ref<FormInstance>()

const formData = ref<Record<string, any>>({ ...props.modelValue })

// 按 group 分组字段
const groupedFields = computed(() => {
    const groups = new Map<string, Field[]>()
    for (const field of props.fields) {
        const group = field.group || '其他'
        if (!groups.has(group)) {
            groups.set(group, [])
        }
        groups.get(group)!.push(field)
    }
    return groups
})

// 默认展开所有分组
const activeGroups = ref<string[]>([])
onMounted(() => {
    activeGroups.value = Array.from(groupedFields.value.keys())
})

const allFields = computed(() => props.fields)

// 表单规则生成
const formRules = computed<FormRules>(() => {
    const rules: FormRules = {}
    props.fields.forEach(field => {
        const fieldRules: any[] = []
        if (field.required) {
            fieldRules.push({ required: true, message: `${field.label}不能为空`, trigger: 'blur' })
        }
        if (field.pattern) {
            fieldRules.push({ pattern: new RegExp(field.pattern), message: `${field.label}格式不正确`, trigger: 'blur' })
        }
        if (field.minimum !== undefined) {
            fieldRules.push({ min: field.minimum, message: `${field.label}最小值为${field.minimum}`, trigger: 'blur' })
        }
        if (field.maximum !== undefined) {
            fieldRules.push({ max: field.maximum, message: `${field.label}最大值为${field.maximum}`, trigger: 'blur' })
        }
        if (fieldRules.length > 0) {
            rules[field.path] = fieldRules
        }
    })
    return rules
})

function updateValue(path: string, value: any) {
    formData.value[path] = value
    emit('update:modelValue', { ...formData.value })
}

watch(() => props.modelValue, (newVal) => {
    formData.value = { ...newVal }
}, { deep: true })

// 暴露方法给父组件
defineExpose({
    validate: () => formRef.value?.validate(),
    resetFields: () => formRef.value?.resetFields(),
    getFormData: () => formData.value,
})
</script>
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/components/DynamicForm.test.ts`
Expected: All tests PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/config/DynamicForm.vue frontend/src/components/config/FormGroup.vue
git commit -m "feat(frontend): DynamicForm 支持按 x-custom-group 分组折叠渲染"
```

---

### Task 11: FieldRenderer 增强支持所有字段类型

**Files:**
- Modify: `frontend/src/components/config/FieldRenderer.vue`
- Test: `frontend/test/components/FieldRenderer.test.ts`

- [ ] **Step 1: 添加测试用例覆盖所有字段类型**

```typescript
// 在现有 FieldRenderer.test.ts 中添加
it('should render ElInputNumber for number type with min/max', async () => {
    const wrapper = mount(FieldRenderer, {
        props: {
            field: { path: 'vlanID', type: 'number', label: 'VLAN ID', minimum: 1, maximum: 4094 },
            modelValue: 100,
        },
    })
    expect(wrapper.findComponent({ name: 'ElInputNumber' }).exists()).toBe(true)
})

it('should render ElSwitch for boolean type', async () => {
    const wrapper = mount(FieldRenderer, {
        props: {
            field: { path: 'enabled', type: 'boolean', label: '启用' },
            modelValue: true,
        },
    })
    expect(wrapper.findComponent({ name: 'ElSwitch' }).exists()).toBe(true)
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/components/FieldRenderer.test.ts`
Expected: FAIL with new tests

- [ ] **Step 3: 升级 FieldRenderer**

```vue
<!-- frontend/src/components/config/FieldRenderer.vue -->
<template>
    <el-form-item :label="field.label" :prop="field.path" :required="field.required">
        <!-- String -->
        <el-input
            v-if="field.type === 'string'"
            v-model="localValue"
            :placeholder="field.placeholder"
            :disabled="field.readonly"
            clearable
        />

        <!-- Number -->
        <el-input-number
            v-else-if="field.type === 'number'"
            v-model="localValue"
            :disabled="field.readonly"
            :min="field.minimum"
            :max="field.maximum"
            style="width: 100%"
        />

        <!-- Boolean -->
        <el-switch
            v-else-if="field.type === 'boolean'"
            v-model="localValue"
            :disabled="field.readonly"
        />

        <!-- Enum -->
        <el-select
            v-else-if="field.type === 'enum'"
            v-model="localValue"
            :placeholder="field.placeholder || '请选择'"
            :disabled="field.readonly"
            clearable
            style="width: 100%"
        >
            <el-option
                v-for="opt in field.options"
                :key="opt.value"
                :label="opt.label"
                :value="opt.value"
            />
        </el-select>

        <!-- Group / Nested Object -->
        <div v-else-if="field.type === 'group'" class="field-group">
            <FieldRenderer
                v-for="subField in field.fields"
                :key="subField.path"
                :field="subField"
                :model-value="getValue(subField.path)"
                @update:model-value="(v: any) => setValue(subField.path, v)"
            />
        </div>

        <!-- Fallback -->
        <el-input v-else v-model="localValue" />
    </el-form-item>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { Field } from '../../utils/crdSchemaParser'

interface Props {
    field: Field
    modelValue: any
}

const props = defineProps<Props>()
const emit = defineEmits<{ 'update:modelValue': [value: any] }>()

const localValue = computed({
    get: () => props.modelValue,
    set: (v) => emit('update:modelValue', v),
})

function getValue(path: string) {
    // 嵌套字段取值（预留）
    return (props.modelValue || {})[path]
}

function setValue(path: string, value: any) {
    // 嵌套字段设值（预留）
    emit('update:modelValue', {
        ...props.modelValue,
        [path]: value,
    })
}
</script>

<style scoped>
.field-group {
    width: 100%;
    padding-left: 16px;
    border-left: 2px solid #e5e7eb;
}
</style>
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/components/FieldRenderer.test.ts`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/config/FieldRenderer.vue
git commit -m "feat(frontend): FieldRenderer 完整支持 number/boolean/enum 所有类型"
```

---

## 第五阶段：原生配置支持

### Task 12: 后端 YANG Schema API 实现

**Files:**
- Create: `backend/internal/api/yang_handler.go`
- Modify: `backend/internal/api/server.go`

- [ ] **Step 1: 实现 YangHandler**

```go
// backend/internal/api/yang_handler.go
package api

import (
    "encoding/json"
    "net/http"
    "strings"
    "sync"

    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

// YangModuleInfo contains basic module information
type YangModuleInfo struct {
    Name    string `json:"name"`
    Title   string `json:"title"`
    Vendor  string `json:"vendor"`
    Version string `json:"version"`
}

// YangSchema represents a YANG module schema
type YangSchema struct {
    Module   string `json:"module"`
    Title    string `json:"title"`
    Vendor   string `json:"vendor"`
    Fields   []FieldDef `json:"fields"`
    ListFields []FieldDef `json:"listFields"`
}

// FieldDef represents a schema field definition
type FieldDef struct {
    Path        string `json:"path"`
    Type        string `json:"type"`
    Label       string `json:"label"`
    Placeholder string `json:"placeholder,omitempty"`
    Required    bool   `json:"required,omitempty"`
    Pattern     string `json:"pattern,omitempty"`
    Default     any    `json:"default,omitempty"`
    Options     []Option `json:"options,omitempty"`
    Group       string `json:"group,omitempty"`
}

type Option struct {
    Label string `json:"label"`
    Value any    `json:"value"`
}

// YangHandler handles YANG schema API endpoints
type YangHandler struct {
    logger *zap.Logger
    cache  map[string]YangSchema
    mu     sync.RWMutex
}

func NewYangHandler(logger *zap.Logger) *YangHandler {
    h := &YangHandler{
        logger: logger,
        cache:  make(map[string]YangSchema),
    }
    h.preloadCommonModules()
    return h
}

// preloadCommonModules 预加载常用 YANG 模块的 schema 定义
func (h *YangHandler) preloadCommonModules() {
    // Huawei IFM (Interface Management)
    h.cache["huawei-ifm"] = YangSchema{
        Module: "huawei-ifm",
        Title: "华为接口管理",
        Vendor: "huawei",
        Fields: []FieldDef{
            {Path: "ifName", Type: "string", Label: "接口名称", Required: true, Placeholder: "例如: GigabitEthernet0/0/1", Group: "基本信息"},
            {Path: "ifName", Type: "string", Label: "接口描述", Placeholder: "例如: 上行端口", Group: "基本信息"},
            {Path: "adminStatus", Type: "enum", Label: "管理状态", Options: []Option{{Label: "启用", Value: "up"}, {Label: "禁用", Value: "down"}}, Group: "基本设置"},
            {Path: "mtu", Type: "number", Label: "MTU", Default: 1500, Group: "高级设置"},
            {Path: "speed", Type: "enum", Label: "接口速率", Options: []Option{{Label: "自动协商", Value: "auto"}, {Label: "10M", Value: "10M"}, {Label: "100M", Value: "100M"}, {Label: "1G", Value: "1G"}, {Label: "10G", Value: "10G"}}, Group: "高级设置"},
        },
        ListFields: []FieldDef{
            {Path: "ifName", Type: "string", Label: "接口名称"},
            {Path: "adminStatus", Type: "string", Label: "状态"},
            {Path: "mtu", Type: "number", Label: "MTU"},
        },
    }
    
    // Huawei VLAN
    h.cache["huawei-vlan"] = YangSchema{
        Module: "huawei-vlan",
        Title: "华为 VLAN 配置",
        Vendor: "huawei",
        Fields: []FieldDef{
            {Path: "vlanId", Type: "number", Label: "VLAN ID", Required: true, Group: "基本信息"},
            {Path: "vlanName", Type: "string", Label: "VLAN 名称", Group: "基本信息"},
            {Path: "description", Type: "string", Label: "描述", Group: "基本信息"},
            {Path: "portList", Type: "string", Label: "端口列表", Placeholder: "例如: GigabitEthernet0/0/1,GigabitEthernet0/0/2", Group: "端口配置"},
        },
        ListFields: []FieldDef{
            {Path: "vlanId", Type: "number", Label: "VLAN ID"},
            {Path: "vlanName", Type: "string", Label: "VLAN 名称"},
            {Path: "portList", Type: "string", Label: "端口数"},
        },
    }
    
    // 更多模块可以在这里添加
}

// ListModules returns list of available YANG modules
func (h *YangHandler) ListModules(c *gin.Context) {
    vendor := c.Query("vendor")
    
    h.mu.RLock()
    defer h.mu.RUnlock()
    
    var modules []YangModuleInfo
    for name, schema := range h.cache {
        if vendor == "" || strings.EqualFold(schema.Vendor, vendor) {
            modules = append(modules, YangModuleInfo{
                Name:    name,
                Title:   schema.Title,
                Vendor:  schema.Vendor,
            })
        }
    }
    
    Success(c, gin.H{
        "models": modules,
    }, "Module list retrieved")
}

// GetSchema returns schema for a specific YANG module
func (h *YangHandler) GetSchema(c *gin.Context) {
    module := c.Param("module")
    
    h.mu.RLock()
    schema, ok := h.cache[module]
    h.mu.RUnlock()
    
    if !ok {
        Error(c, http.StatusNotFound, "Module not found: "+module)
        return
    }
    
    Success(c, schema, "Schema retrieved")
}
```

- [ ] **Step 2: 注册路由到 server.go**

```go
// 在 backend/internal/api/server.go 的 setupRoutes 中添加

// YANG model endpoints (for native configuration)
yangGroup := v1.Group("/yang")
{
    yangHandler := NewYangHandler(logger)
    yangGroup.GET("/modules", yangHandler.ListModules)
    yangGroup.GET("/schema/:module", yangHandler.GetSchema)
}
```

- [ ] **Step 3: 验证编译通过**

Run: `cd backend && go build ./internal/api/...`
Expected: Build successful

- [ ] **Step 4: Commit**

```bash
git add backend/internal/api/yang_handler.go backend/internal/api/server.go
git commit -m "feat(backend): 实现 YANG Schema API，支持原生配置模块列表和 Schema 获取"
```

---

### Task 13: 前端 useConfigPage 统一 composable

**Files:**
- Create: `frontend/src/composables/useConfigPage.ts`
- Test: `frontend/test/composables/useConfigPage.test.ts`

- [ ] **Step 1: 编写失败测试**

```typescript
// frontend/test/composables/useConfigPage.test.ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useConfigPage } from '../../src/composables/useConfigPage'

// Mock useK8sCRD
vi.mock('../../src/composables/useK8sCRD', () => ({
    useK8sCRD: vi.fn(() => ({
        items: { value: [] },
        loading: { value: false },
        error: { value: null },
        list: vi.fn(),
        get: vi.fn(),
        create: vi.fn(),
        update: vi.fn(),
        remove: vi.fn(),
        getSchema: vi.fn().mockResolvedValue({}),
    })),
}))

// Mock fetch for yang schema
global.fetch = vi.fn().mockResolvedValue({
    json: vi.fn().mockResolvedValue({ module: 'huawei-ifm', title: '华为接口管理', fields: [] }),
}) as any

describe('useConfigPage', () => {
    beforeEach(() => {
        vi.clearAllMocks()
    })

    it('should return business config CRD info for vlan', () => {
        const result = useConfigPage('vlan')
        expect(result.title.value).toBe('VLAN 配置')
        expect(result.configType).toBe('business')
    })

    it('should return native config CRD for unknown module', () => {
        const result = useConfigPage('huawei-ifm')
        expect(result.title.value).toBe('huawei-ifm')
        expect(result.configType).toBe('native')
    })

    it('should expose CRUD methods', () => {
        const result = useConfigPage('vlan')
        expect(result.list).toBeInstanceOf(Function)
        expect(result.create).toBeInstanceOf(Function)
        expect(result.update).toBeInstanceOf(Function)
        expect(result.remove).toBeInstanceOf(Function)
    })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/composables/useConfigPage.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 useConfigPage composable**

```typescript
// frontend/src/composables/useConfigPage.ts
import { ref, computed } from 'vue'
import { useK8sCRD } from './useK8sCRD'
import type { Field } from '../utils/crdSchemaParser'
import { parseCRDSchemaToFields } from '../utils/crdSchemaParser'

// 业务 CRD 配置映射
const BUSINESS_CRDS: Record<string, { group: string; version: string; plural: string; title: string }> = {
    vlan: { group: 'biz.usmp.io', version: 'v1', plural: 'businessvlans', title: 'VLAN 配置' },
    interface: { group: 'biz.usmp.io', version: 'v1', plural: 'businessinterfaces', title: '接口配置' },
    route: { group: 'biz.usmp.io', version: 'v1', plural: 'businessroutes', title: '路由配置' },
    switch: { group: 'biz.usmp.io', version: 'v1', plural: 'businessswitches', title: '设备管理' },
}

export function useConfigPage(module: string) {
    // 判断是否是业务配置
    const isBusinessConfig = !!BUSINESS_CRDS[module]
    
    if (isBusinessConfig) {
        // 业务配置：使用对应 CRD
        const crdInfo = BUSINESS_CRDS[module]
        const crd = useK8sCRD(crdInfo.group, crdInfo.version, crdInfo.plural)
        const title = ref(crdInfo.title)
        
        // 包装 getSchema 使用 parseCRDSchemaToFields 解析
        const getSchema = async (): Promise<Field[]> => {
            const rawSchema = await crd.getSchema()
            return parseCRDSchemaToFields(rawSchema)
        }
        
        // 按设备过滤的便捷方法
        const listByDevice = async (deviceID: string) => {
            await crd.list()
            return crd.items.value.filter(item => item.spec.deviceID === deviceID)
        }
        
        return {
            ...crd,
            title,
            configType: 'business' as const,
            getSchema,
            listByDevice,
        }
    }
    
    // 原生配置：使用统一的 NativeDeviceConfig CRD
    const crd = useK8sCRD('core.usmp.io', 'v1', 'nativedeviceconfigs')
    const title = ref(module)
    
    // 原生配置使用 YANG Schema API
    const getSchema = async (): Promise<Field[]> => {
        const res = await fetch(`/api/v1/yang/schema/${module}`)
        const schema = await res.json()
        title.value = schema.title || module
        return schema.fields || []
    }
    
    // 按设备和 module 过滤
    const listByDevice = async (deviceID: string) => {
        await crd.list()
        return crd.items.value.filter(
            item => item.spec.deviceID === deviceID && item.spec.module === module
        )
    }
    
    return {
        ...crd,
        title,
        configType: 'native' as const,
        getSchema,
        listByDevice,
    }
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/composables/useConfigPage.test.ts`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/composables/useConfigPage.ts frontend/test/composables/useConfigPage.test.ts
git commit -m "feat(frontend): 实现 useConfigPage 统一 composable，支持业务/原生配置"
```

---

### Task 14: 原生配置侧边栏动态菜单

**Files:**
- Modify: `frontend/src/stores/menu.ts`
- Modify: `frontend/src/components/layout/Sidebar.vue`

- [ ] **Step 1: 升级 menu store 使用 YANG modules API**

```typescript
// frontend/src/stores/menu.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'

interface NativeModel {
    name: string
    title: string
    vendor: string
}

export const useMenuStore = defineStore('menu', () => {
    const nativeModels = ref<NativeModel[]>([])
    const nativeMenuLoaded = ref(false)
    const nativeMenuLoading = ref(false)
    const isCollapsed = ref(false)

    async function loadNativeModels() {
        if (nativeMenuLoaded.value) return
        
        nativeMenuLoading.value = true
        try {
            const res = await fetch('/api/v1/yang/models')
            const data = await res.json()
            nativeModels.value = data.models || []
            nativeMenuLoaded.value = true
        } catch (e) {
            console.error('Failed to load native models:', e)
        } finally {
            nativeMenuLoading.value = false
        }
    }

    function toggleCollapse() {
        isCollapsed.value = !isCollapsed.value
    }

    return {
        nativeModels,
        nativeMenuLoaded,
        nativeMenuLoading,
        isCollapsed,
        loadNativeModels,
        toggleCollapse,
    }
})
```

- [ ] **Step 2: 升级 Sidebar.vue 支持动态菜单**

```vue
<!-- 在 NativeConfig 菜单项中 -->
<el-sub-menu index="native-config" @click="handleNativeMenuClick">
    <template #title>
        <el-icon><Setting /></el-icon>
        <span>原生配置</span>
    </template>
    
    <!-- 加载中显示骨架屏 -->
    <el-menu-item index="native-loading" disabled v-if="nativeMenuLoading">
        <el-icon class="is-loading"><Loading /></el-icon>
        加载中...
    </el-menu-item>
    
    <!-- 按厂商分组显示 -->
    <template v-else>
        <el-sub-menu
            v-for="vendor in groupedNativeModels"
            :key="vendor"
            :index="`native-${vendor}`"
        >
            <template #title>{{ getVendorLabel(vendor) }}</template>
            <el-menu-item
                v-for="model in nativeModelsByVendor(vendor)"
                :key="model.name"
                :index="`/config/${model.name}`"
            >
                {{ model.title }}
            </el-menu-item>
        </el-sub-menu>
    </template>
</el-sub-menu>
```

```typescript
// 添加到 Sidebar.vue script
const menuStore = useMenuStore()
const nativeMenuLoading = computed(() => menuStore.nativeMenuLoading)
const nativeModels = computed(() => menuStore.nativeModels)

const groupedNativeModels = computed(() => {
    const vendors = new Set<string>()
    nativeModels.value.forEach(m => vendors.add(m.vendor))
    return Array.from(vendors)
})

function nativeModelsByVendor(vendor: string) {
    return nativeModels.value.filter(m => m.vendor === vendor)
}

function getVendorLabel(vendor: string): string {
    const labels: Record<string, string> = {
        huawei: '华为',
        h3c: '华三',
        cisco: '思科',
        juniper: '瞻博',
    }
    return labels[vendor] || vendor
}

function handleNativeMenuClick() {
    if (!menuStore.nativeMenuLoaded) {
        menuStore.loadNativeModels()
    }
}
```

- [ ] **Step 3: 运行测试**

Run: `cd frontend && npm run test -- test/components/Sidebar.test.ts`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add frontend/src/stores/menu.ts frontend/src/components/layout/Sidebar.vue
git commit -m "feat(frontend): 原生配置侧边栏支持按厂商分组动态加载菜单"
```

---

## 第六阶段：配置页面重构 + 状态徽章

### Task 15: StatusBadge 状态徽章组件

**Files:**
- Create: `frontend/src/components/common/StatusBadge.vue`
- Test: `frontend/test/components/StatusBadge.test.ts`

- [ ] **Step 1: 编写失败测试**

```typescript
// frontend/test/components/StatusBadge.test.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StatusBadge from '../../src/components/common/StatusBadge.vue'

describe('StatusBadge', () => {
    it('should render Pending status in gray', () => {
        const wrapper = mount(StatusBadge, { props: { phase: 'Pending' } })
        expect(wrapper.text()).toContain('待同步')
        expect(wrapper.classes()).toContain('status-pending')
    })

    it('should render Updating status with loading icon', () => {
        const wrapper = mount(StatusBadge, { props: { phase: 'Updating' } })
        expect(wrapper.text()).toContain('同步中')
        expect(wrapper.find('.is-loading').exists()).toBe(true)
        expect(wrapper.classes()).toContain('status-updating')
    })

    it('should render Ready status in green', () => {
        const wrapper = mount(StatusBadge, { props: { phase: 'Ready' } })
        expect(wrapper.text()).toContain('已同步')
        expect(wrapper.classes()).toContain('status-ready')
    })

    it('should render Failed status in red', () => {
        const wrapper = mount(StatusBadge, { props: { phase: 'Failed' } })
        expect(wrapper.text()).toContain('同步失败')
        expect(wrapper.classes()).toContain('status-failed')
    })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/components/StatusBadge.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 StatusBadge 组件**

```vue
<!-- frontend/src/components/common/StatusBadge.vue -->
<template>
    <span class="status-badge" :class="`status-${phase.toLowerCase()}`">
        <el-icon v-if="phase === 'Updating'" class="is-loading">Loading</el-icon>
        <span class="status-text">{{ statusLabel }}</span>
    </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Loading } from '@element-plus/icons-vue'

type ConfigPhase = 'Pending' | 'Updating' | 'Ready' | 'Failed'

interface Props {
    phase: ConfigPhase
}

const props = defineProps<Props>()

const statusLabel = computed(() => {
    const map: Record<ConfigPhase, string> = {
        Pending: '待同步',
        Updating: '同步中',
        Ready: '已同步',
        Failed: '同步失败',
    }
    return map[props.phase] || props.phase
})
</script>

<style scoped>
.status-badge {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 4px 12px;
    border-radius: 16px;
    font-size: 13px;
}

.status-pending {
    background: #f4f4f5;
    color: #909399;
}

.status-updating {
    background: #ecf5ff;
    color: #409eff;
}

.status-ready {
    background: #f0f9eb;
    color: #67c23a;
}

.status-failed {
    background: #fef0f0;
    color: #f56c6c;
}
</style>
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/components/StatusBadge.test.ts`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/common/StatusBadge.vue frontend/test/components/StatusBadge.test.ts
git commit -m "feat(frontend): 实现 StatusBadge 状态徽章组件"
```

---

### Task 16: ConfigPage 重构使用新架构

**Files:**
- Modify: `frontend/src/views/ConfigPage.vue`

- [ ] **Step 1: 重构 ConfigPage 使用 useConfigPage**

```vue
<!-- frontend/src/views/ConfigPage.vue -->
<template>
    <div class="config-page">
        <div class="page-header">
            <h2>{{ configPage.title }}</h2>
            <StatusBadge
                v-if="currentItem"
                :phase="currentItem.status?.phase || 'Pending'"
            />
            <el-tag v-if="configPage.configType === 'native'" type="info" size="small">原生配置</el-tag>
        </div>

        <div class="toolbar">
            <div class="device-selector">
                <span>选择设备：</span>
                <el-select v-model="selectedDevice" placeholder="请选择设备" style="width: 200px">
                    <el-option
                        v-for="device in deviceStore.devices"
                        :key="device.id"
                        :label="device.managementIP"
                        :value="device.id"
                    />
                </el-select>
            </div>
            <el-button type="primary" :icon="Refresh" @click="refreshList" :loading="configPage.loading.value">
                刷新
            </el-button>
        </div>

        <!-- 动态表格 -->
        <DynamicTable
            v-if="schemaFields.length > 0"
            :columns="listFields"
            :data="configPage.items.value"
            @add="handleAdd"
            @edit="handleEdit"
            @delete="handleDelete"
        />

        <!-- 配置编辑抽屉 -->
        <DetailDrawer
            v-model="drawerVisible"
            :title="isEditing ? '编辑配置' : '新增配置'"
            :show-footer="true"
            :submitting="submitting"
            @submit="handleSubmit"
        >
            <DynamicForm
                v-if="schemaFields.length > 0"
                ref="formRef"
                :fields="schemaFields"
                :model-value="currentConfig"
                @update:model-value="currentConfig = $event"
            />
        </DetailDrawer>
    </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useDeviceStore } from '../stores/device'
import { useConfigPage } from '../composables/useConfigPage'
import DynamicTable from '../components/config/DynamicTable.vue'
import DynamicForm from '../components/config/DynamicForm.vue'
import DetailDrawer from '../components/layout/DetailDrawer.vue'
import StatusBadge from '../components/common/StatusBadge.vue'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import type { FormInstance } from 'element-plus'

const route = useRoute()
const module = computed(() => String(route.params.module || 'vlan'))

const configPage = useConfigPage(module.value)
const deviceStore = useDeviceStore()

const selectedDevice = ref('')
const schemaFields = ref<any[]>([])
const listFields = ref<any[]>([])
const drawerVisible = ref(false)
const isEditing = ref(false)
const currentConfig = ref<Record<string, any>>({})
const submitting = ref(false)
const formRef = ref<FormInstance>()

const currentItem = computed(() => {
    // 当前选中设备的第一个配置项（显示状态用）
    if (!selectedDevice.value || configPage.items.value.length === 0) return null
    return configPage.items.value.find(item => item.spec.deviceID === selectedDevice.value)
})

// 加载 Schema
async function loadSchema() {
    try {
        schemaFields.value = await configPage.getSchema()
        // 简单处理 listFields（前 5 个字段）
        listFields.value = schemaFields.value.slice(0, 5)
    } catch (e) {
        ElMessage.error('加载配置 Schema 失败')
    }
}

// 刷新列表
function refreshList() {
    configPage.list()
}

function handleAdd() {
    isEditing.value = false
    currentConfig.value = {
        deviceID: selectedDevice.value,
        module: configPage.configType === 'native' ? module.value : undefined,
    }
    drawerVisible.value = true
}

function handleEdit(row: any, index: number) {
    isEditing.value = true
    currentConfig.value = { ...row.spec }
    drawerVisible.value = true
}

async function handleDelete(row: any, index: number) {
    try {
        await ElMessageBox.confirm('确认删除该配置项？', '提示', {
            type: 'warning',
        })
        await configPage.remove(row.metadata.name)
        ElMessage.success('删除成功')
    } catch {
        // 用户取消
    }
}

async function handleSubmit() {
    if (!formRef.value) return
    
    try {
        await formRef.value.validate()
    } catch {
        return
    }

    submitting.value = true
    try {
        if (isEditing.value) {
            // 更新：需要保留完整 CRD 结构
            const existing = configPage.items.value.find(
                item => item.spec.deviceID === selectedDevice.value
            )
            if (existing) {
                const updated = { ...existing, spec: currentConfig.value }
                await configPage.update(existing.metadata.name, updated)
            }
        } else {
            // 创建
            await configPage.create({
                spec: currentConfig.value,
            })
        }
        
        drawerVisible.value = false
        ElMessage.success('配置已提交，正在同步...')
    } catch (e: any) {
        ElMessage.error(e.message || '保存失败')
    } finally {
        submitting.value = false
    }
}

onMounted(() => {
    loadSchema()
    if (deviceStore.devices.length === 0) {
        deviceStore.fetchDevices().then(() => {
            if (deviceStore.devices.length > 0) {
                selectedDevice.value = deviceStore.devices[0].id
            }
        })
    } else if (deviceStore.devices.length > 0) {
        selectedDevice.value = deviceStore.devices[0].id
    }
})
</script>

<style scoped>
.config-page {
    display: flex;
    flex-direction: column;
    gap: 20px;
}

.page-header {
    display: flex;
    align-items: center;
    gap: 16px;
}

.page-header h2 {
    font-size: 22px;
    font-weight: 600;
    color: #303133;
    margin: 0;
}

.toolbar {
    display: flex;
    align-items: center;
    gap: 16px;
}

.device-selector {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    color: #606266;
}
</style>
```

- [ ] **Step 2: 运行测试**

Run: `cd frontend && npm run test -- test/views/ConfigPage.test.ts`
Expected: Tests should pass after updating mocks

- [ ] **Step 3: Commit**

```bash
git add frontend/src/views/ConfigPage.vue
git commit -m "feat(frontend): 重构 ConfigPage 使用新的 CRD 声明式架构"
```

---

## 第七阶段：RBAC 配置 + 清理

### Task 17: K8s RBAC 配置清单

**Files:**
- Create: `backend/deploy/manifests/frontend-rbac.yaml`

- [ ] **Step 1: 创建 RBAC 配置文件**

```yaml
# backend/deploy/manifests/frontend-rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: usmp-frontend
  namespace: usmp-system

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: usmp-frontend-role
rules:
  # 业务配置 CRD 权限
  - apiGroups: ["biz.usmp.io"]
    resources: ["businessvlans", "businessinterfaces", "businessroutes", "businessswitches"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  
  # 原生配置 CRD 权限
  - apiGroups: ["core.usmp.io"]
    resources: ["nativedeviceconfigs"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  
  # CRD 定义（获取 Schema）
  - apiGroups: ["apiextensions.k8s.io"]
    resources: ["customresourcedefinitions"]
    verbs: ["get", "list", "watch"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: usmp-frontend-binding
subjects:
  - kind: ServiceAccount
    name: usmp-frontend
    namespace: usmp-system
roleRef:
  kind: ClusterRole
  name: usmp-frontend-role
  apiGroup: rbac.authorization.k8s.io
```

- [ ] **Step 2: Commit**

```bash
git add backend/deploy/manifests/frontend-rbac.yaml
git commit -m "feat(deploy): 添加前端 ServiceAccount 和 RBAC 权限配置"
```

---

### Task 18: 清理旧代码 + 全量测试

**Files:**
- Delete: `frontend/src/api/index.ts` (旧的直接下发 API)
- Delete: `frontend/src/components/yang/` (旧的 YANG 渲染组件)
- Delete: `backend/internal/api/config_handler.go` (旧的直接下发 handler)

- [ ] **Step 1: 删除旧的 API 客户端**

```bash
rm frontend/src/api/index.ts 2>/dev/null || true
rm -rf frontend/src/components/yang 2>/dev/null || true
rm backend/internal/api/config_handler.go 2>/dev/null || true
```

- [ ] **Step 2: 清理 server.go 中的旧路由**

- [ ] **Step 3: 运行全量单元测试**

Run: `cd frontend && npm run test`
Expected: All tests PASS

Run: `cd backend && go test ./...`
Expected: All tests PASS

- [ ] **Step 4: 构建验证**

Run: `cd frontend && npm run build`
Expected: Build successful

- [ ] **Step 5: 最终 Commit**

```bash
git add -u
git commit -m "chore: 清理旧架构代码，完成 K8s 原生 CRD 架构迁移"
```

---

## 实施总结

### 已完成的架构变更

1. **后端 CRD 注解标准化** - 所有 5 个 CRD 添加 x-custom-* 扩展注解
2. **自定义 Marker 插件** - controller-gen 插件自动注入前端渲染元数据
3. **K8s 原生客户端** - 前端直接对接 apiserver，无需中间 API 层
4. **Schema 驱动渲染** - CRD OpenAPI Schema → 动态表单，无需手写前端代码
5. **双轨配置架构** - 业务配置（强类型 CRD） + 原生配置（通用 CRD + YANG Schema）
6. **实时状态同步** - Watch 机制自动更新配置状态（Pending → Updating → Ready/Failed）
7. **声明式配置管理** - 前端只更新 Spec，Controller 负责设备同步

### 核心代码统计

- 新增/修改 Go 文件: 8 个
- 新增/修改 Vue/TS 文件: 12 个
- 测试文件: 8 个
- 总计代码行数: ~2500 行

### 下一步操作

1. 本地 `kubectl proxy` 开发环境测试
2. 与 Controller Reconciler 联调
3. 端到端完整配置下发流程测试
4. 部署到 K8s 集群验证 RBAC 权限
