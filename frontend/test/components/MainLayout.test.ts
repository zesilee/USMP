import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import MainLayout from '../../src/components/layout/MainLayout.vue'
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
  Expand: { template: '<span />' },
  Bell: { template: '<span />' },
  MoreFilled: { template: '<span />' },
  Search: { template: '<span />' }
}))

describe('MainLayout Component', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  const router = createRouter({
    history: createWebHistory(),
    routes: [{ path: '/', name: 'dashboard', component: {} }]
  })

  it('should contain Sidebar, Header and main content area', () => {
    const wrapper = mount(MainLayout, { global: { plugins: [router, ElementPlus] } })
    expect(wrapper.find('.sidebar').exists()).toBe(true)
    expect(wrapper.find('.header').exists()).toBe(true)
    expect(wrapper.find('.main-content').exists()).toBe(true)
  })
})
