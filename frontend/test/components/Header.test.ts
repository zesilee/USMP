import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import Header from '../../src/components/layout/Header.vue'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'

vi.mock('@element-plus/icons-vue', () => ({
  Bell: { template: '<span />' },
  MoreFilled: { template: '<span />' },
  Search: { template: '<span />' }
}))

describe('Header Component', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('should render search input', () => {
    const wrapper = mount(Header, { global: { plugins: [ElementPlus] } })
    expect(wrapper.find('.el-input').exists()).toBe(true)
  })

  it('should render notification icon', () => {
    const wrapper = mount(Header, { global: { plugins: [ElementPlus] } })
    expect(wrapper.find('.el-badge').exists()).toBe(true)
  })

  it('should display device status', () => {
    const wrapper = mount(Header, { global: { plugins: [ElementPlus] } })
    expect(wrapper.find('.device-status').exists()).toBe(true)
  })
})
