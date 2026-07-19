import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import Header from '../../src/components/layout/Header.vue'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import ElementPlus from 'element-plus'
import { useFreshnessStore } from '../../src/stores/freshness'
import { useLocaleStore, LOCALE_STORAGE_KEY } from '../../src/stores/locale'

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

  // ===== UI-01 语言切换（F2）=====
  describe('locale switch', () => {
    afterEach(() => {
      // 复位为 zh-cn，避免污染同文件其它用例（i18n 为模块级单例）
      useLocaleStore().setLocale('zh-cn')
      localStorage.removeItem(LOCALE_STORAGE_KEY)
    })

    it('renders the locale switch control with both options', () => {
      const wrapper = mountHeader()
      expect(wrapper.find('[data-test="locale-switch"]').exists()).toBe(true)
      expect(wrapper.find('[data-test="locale-zh"]').exists()).toBe(true)
      expect(wrapper.find('[data-test="locale-en"]').exists()).toBe(true)
    })

    it('clicking locale-en switches UI text to English and persists en-us', async () => {
      await router.push('/')
      await router.isReady()
      const wrapper = mountHeader()
      // zh 基线：面包屑根为「车队」
      expect(wrapper.find('.crumb-root').text()).toBe('车队')

      await wrapper.find('[data-test="locale-en"]').trigger('click')
      await wrapper.vm.$nextTick()

      expect(wrapper.find('.crumb-root').text()).toBe('Fleet')
      expect(wrapper.find('.crumb').text()).toContain('Fleet Overview')
      expect(localStorage.getItem(LOCALE_STORAGE_KEY)).toBe('en-us')
      expect(useLocaleStore().locale).toBe('en-us')
    })

    it('clicking locale-zh switches back to Chinese and persists zh-cn', async () => {
      const wrapper = mountHeader()
      await wrapper.find('[data-test="locale-en"]').trigger('click')
      await wrapper.vm.$nextTick()
      expect(wrapper.find('.crumb-root').text()).toBe('Fleet')

      await wrapper.find('[data-test="locale-zh"]').trigger('click')
      await wrapper.vm.$nextTick()
      expect(wrapper.find('.crumb-root').text()).toBe('车队')
      expect(localStorage.getItem(LOCALE_STORAGE_KEY)).toBe('zh-cn')
    })
  })
})
