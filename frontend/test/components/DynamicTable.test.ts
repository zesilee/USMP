import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import DynamicTable from '../../src/components/config/DynamicTable.vue'
import ElementPlus from 'element-plus'

const mockColumns = [
  { path: 'name', type: 'string' as const, label: '名称' },
  { path: 'enabled', type: 'boolean' as const, label: '启用' }
]

const mockData = [
  { name: '接口1', enabled: true },
  { name: '接口2', enabled: false }
]

describe('DynamicTable Component', () => {
  it('should render ElTable component', () => {
    const wrapper = mount(DynamicTable, {
      props: { columns: mockColumns, data: mockData },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.findComponent({ name: 'ElTable' }).exists()).toBe(true)
  })

  it('should render add button', () => {
    const wrapper = mount(DynamicTable, {
      props: { columns: mockColumns, data: mockData },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.find('.add-button').exists()).toBe(true)
  })

  it('should emit add event when add button is clicked', async () => {
    const wrapper = mount(DynamicTable, {
      props: { columns: mockColumns, data: mockData },
      global: { plugins: [ElementPlus] }
    })
    await wrapper.find('.add-button').trigger('click')
    expect(wrapper.emitted('add')).toBeDefined()
  })
})
