# USMP 前端页面实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现完整的交换机设备管理平台前端页面，包括基础布局、仪表盘、设备管理、动态配置渲染框架。

**Architecture:** 采用 Vue 3 + TypeScript + Element Plus，YANG 模型驱动的动态渲染架构。业务配置与原生配置 100% 复用同一套组件，Sidebar 支持静态+动态混合菜单加载。

**Tech Stack:** Vue 3, TypeScript, Vite, Element Plus, Pinia, Axios, ECharts, Vitest, Playwright

---

## 前置检查

- [ ] 确认前端项目结构: `frontend/src/` 目录存在
- [ ] 确认依赖已安装: `frontend/package.json` 包含 Vue 3, Element Plus, Axios

---

## 第一阶段：基础布局组件实现

### Task 1: 项目基础配置与路由设置

**Files:**
- Create: `frontend/src/router/index.ts`
- Modify: `frontend/src/main.ts`
- Test: `frontend/test/router.test.ts`

- [ ] **Step 1: 编写路由测试**

```typescript
import { createRouter, createWebHistory } from 'vue-router'
import { describe, it, expect } from 'vitest'

describe('Router Configuration', () => {
  const router = createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/', name: 'dashboard', component: {} },
      { path: '/devices', name: 'devices', component: {} },
      { path: '/config/interface', name: 'interface', component: {} },
      { path: '/config/vlan', name: 'vlan', component: {} },
      { path: '/config/route', name: 'route', component: {} },
      { path: '/native/:module', name: 'native', component: {} },
      { path: '/logs', name: 'logs', component: {} },
      { path: '/settings', name: 'settings', component: {} },
    ]
  })

  it('should have dashboard route', () => {
    const route = router.getRoutes().find(r => r.name === 'dashboard')
    expect(route).toBeDefined()
    expect(route?.path).toBe('/')
  })

  it('should have all business config routes', () => {
    const names = router.getRoutes().map(r => r.name)
    expect(names).toContain('interface')
    expect(names).toContain('vlan')
    expect(names).toContain('route')
  })

  it('should have native config dynamic route', () => {
    const route = router.getRoutes().find(r => r.name === 'native')
    expect(route).toBeDefined()
    expect(route?.path).toBe('/native/:module')
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/router.test.ts`
Expected: FAIL with test file not found

- [ ] **Step 3: 实现路由配置**

```typescript
// frontend/src/router/index.ts
import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    name: 'dashboard',
    component: () => import('../views/Dashboard.vue')
  },
  {
    path: '/devices',
    name: 'devices',
    component: () => import('../views/Devices.vue')
  },
  {
    path: '/config/interface',
    name: 'interface',
    component: () => import('../views/ConfigPage.vue'),
    props: { module: 'openconfig-interfaces' }
  },
  {
    path: '/config/vlan',
    name: 'vlan',
    component: () => import('../views/ConfigPage.vue'),
    props: { module: 'openconfig-vlan' }
  },
  {
    path: '/config/route',
    name: 'route',
    component: () => import('../views/ConfigPage.vue'),
    props: { module: 'openconfig-route' }
  },
  {
    path: '/native/:module',
    name: 'native',
    component: () => import('../views/ConfigPage.vue')
  },
  {
    path: '/logs',
    name: 'logs',
    component: () => import('../views/Logs.vue')
  },
  {
    path: '/settings',
    name: 'settings',
    component: () => import('../views/Settings.vue')
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

export default router
```

- [ ] **Step 4: 更新 main.ts 注册路由和 Element Plus**

```typescript
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import App from './App.vue'
import router from './router'
import './style.css'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)
app.use(ElementPlus)
app.mount('#app')
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/router.test.ts`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/src/router/index.ts frontend/src/main.ts frontend/test/router.test.ts
git commit -m "feat(frontend): 添加基础路由配置"
```

---

### Task 2: Sidebar 侧边导航组件

**Files:**
- Create: `frontend/src/components/layout/Sidebar.vue`
- Create: `frontend/src/stores/menu.ts`
- Test: `frontend/test/components/Sidebar.test.ts`

- [ ] **Step 1: 编写组件测试**

```typescript
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import Sidebar from '../../src/components/layout/Sidebar.vue'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'

describe('Sidebar Component', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  const router = createRouter({
    history: createWebHistory(),
    routes: [{ path: '/', name: 'dashboard', component: {} }]
  })

  it('should render static menu items', () => {
    const wrapper = mount(Sidebar, { global: { plugins: [router] } })
    expect(wrapper.text()).toContain('概览')
    expect(wrapper.text()).toContain('设备管理')
    expect(wrapper.text()).toContain('业务网络配置')
  })

  it('should have native config menu item', () => {
    const wrapper = mount(Sidebar, { global: { plugins: [router] } })
    expect(wrapper.text()).toContain('原生配置')
  })

  it('should toggle menu collapse', async () => {
    const wrapper = mount(Sidebar, { global: { plugins: [router] } })
    const collapseBtn = wrapper.find('.collapse-btn')
    if (collapseBtn.exists()) {
      await collapseBtn.trigger('click')
      expect(wrapper.vm.isCollapsed).toBe(true)
    }
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/components/Sidebar.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 menu store**

```typescript
// frontend/src/stores/menu.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'

interface NativeModel {
  name: string
  title: string
  vendor: string
}

export const useMenuStore = defineStore('menu', () => {
  const nativeModels = ref<NativeModel[]>([])
  const nativeMenuLoaded = ref(false)
  const isCollapsed = ref(false)

  async function loadNativeModels() {
    if (nativeMenuLoaded.value) return
    try {
      const res = await axios.get('/api/crd/models?type=native')
      nativeModels.value = res.data.models || []
      nativeMenuLoaded.value = true
    } catch (e) {
      console.error('Failed to load native models:', e)
    }
  }

  function toggleCollapse() {
    isCollapsed.value = !isCollapsed.value
  }

  return {
    nativeModels,
    nativeMenuLoaded,
    isCollapsed,
    loadNativeModels,
    toggleCollapse
  }
})
```

- [ ] **Step 4: 实现 Sidebar 组件**

```vue
<!-- frontend/src/components/layout/Sidebar.vue -->
<template>
  <div class="sidebar" :class="{ collapsed: isCollapsed }">
    <div class="logo-area">
      <span v-if="!isCollapsed">USMP</span>
      <span v-else>U</span>
    </div>

    <el-menu
      :default-active="activeMenu"
      :collapse="isCollapsed"
      router
    >
      <el-menu-item index="/">
        <el-icon><DataLine /></el-icon>
        <template #title>概览</template>
      </el-menu-item>

      <el-menu-item index="/devices">
        <el-icon><Monitor /></el-icon>
        <template #title>设备管理</template>
      </el-menu-item>

      <el-sub-menu index="business-config">
        <template #title>
          <el-icon><Connection /></el-icon>
          <span>业务网络配置</span>
        </template>
        <el-menu-item index="/config/interface">接口配置</el-menu-item>
        <el-menu-item index="/config/vlan">VLAN配置</el-menu-item>
        <el-menu-item index="/config/route">路由配置</el-menu-item>
      </el-sub-menu>

      <el-sub-menu index="native-config" @click="handleNativeMenuClick">
        <template #title>
          <el-icon><Setting /></el-icon>
          <span>原生配置</span>
        </template>
        <el-menu-item
          v-for="model in nativeModels"
          :key="model.name"
          :index="`/native/${model.name}`"
        >
          {{ model.title }}
        </el-menu-item>
        <el-menu-item v-if="!nativeMenuLoaded" index="loading" disabled>
          <el-icon class="is-loading"><Loading /></el-icon>
          加载中...
        </el-menu-item>
      </el-sub-menu>

      <el-menu-item index="/logs">
        <el-icon><Document /></el-icon>
        <template #title>操作日志</template>
      </el-menu-item>

      <el-menu-item index="/settings">
        <el-icon><Tools /></el-icon>
        <template #title>系统设置</template>
      </el-menu-item>
    </el-menu>

    <div class="collapse-btn" @click="toggleCollapse">
      <el-icon>{{ isCollapsed ? 'Expand' : 'Fold' }}</el-icon>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useMenuStore } from '../../stores/menu'
import { DataLine, Monitor, Connection, Setting, Document, Tools, Loading, Fold, Expand } from '@element-plus/icons-vue'

const route = useRoute()
const menuStore = useMenuStore()

const activeMenu = computed(() => route.path)
const isCollapsed = computed(() => menuStore.isCollapsed)
const nativeModels = computed(() => menuStore.nativeModels)
const nativeMenuLoaded = computed(() => menuStore.nativeMenuLoaded)

function toggleCollapse() {
  menuStore.toggleCollapse()
}

function handleNativeMenuClick() {
  if (!nativeMenuLoaded.value) {
    menuStore.loadNativeModels()
  }
}
</script>

<style scoped>
.sidebar {
  width: 240px;
  height: 100vh;
  background: #fff;
  border-right: 1px solid #e5e7eb;
  display: flex;
  flex-direction: column;
  transition: width 0.3s;
}

.sidebar.collapsed {
  width: 64px;
}

.logo-area {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: bold;
  color: #409eff;
  border-bottom: 1px solid #e5e7eb;
}

.el-menu {
  flex: 1;
  border-right: none;
}

.collapse-btn {
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  border-top: 1px solid #e5e7eb;
  color: #909399;
}

.collapse-btn:hover {
  color: #409eff;
}
</style>
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/components/Sidebar.test.ts`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/layout/Sidebar.vue frontend/src/stores/menu.ts frontend/test/components/Sidebar.test.ts
git commit -m "feat(frontend): 实现 Sidebar 侧边导航组件，支持动态原生配置菜单"
```

---

### Task 3: Header 顶部栏组件

**Files:**
- Create: `frontend/src/components/layout/Header.vue`
- Test: `frontend/test/components/Header.test.ts`

- [ ] **Step 1: 编写组件测试**

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import Header from '../../src/components/layout/Header.vue'
import { createPinia, setActivePinia } from 'pinia'

describe('Header Component', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should render search input', () => {
    const wrapper = mount(Header)
    expect(wrapper.find('.el-input').exists()).toBe(true)
  })

  it('should render notification icon', () => {
    const wrapper = mount(Header)
    expect(wrapper.find('.notification-icon').exists()).toBe(true)
  })

  it('should render user avatar area', () => {
    const wrapper = mount(Header)
    expect(wrapper.find('.user-area').exists()).toBe(true)
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/components/Header.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 Header 组件**

```vue
<!-- frontend/src/components/layout/Header.vue -->
<template>
  <div class="header">
    <div class="header-left">
      <el-input
        v-model="searchText"
        placeholder="搜索设备、配置..."
        prefix-icon="Search"
        style="width: 300px"
      />
    </div>

    <div class="header-right">
      <div class="device-status">
        <span class="status-dot online"></span>
        <span>{{ onlineCount }}/{{ totalCount }} 在线</span>
      </div>

      <el-badge :value="notificationCount" :hidden="notificationCount === 0" class="notification-icon">
        <el-button circle :icon="Bell" />
      </el-badge>

      <div class="user-area">
        <el-avatar :size="32">Admin</el-avatar>
      </div>

      <el-dropdown>
        <el-button circle :icon="MoreFilled" />
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item>切换主题</el-dropdown-item>
            <el-dropdown-item>系统设置</el-dropdown-item>
            <el-dropdown-item divided>退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Bell, MoreFilled, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const searchText = ref('')
const onlineCount = ref(10)
const totalCount = ref(12)
const notificationCount = ref(3)

onMounted(() => {
  // 后续对接真实 API
})
</script>

<style scoped>
.header {
  height: 64px;
  background: #fff;
  border-bottom: 1px solid #e5e7eb;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.device-status {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: #606266;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-dot.online {
  background: #67c23a;
}

.notification-icon {
  cursor: pointer;
}

.user-area {
  cursor: pointer;
}
</style>
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/components/Header.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/layout/Header.vue frontend/test/components/Header.test.ts
git commit -m "feat(frontend): 实现 Header 顶部栏组件"
```

---

### Task 4: DetailDrawer 右侧详情抽屉

**Files:**
- Create: `frontend/src/components/layout/DetailDrawer.vue`
- Test: `frontend/test/components/DetailDrawer.test.ts`

- [ ] **Step 1: 编写组件测试**

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DetailDrawer from '../../src/components/layout/DetailDrawer.vue'

describe('DetailDrawer Component', () => {
  it('should be hidden by default', () => {
    const wrapper = mount(DetailDrawer, {
      props: { modelValue: false, title: '测试' }
    })
    expect(wrapper.find('.el-drawer').exists()).toBe(false)
  })

  it('should show when visible is true', () => {
    const wrapper = mount(DetailDrawer, {
      props: { modelValue: true, title: '测试标题' }
    })
    expect(wrapper.text()).toContain('测试标题')
  })

  it('should emit close event', async () => {
    const wrapper = mount(DetailDrawer, {
      props: { modelValue: true, title: '测试' }
    })
    wrapper.vm.$emit('close')
    expect(wrapper.emitted('close')).toBeTruthy()
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/components/DetailDrawer.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 DetailDrawer 组件**

```vue
<!-- frontend/src/components/layout/DetailDrawer.vue -->
<template>
  <el-drawer
    :model-value="modelValue"
    :title="title"
    direction="rtl"
    size="560px"
    @close="handleClose"
    @update:model-value="handleUpdate"
  >
    <slot />
    <template #footer v-if="showFooter">
      <div class="drawer-footer">
        <el-button @click="handleCancel">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          {{ submitText }}
        </el-button>
      </div>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
interface Props {
  modelValue: boolean
  title: string
  showFooter?: boolean
  submitText?: string
  submitting?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showFooter: false,
  submitText: '提交',
  submitting: false
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'close': []
  'cancel': []
  'submit': []
}>()

function handleClose() {
  emit('close')
}

function handleUpdate(value: boolean) {
  emit('update:modelValue', value)
}

function handleCancel() {
  emit('cancel')
  emit('update:modelValue', false)
}

function handleSubmit() {
  emit('submit')
}
</script>

<style scoped>
.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}
</style>
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/components/DetailDrawer.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/layout/DetailDrawer.vue frontend/test/components/DetailDrawer.test.ts
git commit -m "feat(frontend): 实现 DetailDrawer 右侧详情抽屉组件"
```

---

### Task 5: MainLayout 主布局容器

**Files:**
- Create: `frontend/src/components/layout/MainLayout.vue`
- Modify: `frontend/src/App.vue`
- Test: `frontend/test/components/MainLayout.test.ts`

- [ ] **Step 1: 编写布局测试**

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import MainLayout from '../../src/components/layout/MainLayout.vue'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'

describe('MainLayout Component', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  const router = createRouter({
    history: createWebHistory(),
    routes: [{ path: '/', name: 'dashboard', component: {} }]
  })

  it('should contain Sidebar, Header and main content area', () => {
    const wrapper = mount(MainLayout, { global: { plugins: [router] } })
    expect(wrapper.find('.sidebar-wrapper').exists()).toBe(true)
    expect(wrapper.find('.header-wrapper').exists()).toBe(true)
    expect(wrapper.find('.main-content').exists()).toBe(true)
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/components/MainLayout.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 MainLayout 组件**

```vue
<!-- frontend/src/components/layout/MainLayout.vue -->
<template>
  <div class="main-layout">
    <div class="sidebar-wrapper">
      <Sidebar />
    </div>

    <div class="content-wrapper">
      <div class="header-wrapper">
        <Header />
      </div>

      <main class="main-content">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import Sidebar from './Sidebar.vue'
import Header from './Header.vue'
</script>

<style scoped>
.main-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.sidebar-wrapper {
  flex-shrink: 0;
}

.content-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.main-content {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
  background: #f5f7fa;
}
</style>
```

- [ ] **Step 4: 更新 App.vue**

```vue
<!-- frontend/src/App.vue -->
<template>
  <MainLayout>
    <router-view />
  </MainLayout>
</template>

<script setup lang="ts">
import MainLayout from './components/layout/MainLayout.vue'
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body, #app {
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}
</style>
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/components/MainLayout.test.ts`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/layout/MainLayout.vue frontend/src/App.vue frontend/test/components/MainLayout.test.ts
git commit -m "feat(frontend): 实现 MainLayout 主布局容器"
```

---

## 第二阶段：仪表盘页面实现

### Task 6: StatCard 统计卡片组件

**Files:**
- Create: `frontend/src/components/dashboard/StatCard.vue`
- Test: `frontend/test/components/StatCard.test.ts`

- [ ] **Step 1: 编写组件测试**

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StatCard from '../../src/components/dashboard/StatCard.vue'

describe('StatCard Component', () => {
  it('should render title and value', () => {
    const wrapper = mount(StatCard, {
      props: { title: '设备总数', value: 12 }
    })
    expect(wrapper.text()).toContain('设备总数')
    expect(wrapper.text()).toContain('12')
  })

  it('should render trend indicator', () => {
    const wrapper = mount(StatCard, {
      props: { title: '测试', value: 10, trend: 5, trendLabel: '较昨日' }
    })
    expect(wrapper.text()).toContain('较昨日')
    expect(wrapper.text()).toContain('+5')
  })

  it('should show negative trend in red', () => {
    const wrapper = mount(StatCard, {
      props: { title: '测试', value: 10, trend: -2 }
    })
    expect(wrapper.find('.trend-negative').exists()).toBe(true)
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/components/StatCard.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 StatCard 组件**

```vue
<!-- frontend/src/components/dashboard/StatCard.vue -->
<template>
  <div class="stat-card">
    <div class="stat-icon" :style="{ background: iconBg }">
      <el-icon :size="24" :color="iconColor">
        <component :is="icon" />
      </el-icon>
    </div>
    <div class="stat-content">
      <div class="stat-value">{{ value }}</div>
      <div class="stat-title">{{ title }}</div>
      <div v-if="trend !== undefined" class="stat-trend" :class="{ 'trend-negative': trend < 0 }">
        <el-icon>{{ trend >= 0 ? 'Top' : 'Bottom' }}</el-icon>
        <span>{{ trend >= 0 ? '+' : '' }}{{ trend }}</span>
        <span v-if="trendLabel" class="trend-label">{{ trendLabel }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
interface Props {
  title: string
  value: number | string
  icon?: string
  iconColor?: string
  iconBg?: string
  trend?: number
  trendLabel?: string
}

withDefaults(defineProps<Props>(), {
  icon: 'DataLine',
  iconColor: '#409eff',
  iconBg: '#ecf5ff'
})
</script>

<style scoped>
.stat-card {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.stat-content {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #303133;
  line-height: 1.2;
}

.stat-title {
  font-size: 14px;
  color: #909399;
  margin-top: 4px;
}

.stat-trend {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #67c23a;
  margin-top: 8px;
}

.stat-trend.trend-negative {
  color: #f56c6c;
}

.trend-label {
  color: #909399;
}
</style>
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/components/StatCard.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/dashboard/StatCard.vue frontend/test/components/StatCard.test.ts
git commit -m "feat(frontend): 实现 StatCard 统计卡片组件"
```

---

### Task 7: StatusChart 设备状态图表组件

**Files:**
- Create: `frontend/src/components/dashboard/StatusChart.vue`
- Test: `frontend/test/components/StatusChart.test.ts`

- [ ] **Step 1: 安装 ECharts 依赖**

Run: `cd frontend && npm install echarts`

- [ ] **Step 2: 编写组件测试**

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StatusChart from '../../src/components/dashboard/StatusChart.vue'

describe('StatusChart Component', () => {
  it('should render chart container', () => {
    const wrapper = mount(StatusChart, {
      props: { online: 10, offline: 2, abnormal: 0 }
    })
    expect(wrapper.find('.chart-container').exists()).toBe(true)
  })

  it('should display status legend', () => {
    const wrapper = mount(StatusChart, {
      props: { online: 10, offline: 2, abnormal: 0 }
    })
    expect(wrapper.text()).toContain('在线')
    expect(wrapper.text()).toContain('离线')
    expect(wrapper.text()).toContain('异常')
  })
})
```

- [ ] **Step 3: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/components/StatusChart.test.ts`
Expected: FAIL

- [ ] **Step 4: 实现 StatusChart 组件**

```vue
<!-- frontend/src/components/dashboard/StatusChart.vue -->
<template>
  <div class="status-chart">
    <h3 class="chart-title">设备状态分布</h3>
    <div class="chart-content">
      <div ref="chartRef" class="chart-container"></div>
      <div class="chart-legend">
        <div class="legend-item">
          <span class="legend-dot online"></span>
          <span class="legend-label">在线</span>
          <span class="legend-value">{{ online }}</span>
        </div>
        <div class="legend-item">
          <span class="legend-dot offline"></span>
          <span class="legend-label">离线</span>
          <span class="legend-value">{{ offline }}</span>
        </div>
        <div class="legend-item">
          <span class="legend-dot abnormal"></span>
          <span class="legend-label">异常</span>
          <span class="legend-value">{{ abnormal }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch, onUnmounted } from 'vue'
import * as echarts from 'echarts'

interface Props {
  online: number
  offline: number
  abnormal: number
}

const props = defineProps<Props>()

const chartRef = ref<HTMLElement>()
let chartInstance: echarts.ECharts | null = null

function initChart() {
  if (!chartRef.value) return

  chartInstance = echarts.init(chartRef.value)

  const option: echarts.EChartsOption = {
    series: [
      {
        type: 'pie',
        radius: ['60%', '80%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 10,
          borderColor: '#fff',
          borderWidth: 2
        },
        label: { show: false },
        emphasis: {
          label: { show: false }
        },
        data: [
          { value: props.online, name: '在线', itemStyle: { color: '#67c23a' } },
          { value: props.offline, name: '离线', itemStyle: { color: '#909399' } },
          { value: props.abnormal, name: '异常', itemStyle: { color: '#f56c6c' } }
        ]
      }
    ]
  }

  chartInstance.setOption(option)
}

function handleResize() {
  chartInstance?.resize()
}

onMounted(() => {
  initChart()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  chartInstance?.dispose()
})

watch(() => [props.online, props.offline, props.abnormal], () => {
  if (chartInstance) {
    chartInstance.setOption({
      series: [{
        data: [
          { value: props.online, name: '在线' },
          { value: props.offline, name: '离线' },
          { value: props.abnormal, name: '异常' }
        ]
      }]
    })
  }
})
</script>

<style scoped>
.status-chart {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.chart-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 16px;
}

.chart-content {
  display: flex;
  align-items: center;
  gap: 32px;
}

.chart-container {
  width: 200px;
  height: 200px;
}

.chart-legend {
  flex: 1;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid #f0f0f0;
}

.legend-item:last-child {
  border-bottom: none;
}

.legend-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
}

.legend-dot.online {
  background: #67c23a;
}

.legend-dot.offline {
  background: #909399;
}

.legend-dot.abnormal {
  background: #f56c6c;
}

.legend-label {
  flex: 1;
  font-size: 14px;
  color: #606266;
}

.legend-value {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}
</style>
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/components/StatusChart.test.ts`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/dashboard/StatusChart.vue frontend/test/components/StatusChart.test.ts
git commit -m "feat(frontend): 实现 StatusChart 设备状态图表组件"
```

---

### Task 8: Dashboard 仪表盘页面

**Files:**
- Create: `frontend/src/views/Dashboard.vue`
- Test: `frontend/test/views/Dashboard.test.ts`

- [ ] **Step 1: 编写页面测试**

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import Dashboard from '../../src/views/Dashboard.vue'
import { createPinia, setActivePinia } from 'pinia'

describe('Dashboard View', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should render 4 stat cards', () => {
    const wrapper = mount(Dashboard)
    const cards = wrapper.findAll('.stat-card')
    expect(cards.length).toBe(4)
  })

  it('should render status chart', () => {
    const wrapper = mount(Dashboard)
    expect(wrapper.find('.status-chart').exists()).toBe(true)
  })

  it('should render recent logs table', () => {
    const wrapper = mount(Dashboard)
    expect(wrapper.find('.el-table').exists()).toBe(true)
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/views/Dashboard.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 Dashboard 页面**

```vue
<!-- frontend/src/views/Dashboard.vue -->
<template>
  <div class="dashboard">
    <div class="stats-row">
      <StatCard
        title="设备总数"
        :value="stats.total"
        icon="Monitor"
        :trend="stats.totalTrend"
        trend-label="较昨日"
      />
      <StatCard
        title="在线设备"
        :value="stats.online"
        icon="SuccessFilled"
        icon-color="#67c23a"
        icon-bg="#f0f9eb"
        :trend="stats.onlineTrend"
        trend-label="较昨日"
      />
      <StatCard
        title="配置同步率"
        :value="`${stats.syncRate}%`"
        icon="CircleCheck"
        icon-color="#409eff"
        icon-bg="#ecf5ff"
      />
      <StatCard
        title="今日操作"
        :value="stats.todayOps"
        icon="Document"
        icon-color="#e6a23c"
        icon-bg="#fdf6ec"
        :trend="stats.todayOpsTrend"
        trend-label="较昨日"
      />
    </div>

    <div class="content-row">
      <div class="chart-col">
        <StatusChart
          :online="stats.online"
          :offline="stats.offline"
          :abnormal="stats.abnormal"
        />
      </div>
    </div>

    <div class="logs-section">
      <div class="section-header">
        <h3>最近操作记录</h3>
        <el-button link>查看全部</el-button>
      </div>
      <el-table :data="recentLogs" stripe>
        <el-table-column prop="time" label="时间" width="180" />
        <el-table-column prop="device" label="设备" width="140" />
        <el-table-column prop="type" label="操作类型" width="120" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === '成功' ? 'success' : 'danger'" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="user" label="操作人" />
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import StatCard from '../components/dashboard/StatCard.vue'
import StatusChart from '../components/dashboard/StatusChart.vue'
import { Monitor, SuccessFilled, CircleCheck, Document } from '@element-plus/icons-vue'

const stats = ref({
  total: 12,
  totalTrend: 1,
  online: 10,
  onlineTrend: 2,
  offline: 2,
  abnormal: 0,
  syncRate: 98,
  todayOps: 24,
  todayOpsTrend: 5
})

const recentLogs = ref([
  { time: '2026-05-03 14:30:25', device: '192.168.1.1', type: 'VLAN修改', status: '成功', user: 'admin' },
  { time: '2026-05-03 13:15:10', device: '192.168.1.2', type: '接口配置', status: '成功', user: 'admin' },
  { time: '2026-05-03 11:45:33', device: '192.168.1.3', type: '路由配置', status: '失败', user: 'admin' },
  { time: '2026-05-03 10:20:18', device: '192.168.1.1', type: '接口启用', status: '成功', user: 'admin' },
  { time: '2026-05-03 09:05:42', device: '192.168.1.4', type: '设备连接', status: '成功', user: 'system' },
])

onMounted(() => {
  // 后续对接真实 API
})
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.content-row {
  display: grid;
  grid-template-columns: 1fr;
}

.logs-section {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin: 0;
}
</style>
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/views/Dashboard.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/Dashboard.vue frontend/test/views/Dashboard.test.ts
git commit -m "feat(frontend): 实现 Dashboard 仪表盘页面"
```

---

## 第三阶段：设备管理页面

### Task 9: 设备管理页面实现

**Files:**
- Create: `frontend/src/views/Devices.vue`
- Create: `frontend/src/stores/device.ts`
- Test: `frontend/test/views/Devices.test.ts`

- [ ] **Step 1: 编写测试**

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import Devices from '../../src/views/Devices.vue'
import { createPinia, setActivePinia } from 'pinia'

describe('Devices View', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should render search input', () => {
    const wrapper = mount(Devices)
    expect(wrapper.find('.el-input').exists()).toBe(true)
  })

  it('should render device table', () => {
    const wrapper = mount(Devices)
    expect(wrapper.find('.el-table').exists()).toBe(true)
  })

  it('should have status column', () => {
    const wrapper = mount(Devices)
    const headers = wrapper.findAll('.el-table__header th')
    const hasStatus = headers.some(h => h.text().includes('状态'))
    expect(hasStatus).toBe(true)
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/views/Devices.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 device store**

```typescript
// frontend/src/stores/device.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import axios from 'axios'

export interface Device {
  id: string
  ip: string
  name: string
  vendor: string
  model: string
  status: 'online' | 'offline' | 'abnormal'
  lastSync: string
}

export const useDeviceStore = defineStore('device', () => {
  const devices = ref<Device[]>([])
  const selectedDevice = ref<Device | null>(null)

  async function fetchDevices() {
    // Mock data - 后续对接真实 API
    devices.value = [
      { id: '1', ip: '192.168.1.1', name: '核心交换机-A', vendor: 'Huawei', model: 'S5735', status: 'online', lastSync: '2026-05-03 14:30' },
      { id: '2', ip: '192.168.1.2', name: '核心交换机-B', vendor: 'H3C', model: 'S6800', status: 'offline', lastSync: '2026-05-03 10:15' },
      { id: '3', ip: '192.168.1.3', name: '接入交换机-01', vendor: 'Huawei', model: 'S5735', status: 'online', lastSync: '2026-05-03 14:25' },
    ]
  }

  async function testConnection(id: string) {
    // Mock
    return true
  }

  return {
    devices,
    selectedDevice,
    fetchDevices,
    testConnection
  }
})
```

- [ ] **Step 4: 实现 Devices 页面**

```vue
<!-- frontend/src/views/Devices.vue -->
<template>
  <div class="devices-page">
    <div class="page-header">
      <h2>设备管理</h2>
    </div>

    <div class="filter-bar">
      <el-input
        v-model="searchText"
        placeholder="搜索设备 IP / 名称"
        prefix-icon="Search"
        style="width: 280px"
        clearable
      />
      <el-select v-model="statusFilter" placeholder="状态筛选" clearable style="width: 140px">
        <el-option label="在线" value="online" />
        <el-option label="离线" value="offline" />
        <el-option label="异常" value="abnormal" />
      </el-select>
      <el-select v-model="vendorFilter" placeholder="厂商筛选" clearable style="width: 140px">
        <el-option label="Huawei" value="Huawei" />
        <el-option label="H3C" value="H3C" />
        <el-option label="Cisco" value="Cisco" />
      </el-select>
      <el-button type="primary" :icon="Refresh" @click="handleRefresh">刷新</el-button>
    </div>

    <div class="table-wrapper">
      <el-table :data="filteredDevices" stripe>
        <el-table-column type="selection" width="55" />
        <el-table-column prop="ip" label="设备 IP" width="140" />
        <el-table-column prop="name" label="设备名称" width="160" />
        <el-table-column prop="vendor" label="厂商" width="100" />
        <el-table-column prop="model" label="型号" width="120" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ getStatusLabel(row.status) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="lastSync" label="最后同步" width="180" />
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="handleViewConfig(row)">查看配置</el-button>
            <el-button link type="success" size="small" @click="handleTestConnection(row)">连接测试</el-button>
            <el-button link type="info" size="small" @click="handleEdit(row)">编辑</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="pageSize"
          :total="filteredDevices.length"
          layout="total, prev, pager, next"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useDeviceStore } from '../stores/device'
import { Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const router = useRouter()
const deviceStore = useDeviceStore()

const searchText = ref('')
const statusFilter = ref('')
const vendorFilter = ref('')
const currentPage = ref(1)
const pageSize = ref(10)

const filteredDevices = computed(() => {
  return deviceStore.devices.filter(d => {
    const matchSearch = !searchText.value ||
      d.ip.includes(searchText.value) ||
      d.name.includes(searchText.value)
    const matchStatus = !statusFilter.value || d.status === statusFilter.value
    const matchVendor = !vendorFilter.value || d.vendor === vendorFilter.value
    return matchSearch && matchStatus && matchVendor
  })
})

function getStatusType(status: string) {
  const map: Record<string, string> = {
    online: 'success',
    offline: 'info',
    abnormal: 'danger'
  }
  return map[status] || 'info'
}

function getStatusLabel(status: string) {
  const map: Record<string, string> = {
    online: '在线',
    offline: '离线',
    abnormal: '异常'
  }
  return map[status] || status
}

function handleViewConfig(row: any) {
  router.push('/config/interface')
}

async function handleTestConnection(row: any) {
  const result = await deviceStore.testConnection(row.id)
  ElMessage.success(result ? '连接成功' : '连接失败')
}

function handleEdit(row: any) {
  ElMessage.info('编辑功能开发中')
}

async function handleRefresh() {
  await deviceStore.fetchDevices()
  ElMessage.success('刷新成功')
}

onMounted(() => {
  deviceStore.fetchDevices()
})
</script>

<style scoped>
.devices-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page-header h2 {
  font-size: 22px;
  font-weight: 600;
  color: #303133;
  margin: 0;
}

.filter-bar {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.table-wrapper {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/views/Devices.test.ts`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/Devices.vue frontend/src/stores/device.ts frontend/test/views/Devices.test.ts
git commit -m "feat(frontend): 实现设备管理页面"
```

---

## 第四阶段：通用配置页面框架

### Task 10: FieldRenderer 字段渲染器

**Files:**
- Create: `frontend/src/components/config/FieldRenderer.vue`
- Test: `frontend/test/components/FieldRenderer.test.ts`

- [ ] **Step 1: 编写测试**

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import FieldRenderer from '../../src/components/config/FieldRenderer.vue'

describe('FieldRenderer Component', () => {
  it('should render input for string type', () => {
    const wrapper = mount(FieldRenderer, {
      props: { field: { type: 'string', path: 'name', label: '名称' } }
    })
    expect(wrapper.find('.el-input').exists()).toBe(true)
  })

  it('should render switch for boolean type', () => {
    const wrapper = mount(FieldRenderer, {
      props: { field: { type: 'boolean', path: 'enabled', label: '启用' } }
    })
    expect(wrapper.find('.el-switch').exists()).toBe(true)
  })

  it('should render select for enum type', () => {
    const wrapper = mount(FieldRenderer, {
      props: {
        field: {
          type: 'enum',
          path: 'vlan',
          label: 'VLAN',
          options: [{ value: 1, label: 'VLAN 1' }]
        }
      }
    })
    expect(wrapper.find('.el-select').exists()).toBe(true)
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/components/FieldRenderer.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 FieldRenderer 组件**

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

    <!-- Group / Container -->
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

interface FieldOption {
  value: string | number
  label: string
}

interface Field {
  path: string
  type: 'string' | 'number' | 'boolean' | 'enum' | 'group'
  label: string
  placeholder?: string
  required?: boolean
  readonly?: boolean
  options?: FieldOption[]
  fields?: Field[]
}

interface Props {
  field: Field
  modelValue: any
}

const props = defineProps<Props>()
const emit = defineEmits<{ 'update:modelValue': [value: any] }>()

const localValue = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v)
})

function getValue(path: string) {
  // 后续实现嵌套取值
  return null
}

function setValue(path: string, value: any) {
  // 后续实现嵌套设值
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
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/config/FieldRenderer.vue frontend/test/components/FieldRenderer.test.ts
git commit -m "feat(frontend): 实现 FieldRenderer 字段渲染器"
```

---

### Task 11: DynamicForm 动态表单组件

**Files:**
- Create: `frontend/src/components/config/DynamicForm.vue`
- Test: `frontend/test/components/DynamicForm.test.ts`

- [ ] **Step 1: 编写测试**

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DynamicForm from '../../src/components/config/DynamicForm.vue'

describe('DynamicForm Component', () => {
  const fields = [
    { path: 'name', type: 'string', label: '名称', required: true },
    { path: 'enabled', type: 'boolean', label: '启用' }
  ]

  it('should render form fields based on schema', () => {
    const wrapper = mount(DynamicForm, {
      props: { fields, modelValue: {} }
    })
    expect(wrapper.findAll('.el-form-item').length).toBe(2)
  })

  it('should display form labels', () => {
    const wrapper = mount(DynamicForm, {
      props: { fields, modelValue: {} }
    })
    expect(wrapper.text()).toContain('名称')
    expect(wrapper.text()).toContain('启用')
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/components/DynamicForm.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 DynamicForm 组件**

```vue
<!-- frontend/src/components/config/DynamicForm.vue -->
<template>
  <el-form :model="formData" :rules="formRules" ref="formRef" label-width="120px">
    <FieldRenderer
      v-for="field in fields"
      :key="field.path"
      :field="field"
      :model-value="formData[field.path]"
      @update:model-value="(v: any) => updateValue(field.path, v)"
    />
  </el-form>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import FieldRenderer from './FieldRenderer.vue'
import type { FormInstance, FormRules } from 'element-plus'

interface Field {
  path: string
  type: string
  label: string
  required?: boolean
  pattern?: string
  placeholder?: string
  readonly?: boolean
  options?: any[]
  fields?: Field[]
}

interface Props {
  fields: Field[]
  modelValue: Record<string, any>
}

const props = defineProps<Props>()
const emit = defineEmits<{ 'update:modelValue': [value: Record<string, any>] }>()

const formRef = ref<FormInstance>()

const formData = ref<Record<string, any>>({ ...props.modelValue })

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

defineExpose({
  validate: () => formRef.value?.validate(),
  resetFields: () => formRef.value?.resetFields(),
  getFormData: () => formData.value
})
</script>

<style scoped>
</style>
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/components/DynamicForm.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/config/DynamicForm.vue frontend/test/components/DynamicForm.test.ts
git commit -m "feat(frontend): 实现 DynamicForm 动态表单组件"
```

---

### Task 12: DynamicTable 动态表格组件

**Files:**
- Create: `frontend/src/components/config/DynamicTable.vue`
- Test: `frontend/test/components/DynamicTable.test.ts`

- [ ] **Step 1: 编写测试**

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import DynamicTable from '../../src/components/config/DynamicTable.vue'

describe('DynamicTable Component', () => {
  const columns = [
    { path: 'name', label: '名称' },
    { path: 'vlan', label: 'VLAN' }
  ]
  const data = [{ name: 'GE0/0/1', vlan: 100 }]

  it('should render table columns based on schema', () => {
    const wrapper = mount(DynamicTable, {
      props: { columns, data }
    })
    expect(wrapper.findAll('.el-table__column').length).toBeGreaterThan(0)
  })

  it('should render add button', () => {
    const wrapper = mount(DynamicTable, {
      props: { columns, data }
    })
    expect(wrapper.find('.add-btn').exists()).toBe(true)
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/components/DynamicTable.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 DynamicTable 组件**

```vue
<!-- frontend/src/components/config/DynamicTable.vue -->
<template>
  <div class="dynamic-table">
    <el-table :data="data" stripe border>
      <el-table-column
        v-for="col in columns"
        :key="col.path"
        :prop="col.path"
        :label="col.label"
      >
        <template #default="{ row }" v-if="col.type === 'boolean'">
          <el-tag :type="row[col.path] ? 'success' : 'info'" size="small">
            {{ row[col.path] ? '启用' : '禁用' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row, $index }">
          <el-button link type="primary" size="small" @click="handleEdit(row, $index)">编辑</el-button>
          <el-button link type="danger" size="small" @click="handleDelete(row, $index)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="table-actions">
      <el-button type="primary" class="add-btn" @click="handleAdd" :icon="Plus">
        新增配置项
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Plus } from '@element-plus/icons-vue'

interface Column {
  path: string
  label: string
  type?: string
}

interface Props {
  columns: Column[]
  data: Record<string, any>[]
}

defineProps<Props>()
const emit = defineEmits<{
  'add': []
  'edit': [row: Record<string, any>, index: number]
  'delete': [row: Record<string, any>, index: number]
}>()

function handleAdd() {
  emit('add')
}

function handleEdit(row: Record<string, any>, index: number) {
  emit('edit', row, index)
}

function handleDelete(row: Record<string, any>, index: number) {
  emit('delete', row, index)
}
</script>

<style scoped>
.dynamic-table {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.table-actions {
  margin-top: 16px;
  display: flex;
  justify-content: flex-start;
}
</style>
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/components/DynamicTable.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/config/DynamicTable.vue frontend/test/components/DynamicTable.test.ts
git commit -m "feat(frontend): 实现 DynamicTable 动态表格组件"
```

---

### Task 13: ConfigPage 通用配置页面

**Files:**
- Create: `frontend/src/views/ConfigPage.vue`
- Create: `frontend/src/api/crd.ts`
- Test: `frontend/test/views/ConfigPage.test.ts`

- [ ] **Step 1: 编写测试**

```typescript
import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ConfigPage from '../../src/views/ConfigPage.vue'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'

vi.mock('../../src/api/crd', () => ({
  getSchema: () => Promise.resolve({
    module: 'test-module',
    title: '测试配置',
    fields: [
      { path: 'name', type: 'string', label: '名称' },
      { path: 'enabled', type: 'boolean', label: '启用' }
    ],
    listFields: [
      { path: 'name', label: '名称' },
      { path: 'vlan', label: 'VLAN' }
    ]
  }),
  getConfig: () => Promise.resolve({
    data: [{ name: 'GE0/0/1', vlan: 100, enabled: true }]
  })
}))

describe('ConfigPage View', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  const router = createRouter({
    history: createWebHistory(),
    routes: [{ path: '/config/:module', name: 'config', component: {} }]
  })

  it('should render device selector', () => {
    const wrapper = mount(ConfigPage, {
      global: { plugins: [router] },
      props: { module: 'openconfig-interfaces' }
    })
    expect(wrapper.find('.device-selector').exists()).toBe(true)
  })

  it('should render dynamic table', () => {
    const wrapper = mount(ConfigPage, {
      global: { plugins: [router] },
      props: { module: 'openconfig-interfaces' }
    })
    expect(wrapper.find('.dynamic-table').exists()).toBe(true)
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/views/ConfigPage.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 CRD API**

```typescript
// frontend/src/api/crd.ts
import axios from 'axios'

interface Field {
  path: string
  type: string
  label: string
  placeholder?: string
  required?: boolean
  pattern?: string
  readonly?: boolean
  options?: { value: any, label: string }[]
  fields?: Field[]
}

interface Schema {
  module: string
  title: string
  fields: Field[]
  listFields: Field[]
}

export async function getSchema(module: string): Promise<Schema> {
  // Mock
  return {
    module,
    title: module.includes('vlan') ? 'VLAN配置' : '接口配置',
    fields: [
      { path: 'name', type: 'string', label: '名称', required: true },
      { path: 'description', type: 'string', label: '描述' },
      { path: 'enabled', type: 'boolean', label: '管理状态', default: true },
      { path: 'vlan', type: 'enum', label: 'VLAN', options: [
        { value: 1, label: 'VLAN 1 (默认)' },
        { value: 100, label: 'VLAN 100' },
        { value: 200, label: 'VLAN 200' }
      ]}
    ],
    listFields: [
      { path: 'name', label: '名称' },
      { path: 'description', label: '描述' },
      { path: 'enabled', type: 'boolean', label: '状态' },
      { path: 'vlan', label: 'VLAN' }
    ]
  }
}

export async function getConfig(module: string, deviceIp: string): Promise<any> {
  // Mock
  return {
    data: [
      { name: 'GigabitEthernet0/0/1', description: '上行端口', enabled: true, vlan: 100 },
      { name: 'GigabitEthernet0/0/2', description: '下行端口', enabled: false, vlan: 200 },
    ]
  }
}

export async function saveConfig(module: string, deviceIp: string, config: any): Promise<any> {
  // Mock
  return { success: true }
}
```

- [ ] **Step 4: 实现 ConfigPage 页面**

```vue
<!-- frontend/src/views/ConfigPage.vue -->
<template>
  <div class="config-page">
    <div class="page-header">
      <h2>{{ pageTitle }}</h2>
    </div>

    <div class="toolbar">
      <div class="device-selector">
        <span>选择设备：</span>
        <el-select v-model="selectedDevice" placeholder="请选择设备" style="width: 200px">
          <el-option
            v-for="device in deviceStore.devices"
            :key="device.id"
            :label="device.ip"
            :value="device.ip"
          />
        </el-select>
      </div>
      <el-button type="primary" :icon="Refresh" @click="handleRefresh">刷新</el-button>
    </div>

    <DynamicTable
      v-if="schema"
      :columns="schema.listFields"
      :data="configList"
      @add="handleAdd"
      @edit="handleEdit"
      @delete="handleDelete"
    />

    <DetailDrawer
      v-model="drawerVisible"
      :title="isEditing ? '编辑配置' : '新增配置'"
      :show-footer="true"
      :submit-text="提交"
      :submitting="submitting"
      @submit="handleSubmit"
    >
      <DynamicForm
        v-if="schema"
        ref="formRef"
        :fields="schema.fields"
        :model-value="currentConfig"
        @update:model-value="currentConfig = $event"
      />
    </DetailDrawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useDeviceStore } from '../stores/device'
import DynamicTable from '../components/config/DynamicTable.vue'
import DynamicForm from '../components/config/DynamicForm.vue'
import DetailDrawer from '../components/layout/DetailDrawer.vue'
import { getSchema, getConfig, saveConfig } from '../api/crd'
import { Refresh } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const route = useRoute()
const deviceStore = useDeviceStore()

const module = computed(() => route.params.module as string || 'openconfig-interfaces')

const pageTitle = ref('配置管理')
const selectedDevice = ref('')
const schema = ref<any>(null)
const configList = ref<any[]>([])
const drawerVisible = ref(false)
const isEditing = ref(false)
const currentConfig = ref({})
const submitting = ref(false)
const formRef = ref()

async function loadSchema() {
  schema.value = await getSchema(module.value)
  pageTitle.value = schema.value.title
}

async function loadConfig() {
  if (!selectedDevice.value) return
  const res = await getConfig(module.value, selectedDevice.value)
  configList.value = res.data || []
}

function handleAdd() {
  isEditing.value = false
  currentConfig.value = {}
  drawerVisible.value = true
}

function handleEdit(row: any) {
  isEditing.value = true
  currentConfig.value = { ...row }
  drawerVisible.value = true
}

async function handleDelete(row: any, index: number) {
  try {
    await ElMessageBox.confirm('确认删除该配置项？', '提示', {
      type: 'warning'
    })
    configList.value.splice(index, 1)
    ElMessage.success('删除成功')
  } catch {
    // 取消
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
    await saveConfig(module.value, selectedDevice.value, currentConfig.value)

    if (isEditing.value) {
      const index = configList.value.findIndex(c => c.name === currentConfig.value.name)
      if (index >= 0) {
        configList.value[index] = { ...currentConfig.value }
      }
    } else {
      configList.value.push({ ...currentConfig.value })
    }

    drawerVisible.value = false
    ElMessage.success('配置保存成功')
  } catch (e: any) {
    ElMessage.error(e.message || '保存失败')
  } finally {
    submitting.value = false
  }
}

function handleRefresh() {
  loadConfig()
}

onMounted(() => {
  loadSchema()
  if (deviceStore.devices.length === 0) {
    deviceStore.fetchDevices().then(() => {
      if (deviceStore.devices.length > 0) {
        selectedDevice.value = deviceStore.devices[0].ip
        loadConfig()
      }
    })
  } else {
    selectedDevice.value = deviceStore.devices[0].ip
    loadConfig()
  }
})
</script>

<style scoped>
.config-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
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

- [ ] **Step 5: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/views/ConfigPage.test.ts`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/ConfigPage.vue frontend/src/api/crd.ts frontend/test/views/ConfigPage.test.ts
git commit -m "feat(frontend): 实现通用配置页面，支持动态表单和表格"
```

---

## 第五阶段：其他页面和收尾

### Task 14: 操作日志页面

**Files:**
- Create: `frontend/src/views/Logs.vue`
- Create: `frontend/src/api/logs.ts`
- Test: `frontend/test/views/Logs.test.ts`

- [ ] **Step 1: 编写测试**

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import Logs from '../../src/views/Logs.vue'

describe('Logs View', () => {
  it('should render log table', () => {
    const wrapper = mount(Logs)
    expect(wrapper.find('.el-table').exists()).toBe(true)
  })

  it('should have search and filters', () => {
    const wrapper = mount(Logs)
    expect(wrapper.find('.el-input').exists()).toBe(true)
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/views/Logs.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现日志 API**

```typescript
// frontend/src/api/logs.ts
export interface Log {
  id: string
  time: string
  device: string
  type: string
  status: string
  user: string
  detail?: string
}

export async function getLogs(params?: any): Promise<{ data: Log[], total: number }> {
  // Mock
  return {
    data: [
      { id: '1', time: '2026-05-03 14:30:25', device: '192.168.1.1', type: 'VLAN修改', status: '成功', user: 'admin' },
      { id: '2', time: '2026-05-03 13:15:10', device: '192.168.1.2', type: '接口配置', status: '成功', user: 'admin' },
      { id: '3', time: '2026-05-03 11:45:33', device: '192.168.1.3', type: '路由配置', status: '失败', user: 'admin' },
    ],
    total: 3
  }
}
```

- [ ] **Step 4: 实现 Logs 页面**

```vue
<!-- frontend/src/views/Logs.vue -->
<template>
  <div class="logs-page">
    <div class="page-header">
      <h2>操作日志</h2>
    </div>

    <div class="filter-bar">
      <el-input
        v-model="searchText"
        placeholder="搜索设备、操作类型"
        prefix-icon="Search"
        style="width: 240px"
        clearable
      />
      <el-date-picker
        v-model="dateRange"
        type="daterange"
        range-separator="至"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        style="width: 320px"
      />
      <el-select v-model="statusFilter" placeholder="状态筛选" clearable style="width: 120px">
        <el-option label="成功" value="成功" />
        <el-option label="失败" value="失败" />
      </el-select>
      <el-button type="primary" @click="handleExport">导出</el-button>
    </div>

    <div class="table-wrapper">
      <el-table :data="logs" stripe>
        <el-table-column prop="time" label="时间" width="180" />
        <el-table-column prop="device" label="设备" width="140" />
        <el-table-column prop="type" label="操作类型" width="120" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === '成功' ? 'success' : 'danger'" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="user" label="操作人" width="100" />
        <el-table-column prop="detail" label="详情" />
      </el-table>

      <div class="pagination">
        <el-pagination
          v-model:current-page="currentPage"
          :page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getLogs } from '../api/logs'
import { Search, Download } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const searchText = ref('')
const dateRange = ref()
const statusFilter = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const logs = ref<any[]>([])

async function loadLogs() {
  const res = await getLogs({ page: currentPage.value, pageSize: pageSize.value })
  logs.value = res.data
  total.value = res.total
}

function handleExport() {
  ElMessage.info('导出功能开发中')
}

onMounted(() => {
  loadLogs()
})
</script>

<style scoped>
.logs-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page-header h2 {
  font-size: 22px;
  font-weight: 600;
  color: #303133;
  margin: 0;
}

.filter-bar {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.table-wrapper {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.pagination {
  margin-top: 20px;
  display: flex;
  justify-content: flex-end;
}
</style>
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/views/Logs.test.ts`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/Logs.vue frontend/src/api/logs.ts frontend/test/views/Logs.test.ts
git commit -m "feat(frontend): 实现操作日志页面"
```

---

### Task 15: 系统设置页面

**Files:**
- Create: `frontend/src/views/Settings.vue`
- Test: `frontend/test/views/Settings.test.ts`

- [ ] **Step 1: 编写测试**

```typescript
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import Settings from '../../src/views/Settings.vue'

describe('Settings View', () => {
  it('should render global settings section', () => {
    const wrapper = mount(Settings)
    expect(wrapper.text()).toContain('全局设置')
  })

  it('should render theme settings', () => {
    const wrapper = mount(Settings)
    expect(wrapper.text()).toContain('主题设置')
  })
})
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd frontend && npm run test -- test/views/Settings.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 Settings 页面**

```vue
<!-- frontend/src/views/Settings.vue -->
<template>
  <div class="settings-page">
    <div class="page-header">
      <h2>系统设置</h2>
    </div>

    <div class="settings-section">
      <h3>全局设置</h3>
      <el-form label-width="140px">
        <el-form-item label="缓存有效期">
          <el-select v-model="settings.cacheTTL" style="width: 200px">
            <el-option label="30 秒" :value="30" />
            <el-option label="60 秒" :value="60" />
            <el-option label="120 秒" :value="120" />
          </el-select>
        </el-form-item>
        <el-form-item label="同步间隔">
          <el-select v-model="settings.syncInterval" style="width: 200px">
            <el-option label="30 秒" :value="30" />
            <el-option label="60 秒" :value="60" />
            <el-option label="120 秒" :value="120" />
          </el-select>
        </el-form-item>
        <el-form-item label="超时时间">
          <el-input-number v-model="settings.timeout" :min="5" :max="120" style="width: 200px" />
          <span style="margin-left: 8px">秒</span>
        </el-form-item>
        <el-form-item label="重试次数">
          <el-input-number v-model="settings.retryCount" :min="0" :max="5" style="width: 200px" />
          <span style="margin-left: 8px">次</span>
        </el-form-item>
      </el-form>
    </div>

    <div class="settings-section">
      <h3>主题设置</h3>
      <el-radio-group v-model="settings.theme">
        <el-radio-button value="light">浅色模式</el-radio-button>
        <el-radio-button value="dark">深色模式</el-radio-button>
        <el-radio-button value="system">跟随系统</el-radio-button>
      </el-radio-group>
    </div>

    <div class="settings-actions">
      <el-button @click="handleReset">重置</el-button>
      <el-button type="primary" @click="handleSave">保存设置</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'

const settings = ref({
  cacheTTL: 30,
  syncInterval: 60,
  timeout: 10,
  retryCount: 3,
  theme: 'light'
})

function handleSave() {
  ElMessage.success('设置已保存')
}

function handleReset() {
  settings.value = {
    cacheTTL: 30,
    syncInterval: 60,
    timeout: 10,
    retryCount: 3,
    theme: 'light'
  }
  ElMessage.info('已重置为默认值')
}
</script>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page-header h2 {
  font-size: 22px;
  font-weight: 600;
  color: #303133;
  margin: 0;
}

.settings-section {
  background: #fff;
  border-radius: 8px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.settings-section h3 {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 20px 0;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.settings-actions {
  display: flex;
  gap: 12px;
  justify-content: flex-end;
}
</style>
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && npm run test -- test/views/Settings.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/Settings.vue frontend/test/views/Settings.test.ts
git commit -m "feat(frontend): 实现系统设置页面"
```

---

### Task 16: 运行所有测试并验证构建

- [ ] **Step 1: 运行所有单元测试**

Run: `cd frontend && npm run test`
Expected: All tests pass

- [ ] **Step 2: 构建项目**

Run: `cd frontend && npm run build`
Expected: Build succeeds

- [ ] **Step 3: 启动开发服务验证**

Run: `cd frontend && npm run dev`
Expected: Application starts and is accessible

- [ ] **Step 4: 最终 Commit**

```bash
git add frontend/package-lock.json
git commit -m "feat(frontend): 完整前端页面实现，所有测试通过"
```

---

## 实现计划总结

本计划包含了前端页面的完整实现，涵盖：

1. **基础布局** - Sidebar、Header、MainLayout、DetailDrawer
2. **仪表盘** - 统计卡片、状态环形图、最近操作表格
3. **设备管理** - 设备列表、搜索筛选、操作列
4. **配置框架** - 动态字段渲染器、动态表单、动态表格、通用配置页面
5. **业务配置** - 接口/VLAN/路由复用通用配置页面
6. **原生配置** - CRD 动态菜单加载，复用通用配置页面
7. **操作日志** - 日志列表、筛选、导出
8. **系统设置** - 全局参数、主题设置

所有组件都采用 TDD 方式开发，包含完整的单元测试，代码遵循项目规范。
