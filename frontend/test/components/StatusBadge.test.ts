import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StatusBadge from '../../src/components/common/StatusBadge.vue'
import ElementPlus from 'element-plus'

describe('StatusBadge Component', () => {
  it('should render Pending state correctly', () => {
    const wrapper = mount(StatusBadge, {
      props: { phase: 'Pending' },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.text()).toContain('待同步')
    expect(wrapper.find('.status-pending').exists()).toBe(true)
  })

  it('should render Updating state with loading icon', () => {
    const wrapper = mount(StatusBadge, {
      props: { phase: 'Updating' },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.text()).toContain('同步中')
    expect(wrapper.find('.status-updating').exists()).toBe(true)
    expect(wrapper.find('.el-icon').exists()).toBe(true)
  })

  it('should render Ready state correctly', () => {
    const wrapper = mount(StatusBadge, {
      props: { phase: 'Ready' },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.text()).toContain('已同步')
    expect(wrapper.find('.status-ready').exists()).toBe(true)
  })

  it('should render Failed state correctly', () => {
    const wrapper = mount(StatusBadge, {
      props: { phase: 'Failed' },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.text()).toContain('同步失败')
    expect(wrapper.find('.status-failed').exists()).toBe(true)
  })
})
