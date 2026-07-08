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
    // 通用模块控制台（FE-10）：零 per-module props，Tab/列/表单全部由 schema 派生。
    path: '/module/:module',
    name: 'module-console',
    component: () => import('../views/ModuleConsolePage.vue')
  },
  // 旧配置页路由迁移到通用模块控制台（FE-13）：保留书签可达。
  // DeviceConfigPage.vue 暂存（无路由引用），新控制台稳定后随后续 change 删除。
  {
    path: '/config/interface',
    redirect: '/module/ifm'
  },
  {
    path: '/config/vlan',
    redirect: '/module/vlan'
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
