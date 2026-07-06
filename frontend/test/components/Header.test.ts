import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import Header from '../../src/components/layout/Header.vue'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import ElementPlus from 'element-plus'
import { useFreshnessStore } from '../../src/stores/freshness'

vi.mock('@element-plus/icons-vue', () => ({
  Bell: { template: '<span />' },
  Search: { template: '<span />' }
}))

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'dashboard', component: {} },
    { path: '/config/vlan', name: 'vlan', component: {} }
  ]
})

function mountHeader() {
  return mount(Header, { global: { plugins: [router, ElementPlus] } })
}

describe('Header Component', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should render search input', () => {
    expect(mountHeader().find('.el-input').exists()).toBe(true)
  })

  it('should render notification icon', () => {
    expect(mountHeader().find('.el-badge').exists()).toBe(true)
  })

  it('should display device status', () => {
    expect(mountHeader().find('.device-status').exists()).toBe(true)
  })

  it('should render breadcrumb label from route name', async () => {
    await router.push('/config/vlan')
    await router.isReady()
    const wrapper = mountHeader()
    expect(wrapper.find('.crumb').text()).toContain('VLAN 配置')
  })

  it('should render freshness ring (idle when no cache data)', () => {
    const wrapper = mountHeader()
    expect(wrapper.find('.fresh').exists()).toBe(true)
    expect(wrapper.find('.fresh').classes()).toContain('is-idle')
  })

  it('freshness ring reflects store data (fresh state)', async () => {
    const store = useFreshnessStore()
    store.record({ cache_age_seconds: 0, ttl_seconds: 30, source: 'device' })
    const wrapper = mountHeader()
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.fresh').classes()).toContain('is-fresh')
  })
})
