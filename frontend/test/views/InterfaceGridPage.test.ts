import { flushPromises, mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import ElementPlus from 'element-plus'
import InterfaceGridPage from '../../src/views/InterfaceGridPage.vue'
import { applyInterfaceGridConfig, getInterfaceGridSchema } from '../../src/api'

vi.mock('../../src/api', () => ({
  getInterfaceGridSchema: vi.fn(),
  applyInterfaceGridConfig: vi.fn()
}))

const schemaResponse = {
  data: {
    success: true,
    data: {
      schemaVersion: 'interfaces:v1',
      module: 'huawei-ifm',
      targetPath: '/ifm:ifm/ifm:interfaces',
      capabilitySource: 'module-set',
      layout: { type: 'grid', columns: 12, gap: 'md' },
      sections: [{ id: 'interfaces', title: '接口配置', widgets: ['interfaces-table'] }],
      widgets: [{
        id: 'interfaces-table',
        type: 'table',
        label: '接口列表',
        rowKey: 'name',
        grid: { span: 12 },
        columns: [{ id: 'name', type: 'text', label: '接口名称' }]
      }],
      values: { 'interfaces-table': [{ name: 'GigabitEthernet0/0/1' }] }
    }
  }
}

describe('InterfaceGridPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getInterfaceGridSchema).mockResolvedValue(schemaResponse as any)
    vi.mocked(applyInterfaceGridConfig).mockResolvedValue({ data: { success: true, data: { schemaVersion: 'interfaces:v1' } } } as any)
  })

  it('loads and renders backend grid schema', async () => {
    const wrapper = mount(InterfaceGridPage, {
      props: { deviceIp: '192.168.1.1' },
      global: { plugins: [ElementPlus] }
    })
    await flushPromises()

    expect(getInterfaceGridSchema).toHaveBeenCalledWith('192.168.1.1')
    expect(wrapper.text()).toContain('接口配置')
    expect(wrapper.text()).toContain('GigabitEthernet0/0/1')
  })

  it('submits schemaVersion and values to apply api', async () => {
    const wrapper = mount(InterfaceGridPage, {
      props: { deviceIp: '192.168.1.1' },
      global: { plugins: [ElementPlus] }
    })
    await flushPromises()

    await wrapper.get('[data-test="grid-submit"]').trigger('click')
    await flushPromises()

    expect(applyInterfaceGridConfig).toHaveBeenCalledWith('192.168.1.1', {
      schemaVersion: 'interfaces:v1',
      values: { 'interfaces-table': [{ name: 'GigabitEthernet0/0/1' }] }
    })
  })
})
