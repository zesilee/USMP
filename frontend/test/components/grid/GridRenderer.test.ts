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
        { id: 'name', type: 'text' as const, label: '接口名称', readonly: true },
        { id: 'mtu', type: 'number' as const, label: 'MTU' },
        { id: 'admin-status', type: 'select' as const, label: '管理状态', options: [{ label: '启用', value: 2 }, { label: '禁用', value: 1 }] }
      ]
    }
  ],
  values: {}
}

const mockValues = {
  'interfaces-table': [
    { name: 'GigabitEthernet0/0/1', mtu: 1500, 'admin-status': 2 }
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
      { name: 'GigabitEthernet0/0/1', mtu: 1500, 'admin-status': 2 }
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

  it('should emit updated table values after editing a row', async () => {
    const wrapper = mount(GridRenderer, {
      props: {
        schema: mockSchema,
        modelValue: mockValues
      },
      global: { plugins: [ElementPlus] },
      attachTo: document.body
    })

    await flushPromises()
    await wrapper.find('[data-test="grid-edit-row"]').trigger('click')
    await flushPromises()

    await wrapper.find('[data-test="grid-field-mtu"] input').setValue('9000')
    await wrapper.find('[data-test="grid-save-row"]').trigger('click')
    await flushPromises()

    expect(wrapper.emitted('update:modelValue')).toBeDefined()
    const updates = wrapper.emitted('update:modelValue') as unknown[][]
    expect(updates.at(-1)?.[0]).toEqual({
      'interfaces-table': [
        { name: 'GigabitEthernet0/0/1', mtu: 9000, 'admin-status': 2 }
      ]
    })
    wrapper.unmount()
  })

  it('should render backend field errors for table rows', async () => {
    const wrapper = mount(GridRenderer, {
      props: {
        schema: mockSchema,
        modelValue: mockValues,
        errors: {
          'interfaces-table:row:GigabitEthernet0/0/1:mtu': ['MTU 必须在 1280 到 9216 之间']
        }
      },
      global: { plugins: [ElementPlus] }
    })

    await flushPromises()

    expect(wrapper.text()).toContain('MTU 必须在 1280 到 9216 之间')
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
