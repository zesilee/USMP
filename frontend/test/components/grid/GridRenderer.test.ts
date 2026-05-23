import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import GridRenderer from '../../../src/components/grid/GridRenderer.vue'
import ElementPlus from 'element-plus'

const mockSchema = {
  schemaVersion: '1.0.0',
  module: 'openconfig-interfaces',
  targetPath: '/interfaces',
  capabilitySource: 'NETCONF 1.0',
  layout: { type: 'grid', columns: 24, gap: '16px' },
  sections: [
    {
      id: 'interfaces-section',
      title: '接口配置',
      description: '管理交换机接口配置',
      widgets: ['interfaces-table']
    }
  ],
  widgets: [
    {
      id: 'interfaces-table',
      type: 'table' as const,
      label: '接口列表',
      rowKey: 'name',
      grid: { span: 24 },
      columns: [
        { id: 'name', type: 'text' as const, label: '接口名称' },
        { id: 'mtu', type: 'number' as const, label: 'MTU' }
      ]
    }
  ],
  values: {}
}

const mockValues = {
  'interfaces-table': [
    { name: 'GigabitEthernet0/0/1', mtu: 1500 }
  ]
}

describe('GridRenderer Component', () => {
  it('should render sections, widgets, and data correctly', async () => {
    const wrapper = mount(GridRenderer, {
      props: {
        schema: mockSchema,
        modelValue: mockValues
      },
      global: { plugins: [ElementPlus] },
      stubs: {
        transition: false,
        'transition-group': false
      }
    })

    await flushPromises()
    expect(wrapper.text()).toContain('接口配置')
    expect(wrapper.text()).toContain('接口列表')
    expect(wrapper.text()).toContain('GigabitEthernet0/0/1')
    expect(wrapper.text()).toContain('MTU')
    // Check that grid widget exists
    const gridWidget = wrapper.findComponent({ name: 'GridWidget' })
    expect(gridWidget.exists()).toBe(true)
    // Check that modelValue has the data
    expect(wrapper.props('modelValue')['interfaces-table']).toEqual([
      { name: 'GigabitEthernet0/0/1', mtu: 1500 }
    ])
  })

  it('should emit refresh event when refresh button is clicked', async () => {
    const wrapper = mount(GridRenderer, {
      props: {
        schema: mockSchema,
        modelValue: mockValues
      },
      global: { plugins: [ElementPlus] }
    })

    await wrapper.find('[data-test="grid-refresh"]').trigger('click')
    expect(wrapper.emitted('refresh')).toBeDefined()
  })

  it('should emit submit event when submit button is clicked', async () => {
    const wrapper = mount(GridRenderer, {
      props: {
        schema: mockSchema,
        modelValue: mockValues
      },
      global: { plugins: [ElementPlus] }
    })

    await wrapper.find('[data-test="grid-submit"]').trigger('click')
    expect(wrapper.emitted('submit')).toBeDefined()
  })
})
