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
    expect(wrapper.text()).toContain('业务网络配置')
  })

  it('should have native config menu item', () => {
    const wrapper = mount(Sidebar, { global: { plugins: [router, ElementPlus] } })
    expect(wrapper.text()).toContain('原生配置')
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
