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
    // 业务网络配置控制台（FE-17）：平台作用域（一个意图实例管 N 台设备），
    // 与设备作用域的 /module/:module 并列。
    path: '/business/:module',
    name: 'business-console',
    component: () => import('../views/BusinessConsolePage.vue')
  },
  {
    // 通用模块控制台（FE-10）：零 per-module props，Tab/列/表单全部由 schema 派生。
    path: '/module/:module',
    name: 'module-console',
    component: () => import('../views/ModuleConsolePage.vue')
  },
  // 旧配置页路由迁移到通用模块控制台（FE-13）：保留书签可达（DeviceConfigPage 已物理删除）。
  {
    path: '/config/interface',
    redirect: '/module/ifm'
  },
  {
    path: '/config/vlan',
    redirect: '/module/vlan'
  },
  // /config/route 与 /native/:module（Stack A CRD 死路）已退役（FE-13）：
  // 生产中从未可用（K8s API 面已退出生产），无重定向义务。
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
