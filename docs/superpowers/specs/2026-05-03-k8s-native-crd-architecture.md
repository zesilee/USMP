# K8s 原生 CRD 架构设计规格说明书

**日期**: 2026-05-03
**版本**: v1.0
**设计目标**: 前后端直接对接 K8s apiserver，CRD 作为唯一 Schema 源，实现声明式配置管理

---

## 一、架构总览

### 1.1 核心设计思想

**CRD = Schema = 声明式配置**

前端直接通过 Kubernetes 原生客户端与 apiserver 交互，CRD 既是数据模型也是表单 Schema。后端仅需 Controller Reconciler 处理与交换机的实际交互。

```
┌─────────────────────────────────────────────────────────────────────┐
│                        前端 (Vue3)                                    │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  DynamicForm / DynamicTable / StatusBadge                    │  │
│  │  ↑ 基于 CRD OpenAPI Schema 动态渲染                           │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                ↓ CRUD + Watch                          │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │  @kubernetes/client-node (ServiceAccount Token 认证)          │  │
│  └───────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                                    ↓ HTTPS
┌─────────────────────────────────────────────────────────────────────┐
│                    Kubernetes API Server                             │
│  ┌───────────────┐ ┌─────────────────┐ ┌─────────────────────────┐  │
│  │ BusinessVlan  │ │BusinessInterface│ │ NativeDeviceConfig      │  │
│  │ BusinessSwitch│ │ BusinessRoute   │ │ (通用原生配置)          │  │
│  │ (5个强类型)   │ │ (业务配置)      │ │ module + raw config    │  │
│  └───────────────┘ └─────────────────┘ └─────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
                                    ↓ Watch 触发
┌─────────────────────────────────────────────────────────────────────┐
│                    Controller Reconciler                            │
│  ┌─────────────────┐  ┌─────────────────────────────────────────┐  │
│  │ 业务 Controllers│  │ NativeConfig Reconciler                  │  │
│  │ (VLAN/接口/路由)│  │ (根据 module 翻译 + 下发 NETCONF)        │  │
│  └─────────────────┘  └─────────────────────────────────────────┘  │
│                                ↓ NETCONF                             │
└─────────────────────────────────────────────────────────────────────┘
                                    ↓
┌─────────────────────────────────────────────────────────────────────┐
│                       交换机设备池                                    │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.2 双轨 CRD 架构

| 类型 | CRD 数量 | Schema 来源 | 适用场景 |
|------|----------|-------------|---------|
| **业务配置** | 5 个固定 | CRD OpenAPI Schema (Go 注解生成) | VLAN、接口、路由等常用配置 |
| **原生配置** | 1 个通用 | YANG 模型文件 (后端缓存) | 厂商特定的所有原生配置模块 |

---

## 二、业务配置 CRD 设计规范

### 2.1 注解规范

通过标准 Kubebuilder 注解 + 自定义扩展注解控制前端渲染：

```go
// BusinessVlanSpec defines the desired state of BusinessVlan
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

// BusinessVlanStatus defines the observed state of BusinessVlan
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

### 2.2 自定义注解清单

| 注解 | 类型 | 作用 |
|------|------|------|
| `+custom:label="xxx"` | string | 表单字段标签 |
| `+custom:placeholder="xxx"` | string | 输入框占位符 |
| `+custom:group="xxx"` | string | 字段分组（折叠面板） |
| `+custom:readonly=true` | bool | 字段只读 |
| `+custom:hidden=true` | bool | 隐藏字段（前端不展示） |

### 2.3 标准注解映射

| Kubebuilder 注解 | 映射到前端 |
|-----------------|-----------|
| `+kubebuilder:validation:Required` | required=true |
| `+kubebuilder:validation:Enum=xxx` | enum options |
| `+kubebuilder:validation:Pattern` | pattern 校验 |
| `+kubebuilder:validation:Minimum/Maximum` | number 范围限制 |
| `+kubebuilder:default` | 默认值 |

---

## 三、原生配置 CRD 设计

### 3.1 NativeDeviceConfig CRD

```go
// NativeDeviceConfigSpec defines the desired state of NativeDeviceConfig
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
    // +custom:dynamic=true  // 标识此字段需要动态加载 Schema
    Config map[string]interface{} `json:"config"`
}

// NativeDeviceConfigStatus defines the observed state
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

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Device",type="string",JSONPath=".spec.deviceID"
// +kubebuilder:printcolumn:name="Module",type="string",JSONPath=".spec.module"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
type NativeDeviceConfig struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   NativeDeviceConfigSpec   `json:"spec,omitempty"`
    Status NativeDeviceConfigStatus `json:"status,omitempty"`
}
```

### 3.2 原生配置 Schema 加载机制

后端仅需提供一个轻量 API 用于按需加载 YANG Schema：

```go
// api/handler/yang_handler.go
func (h *YangHandler) GetModuleSchema(c *gin.Context) {
    module := c.Param("module")
    
    // 从缓存的 YANG 库中解析出字段定义
    schema, err := h.yangLibrary.GetSchema(module)
    if err != nil {
        Error(c, 404, "Module not found: "+module)
        return
    }
    
    // 转换为前端 Field 格式
    fields := convertYangSchemaToFields(schema)
    
    Success(c, gin.H{
        "module": module,
        "title": schema.Title,
        "fields": fields,
        "listFields": extractListFields(schema),
    }, "Schema retrieved")
}

// 列出所有可用的原生配置模块（侧边栏菜单用）
func (h *YangHandler) ListModules(c *gin.Context) {
    vendor := c.Query("vendor")
    modules := h.yangLibrary.ListModules(vendor)
    
    Success(c, gin.H{
        "models": modules,
    }, "Module list")
}
```

**API 路由**：
```
GET /api/yang/schema/:module  - 获取指定 YANG 模块的 Schema
GET /api/yang/models?vendor=huawei - 列出可用模块（侧边栏动态菜单）
```

---

## 四、前端架构设计

### 4.1 K8s 客户端 Composable

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
  
  // 开发环境: 从配置加载
  // 生产环境: kc.loadFromDefault() 自动加载 ServiceAccount
  if (import.meta.env.DEV) {
    kc.loadFromFile('~/.kube/config')
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
      items.value = res.body.items
      error.value = null
    } catch (e: any) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  const get = async (name: string) => {
    const res = await client.getClusterCustomObject(group, version, plural, name)
    return res.body as CRDItem
  }

  const create = async (body: Partial<CRDItem>) => {
    const res = await client.createClusterCustomObject(group, version, plural, body)
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
          if (idx >= 0) items.value[idx] = item
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
    const res = await fetch(`/apis/apiextensions.k8s.io/v1/customresourcedefinitions/${crdName}`, {
      headers: kc.getDefaultHeaders()
    })
    const crd = await res.json()
    return crd.spec.versions[0].schema.openAPIV3Schema
  }

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
    getSchema
  }
}
```

### 4.2 业务配置与原生配置统一入口

```typescript
// frontend/src/composables/useConfigPage.ts
import { computed } from 'vue'
import { useK8sCRD } from './useK8sCRD'

interface CRDInfo {
  group: string
  version: string
  plural: string
  title: string
  configType: 'business' | 'native'
}

const BUSINESS_CRDS: Record<string, CRDInfo> = {
  'vlan': { group: 'biz.usmp.io', version: 'v1', plural: 'businessvlans', title: 'VLAN 配置', configType: 'business' },
  'interface': { group: 'biz.usmp.io', version: 'v1', plural: 'businessinterfaces', title: '接口配置', configType: 'business' },
  'route': { group: 'biz.usmp.io', version: 'v1', plural: 'businessroutes', title: '路由配置', configType: 'business' },
  'switch': { group: 'biz.usmp.io', version: 'v1', plural: 'businessswitches', title: '设备管理', configType: 'business' },
}

export function useConfigPage(module: string) {
  // 原生配置使用统一的 NativeDeviceConfig CRD
  if (!BUSINESS_CRDS[module]) {
    const crd = useK8sCRD('core.usmp.io', 'v1', 'nativedeviceconfigs')
    
    // 原生配置需要动态加载 YANG Schema
    const getModuleSchema = async () => {
      const res = await fetch(`/api/yang/schema/${module}`)
      return res.json()
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
      getSchema: getModuleSchema,
      listByDevice,
      title: module,
      configType: 'native' as const
    }
  }

  // 业务配置直接使用 CRD Schema
  const info = BUSINESS_CRDS[module]
  const crd = useK8sCRD(info.group, info.version, info.plural)
  
  return {
    ...crd,
    title: info.title,
    configType: 'business' as const
  }
}
```

### 4.3 CRD Schema → 表单字段解析器

```typescript
// frontend/src/utils/crdSchemaParser.ts
import type { Field } from '../components/config/types'

interface OpenAPIProperty {
  type: string
  description?: string
  enum?: string[]
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
      options: prop.enum?.map(v => ({ label: v, value: v })),
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

---

## 五、CRD 注解到 OpenAPI 的扩展

### 5.1 修改 Controller-gen 支持自定义注解

**目标**：将 `+custom:xxx` 注解转换为 OpenAPI Schema 的 `x-` 扩展字段。

```go
// hack/custom-marker/main.go - 自定义 marker 插件
package main

import (
    "sigs.k8s.io/controller-tools/pkg/markers"
    "sigs.k8s.io/controller-tools/pkg/crd"
)

// CustomFieldAnnotation 定义自定义注解 marker
type CustomFieldAnnotation struct {
    Label       string `marker:"label"`
    Placeholder string `marker:"placeholder"`
    Group       string `marker:"group"`
    Readonly    bool   `marker:"readonly"`
    Hidden      bool   `marker:"hidden"`
}

func init() {
    // 注册自定义 marker
    markers.Must(markers.MakeDefinition("custom:label", markers.DescribesField, ""))
    markers.Must(markers.MakeDefinition("custom:placeholder", markers.DescribesField, ""))
    markers.Must(markers.MakeDefinition("custom:group", markers.DescribesField, ""))
    markers.Must(markers.MakeDefinition("custom:readonly", markers.DescribesField, false))
    markers.Must(markers.MakeDefinition("custom:hidden", markers.DescribesField, false))
}

// 生成 CRD 时注入扩展字段
func injectCustomExtensions(schema *apiext.JSONSchemaProps, fieldInfo *markers.FieldInfo) {
    // 从 marker 中提取注解并设置到 schema.Extensions
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
```

---

## 六、身份认证与权限控制

### 6.1 ServiceAccount + RBAC 配置

```yaml
# deploy/manifests/frontend-rbac.yaml
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
    verbs: ["get", "list"]

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

### 6.2 前端 Token 注入

生产环境中，ServiceAccount Token 自动挂载到 Pod：

```typescript
// 生产环境 Token 自动加载
// kc.loadFromDefault() 会自动读取:
// /var/run/secrets/kubernetes.io/serviceaccount/token
```

---

## 七、待清理的旧架构代码

### 7.1 后端需删除

```
internal/api/config_handler.go  # 旧的直接下发 API
internal/api/response.go       # 可保留通用响应
pkg/yang-runtime/manager/*     # 旧的 Actor 系统，仅保留 ClientPool
```

### 7.2 前端需删除

```
src/api/crd.ts                 # 旧的 REST API 客户端
src/stores/*.ts                # 旧的 Pinia stores（改用 useK8sCRD）
src/components/yang/*          # 旧 YANG 渲染组件
components/DynamicForm.vue     # 旧版（CRD Schema 解析后复用同一组件）
```

---

## 八、实施里程碑

| 阶段 | 任务 | 预期工作量 |
|------|------|-----------|
| 1 | CRD 注解标准化 + controller-gen 插件开发 | 1 天 |
| 2 | 前端 K8s 客户端集成 + useK8sCRD composable | 1 天 |
| 3 | CRD Schema 解析器 + 动态表单适配 | 1 天 |
| 4 | 原生配置 YANG Schema API + 动态菜单 | 0.5 天 |
| 5 | 配置页面重构 + 状态徽章集成 | 1 天 |
| 6 | RBAC 配置 + 生产环境测试 | 0.5 天 |
| 7 | 清理旧架构代码 + 测试通过 | 1 天 |
| **总计** | | **6 天** |

---

## 九、核心优势总结

✅ **真正的 K8s 原生** - 无额外 API 层，架构简洁，运维友好  
✅ **单一数据源** - CRD 是唯一 Schema，无重复定义  
✅ **声明式优先** - 前端只关心 Spec，Status 由 Controller 负责  
✅ **原生 Watch** - apiserver 的 Watch 机制比自定义 SSE 更可靠  
✅ **RBAC 复用** - 直接复用 K8s 权限体系  
✅ **动态可扩展** - 新增业务 CRD 前端零改动，新增原生配置只需更新 YANG 文件
✅ **业务/原生统一** - 两种配置类型使用完全相同的前端组件和交互体验
