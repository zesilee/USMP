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
    component: () => import('../views/InterfaceGridPage.vue')
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
