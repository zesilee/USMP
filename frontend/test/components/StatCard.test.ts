import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StatCard from '../../src/components/dashboard/StatCard.vue'
import ElementPlus from 'element-plus'

describe('StatCard Component', () => {
  it('should render title and value correctly', () => {
    const wrapper = mount(StatCard, {
      props: {
        title: '设备总数',
        value: 28
      },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.text()).toContain('设备总数')
    expect(wrapper.text()).toContain('28')
  })

  it('should render trend indicator with positive trend', () => {
    const wrapper = mount(StatCard, {
      props: {
        title: '在线设备',
        value: 25,
        trend: 5,
        trendLabel: '较昨日'
      },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.text()).toContain('较昨日')
    expect(wrapper.text()).toContain('+5')
    expect(wrapper.find('.trend-positive').exists()).toBe(true)
  })

  it('should apply red class for negative trend', () => {
    const wrapper = mount(StatCard, {
      props: {
        title: '离线设备',
        value: 3,
        trend: -2,
        trendLabel: '较昨日'
      },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.text()).toContain('-2')
    expect(wrapper.find('.trend-negative').exists()).toBe(true)
  })
})
