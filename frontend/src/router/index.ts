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
  {
    path: '/config/interface',
    name: 'interface',
    component: () => import('../views/DeviceConfigPage.vue'),
    props: {
      title: '接口配置（华为）',
      addLabel: '新增接口',
      options: { module: 'ifm', configPath: 'ifm:ifm/ifm:interfaces', itemListSuffix: '/interface', listKey: 'interface', keyField: 'name' },
      columns: [
        { prop: 'name', label: '接口名', width: 200 },
        { prop: 'description', label: '描述' },
        { prop: 'admin-status', label: '管理状态', width: 120 },
        { prop: 'mtu', label: 'MTU', width: 100 }
      ]
    }
  },
  {
    path: '/config/vlan',
    name: 'vlan',
    component: () => import('../views/DeviceConfigPage.vue'),
    props: {
      title: 'VLAN 配置（华为）',
      addLabel: '新增 VLAN',
      options: { module: 'vlan', configPath: 'huawei-vlan:vlan/vlans', itemListSuffix: '/vlan', listKey: 'vlans', keyField: 'id' },
      columns: [
        { prop: 'id', label: 'VLAN ID', width: 120 },
        { prop: 'name', label: '名称', width: 180 },
        { prop: 'description', label: '描述' },
        { prop: 'admin-status', label: '管理状态', width: 120 }
      ]
    }
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
