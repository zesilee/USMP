import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import Dashboard from '../../src/views/Dashboard.vue'
import ElementPlus from 'element-plus'

vi.mock('@element-plus/icons-vue', () => ({
  Monitor: { template: '<span />' },
  SuccessFilled: { template: '<span />' },
  CircleCheck: { template: '<span />' },
  Document: { template: '<span />' }
}))

vi.mock('echarts', () => ({
  init: vi.fn(() => ({
    setOption: vi.fn(),
    resize: vi.fn(),
    dispose: vi.fn()
  }))
}))

describe('Dashboard View', () => {
  it('should render 4 StatCard components', () => {
    const wrapper = mount(Dashboard, {
      global: { plugins: [ElementPlus] }
    })
    const statCards = wrapper.findAllComponents({ name: 'StatCard' })
    expect(statCards.length).toBe(4)
  })

  it('should render StatusChart component', () => {
    const wrapper = mount(Dashboard, {
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.findComponent({ name: 'StatusChart' }).exists()).toBe(true)
  })

  it('should render operation log table', () => {
    const wrapper = mount(Dashboard, {
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.find('.el-table').exists()).toBe(true)
  })

  it('should display correct chart data', () => {
    const wrapper = mount(Dashboard, {
      global: { plugins: [ElementPlus] }
    })
    const chart = wrapper.findComponent({ name: 'StatusChart' })
    expect(chart.props('online')).toBe(10)
    expect(chart.props('offline')).toBe(2)
    expect(chart.props('abnormal')).toBe(0)
  })
})
