import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import Sidebar from '../../src/components/layout/Sidebar.vue'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import ElementPlus from 'element-plus'
import * as apiModule from '../../src/api'

// 菜单模块列表走 api 客户端（listYangModules，绝对 baseURL）——staging nginx
// 不代理 /api，裸相对 fetch 会命中 SPA fallback 返回 index.html。测试统一 mock
// 该导出，而非 stubGlobal(fetch)。
function mockYangModules(data: any[]) {
  return vi.spyOn(apiModule, 'listYangModules').mockResolvedValue({ data: { data } } as any)
}

// LT-03：左树接口默认 mock 为失败（既有用例走 category 降级路径，行为不变）；
// 左树用例单独 mock 成功载荷。
function mockLeftTree(data: any[] | Error) {
  if (data instanceof Error) return vi.spyOn(apiModule, 'getLeftTree').mockRejectedValue(data)
  return vi.spyOn(apiModule, 'getLeftTree').mockResolvedValue({ data: { data } } as any)
}

vi.mock('@element-plus/icons-vue', () => ({
  DataLine: { template: '<span />' },
  Monitor: { template: '<span />' },
  Connection: { template: '<span />' },
  Share: { template: '<span />' },
  Setting: { template: '<span />' },
  Document: { template: '<span />' },
  Tools: { template: '<span />' },
  Loading: { template: '<span />' },
  Fold: { template: '<span />' },
  Expand: { template: '<span />' }
}))

describe('Sidebar Component', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    mockLeftTree(new Error('lefttree unavailable in legacy tests'))
  })

  const router = createRouter({
    history: createWebHistory(),
    routes: [{ path: '/', name: 'dashboard', component: {} }]
  })

  it('should render static menu items', () => {
    const wrapper = mount(Sidebar, { global: { plugins: [router, ElementPlus] } })
    expect(wrapper.text()).toContain('概览')
    expect(wrapper.text()).toContain('设备管理')
    expect(wrapper.text()).toContain('原生配置')
  })

  // 概念重定位：模块控制台菜单 = 原生配置；旧「业务网络配置」文案与
  // Stack A CRD 菜单（/native/*）不得存在（FE-13 菜单命名与概念对齐场景）
  it('概念对齐：无「业务网络配置」文案、无 /native/* 菜单项', () => {
    const wrapper = mount(Sidebar, { global: { plugins: [router, ElementPlus] } })
    expect(wrapper.text()).not.toContain('业务网络配置')
    const deadItems = wrapper.findAll('.el-menu-item').filter((n) =>
      (n.attributes('index') || '').startsWith('/native/'))
    expect(deadItems).toHaveLength(0)
  })

  it('should toggle menu collapse', async () => {
    const wrapper = mount(Sidebar, { global: { plugins: [router, ElementPlus] } })
    const collapseBtn = wrapper.find('.collapse-btn')
    if (collapseBtn.exists()) {
      await collapseBtn.trigger('click')
      expect(wrapper.find('.sidebar').classes()).toContain('collapsed')
    }
  })
})

describe('Sidebar · 原生配置菜单模型驱动（FE-13）', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  const router = createRouter({
    history: createWebHistory(),
    routes: [{ path: '/', name: 'dashboard', component: {} }]
  })

  it('原生配置子菜单项来自 menu store，指向 /module/:name', async () => {
    mockYangModules([{ name: 'ifm', description: '接口管理', vendor: 'huawei' }])
    const wrapper = mount(Sidebar, { global: { plugins: [router, ElementPlus] } })
    await new Promise((r) => setTimeout(r))
    expect(wrapper.text()).toContain('接口管理')
    const item = wrapper.findAll('.el-menu-item').find((n) => n.text().includes('接口管理'))
    expect(item).toBeTruthy()
  })

  it('模块列表加载失败：回退内置项，菜单不空（R08）', async () => {
    vi.spyOn(apiModule, 'listYangModules').mockRejectedValue(new Error('down'))
    const wrapper = mount(Sidebar, { global: { plugins: [router, ElementPlus] } })
    await new Promise((r) => setTimeout(r))
    expect(wrapper.text()).toContain('接口管理')
    expect(wrapper.text()).toContain('VLAN 配置')
  })
})

describe('Sidebar · 业务菜单任务域分组（FE-13）', () => {
  const router = createRouter({
    history: createWebHistory(),
    routes: [{ path: '/', name: 'dashboard', component: {} }],
  })

  it('模块带 category → 渲染分组标题；未标注模块归「其他」', async () => {
    setActivePinia(createPinia())
    mockYangModules([
      { name: 'ifm', description: '接口管理', vendor: 'huawei', category: 'interface-mgr' },
      { name: 'bgp', description: 'BGP', vendor: 'huawei' },
    ])
    const wrapper = mount(Sidebar, { global: { plugins: [router, ElementPlus] } })
    await new Promise((r) => setTimeout(r))
    expect(wrapper.text()).toContain('interface-mgr')
    expect(wrapper.text()).toContain('其他')
    expect(wrapper.find('[data-test="module-item-ifm"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="module-item-bgp"]').exists()).toBe(true)
  })

  it('全部无 category → 不渲染分组标题（平铺，渲染不失败 R08）', async () => {
    setActivePinia(createPinia())
    mockYangModules([{ name: 'ifm', description: '接口管理', vendor: 'huawei' }])
    const wrapper = mount(Sidebar, { global: { plugins: [router, ElementPlus] } })
    await new Promise((r) => setTimeout(r))
    expect(wrapper.find('[data-test="module-item-ifm"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('其他')
  })
})


describe('Sidebar · SND 左树（LT-03）', () => {
  const router = createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/', name: 'dashboard', component: {} },
      { path: '/module/:module', name: 'module-console', component: {} },
    ],
  })

  const sampleTree = [
    {
      zh: '以太网交换', en: 'Ethernet Switching',
      children: [
        { zh: 'VLAN', en: 'VLAN', children: [
          { zh: 'huawei-vlan', en: 'huawei-vlan', sourceModule: 'huawei-vlan', available: true, module: 'vlan' },
        ]},
      ],
    },
    {
      zh: '安全', en: 'Security',
      children: [{ zh: 'huawei-dsa', en: 'huawei-dsa', sourceModule: 'huawei-dsa', available: false }],
    },
  ]

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('左树加载成功：渲染分组与叶子，已接入叶路由 /module/<root>', async () => {
    mockLeftTree(sampleTree)
    mockYangModules([])
    const wrapper = mount(Sidebar, { global: { plugins: [router, ElementPlus] } })
    await new Promise((r) => setTimeout(r))
    expect(wrapper.find('[data-test="lefttree-group-以太网交换"]').exists()).toBe(true)
    const vlanLeaf = wrapper.find('[data-test="lefttree-leaf-huawei-vlan"]')
    expect(vlanLeaf.exists()).toBe(true)
    await vlanLeaf.trigger('click')
    await new Promise((r) => setTimeout(r))
    expect(router.currentRoute.value.path).toBe('/module/vlan')
  })

  it('未接入叶：禁用态 + 「未接入」占位（全树+占位拍板）', async () => {
    mockLeftTree(sampleTree)
    mockYangModules([])
    const wrapper = mount(Sidebar, { global: { plugins: [router, ElementPlus] } })
    await new Promise((r) => setTimeout(r))
    const dsaLeaf = wrapper.find('[data-test="lefttree-leaf-huawei-dsa"]')
    expect(dsaLeaf.exists()).toBe(true)
    expect(dsaLeaf.classes()).toContain('is-disabled')
    expect(dsaLeaf.text()).toContain('未接入')
  })

  it('左树失败：回退 category 分组导航（R08）', async () => {
    mockLeftTree(new Error('down'))
    mockYangModules([{ name: 'ifm', description: '接口管理', vendor: 'huawei', category: 'interface-mgr' }])
    const wrapper = mount(Sidebar, { global: { plugins: [router, ElementPlus] } })
    await new Promise((r) => setTimeout(r))
    expect(wrapper.find('[data-test="module-item-ifm"]').exists()).toBe(true)
    expect(wrapper.find('[data-test^="lefttree-leaf"]').exists()).toBe(false)
  })
})
