import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ConfigPage from '../../src/views/ConfigPage.vue'
import ElementPlus from 'element-plus'
import * as composables from '../../src/composables/useDeviceConfig'

vi.mock('../../src/composables/useDeviceConfig', () => ({
  useDeviceConfig: vi.fn(() => ({
    configCR: null,
    schema: { value: { fields: [], listFields: [] } },
    isLoading: { value: false },
    isSyncing: { value: false },
    currentPhase: { value: 'Ready' },
    error: { value: null },
    save: vi.fn(),
    remove: vi.fn(),
    refresh: vi.fn()
  }))
}))

describe('ConfigPage View', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('should render page title', () => {
    const wrapper = mount(ConfigPage, {
      props: { module: 'interface' },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.find('.page-title').exists()).toBe(true)
  })

  it('should render device select dropdown', () => {
    const wrapper = mount(ConfigPage, {
      props: { module: 'interface' },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.findComponent({ name: 'ElSelect' }).exists()).toBe(true)
  })

  it('should display correct module title', () => {
    const wrapper = mount(ConfigPage, {
      props: { module: 'interface' },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.find('.page-title').text()).toContain('接口配置')
  })
})
