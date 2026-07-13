import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import Sidebar from '../../src/components/layout/Sidebar.vue'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import ElementPlus from 'element-plus'

vi.mock('@element-plus/icons-vue', () => ({
  DataLine: { template: '<span />' },
  Monitor: { template: '<span />' },
  Connection: { template: '<span />' },
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
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      json: async () => ({ data: [{ name: 'ifm', description: '接口管理', vendor: 'huawei' }] })
    }))
    const wrapper = mount(Sidebar, { global: { plugins: [router, ElementPlus] } })
    await new Promise((r) => setTimeout(r))
    expect(wrapper.text()).toContain('接口管理')
    const item = wrapper.findAll('.el-menu-item').find((n) => n.text().includes('接口管理'))
    expect(item).toBeTruthy()
  })

  it('模块列表加载失败：回退内置项，菜单不空（R08）', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('down')))
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
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      json: async () => ({
        data: [
          { name: 'ifm', description: '接口管理', vendor: 'huawei', category: 'interface-mgr' },
          { name: 'interfaces', description: 'oc 接口', vendor: 'openconfig' },
        ],
      }),
    }))
    const wrapper = mount(Sidebar, { global: { plugins: [router, ElementPlus] } })
    await new Promise((r) => setTimeout(r))
    expect(wrapper.text()).toContain('interface-mgr')
    expect(wrapper.text()).toContain('其他')
    expect(wrapper.find('[data-test="module-item-ifm"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="module-item-interfaces"]').exists()).toBe(true)
  })

  it('全部无 category → 不渲染分组标题（平铺，渲染不失败 R08）', async () => {
    setActivePinia(createPinia())
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      json: async () => ({
        data: [{ name: 'ifm', description: '接口管理', vendor: 'huawei' }],
      }),
    }))
    const wrapper = mount(Sidebar, { global: { plugins: [router, ElementPlus] } })
    await new Promise((r) => setTimeout(r))
    expect(wrapper.find('[data-test="module-item-ifm"]').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('其他')
  })
})
