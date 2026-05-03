import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import StatusChart from '../../src/components/dashboard/StatusChart.vue'
import ElementPlus from 'element-plus'

vi.mock('echarts', () => ({
  init: vi.fn(() => ({
    setOption: vi.fn(),
    resize: vi.fn(),
    dispose: vi.fn()
  }))
}))

describe('StatusChart Component', () => {
  it('should render chart container', () => {
    const wrapper = mount(StatusChart, {
      props: {
        online: 10,
        offline: 2,
        abnormal: 0
      },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.find('.status-chart-container').exists()).toBe(true)
  })

  it('should render legend with correct values', () => {
    const wrapper = mount(StatusChart, {
      props: {
        online: 10,
        offline: 2,
        abnormal: 0
      },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.find('.legend-area').exists()).toBe(true)
    expect(wrapper.text()).toContain('10')
    expect(wrapper.text()).toContain('2')
    expect(wrapper.text()).toContain('0')
  })

  it('should display status labels in legend', () => {
    const wrapper = mount(StatusChart, {
      props: {
        online: 10,
        offline: 2,
        abnormal: 1
      },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.text()).toContain('在线')
    expect(wrapper.text()).toContain('离线')
    expect(wrapper.text()).toContain('异常')
  })
})
