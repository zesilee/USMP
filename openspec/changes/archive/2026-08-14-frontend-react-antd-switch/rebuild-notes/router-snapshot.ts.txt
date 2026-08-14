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
  {
    // rpc 直达（FE-19）：左树 rpc 节点入口，仅渲染该 rpc 执行面板。
    path: '/module/:module/rpc/:rpcName',
    name: 'module-rpc',
    component: () => import('../views/ModuleConsolePage.vue')
  },
  // legacy 路由一律不存在（FE-13）：/native/:module、/config/route（Stack A CRD 死路，
  // 生产中从未可用）；/config/interface、/config/vlan 重定向兼容期已结束，站内入口直连 /module/:module。
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
