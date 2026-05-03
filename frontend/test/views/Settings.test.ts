import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import Settings from '../../src/views/Settings.vue'
import ElementPlus from 'element-plus'
import axios from 'axios'

vi.mock('axios')

describe('Settings View', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(axios.get).mockResolvedValue({ data: {} })
    vi.mocked(axios.post).mockResolvedValue({ data: { success: true } })
  })

  it('should render global settings section', () => {
    const wrapper = mount(Settings, {
      global: { plugins: [ElementPlus, createPinia()] }
    })
    expect(wrapper.text()).toContain('全局设置')
  })

  it('should render theme settings section', () => {
    const wrapper = mount(Settings, {
      global: { plugins: [ElementPlus, createPinia()] }
    })
    expect(wrapper.text()).toContain('主题设置')
  })
})
