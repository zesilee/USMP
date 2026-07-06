import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import DeviceConfigPage from '../../src/views/DeviceConfigPage.vue'
import { getYangSchema, getConfig, setConfig } from '../../src/api'

vi.mock('../../src/api')

const vlanNested = {
  fields: [
    {
      path: '/vlan/vlans',
      type: 'group',
      label: 'vlans',
      fields: [
        {
          path: '/vlan/vlans/vlan',
          type: 'list',
          label: 'vlan',
          fields: [
            { path: '/vlan/vlans/vlan/id', type: 'number', label: 'VLAN ID', required: true },
            { path: '/vlan/vlans/vlan/name', type: 'string', label: '名称' },
          ],
        },
      ],
    },
  ],
}

const options = { module: 'vlan', configPath: 'huawei-vlan:vlan/vlans', itemListSuffix: '/vlan', listKey: 'vlans', keyField: 'id' }
const columns = [{ prop: 'id', label: 'VLAN ID' }, { prop: 'name', label: '名称' }]

function mountPage() {
  return mount(DeviceConfigPage, {
    props: { title: 'VLAN 配置', addLabel: '新增 VLAN', options, columns },
    global: { plugins: [createPinia(), ElementPlus] },
  })
}

describe('DeviceConfigPage · 实时差异预览接线', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: vlanNested } } as any)
    vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: { vlans: [] } } } } as any)
    vi.mocked(setConfig).mockResolvedValue({ data: {} } as any)
  })

  it('新增空表单无改动 → submittable=false；填入后有 diff → true', async () => {
    const w = mountPage()
    await flushPromises()
    const vm = w.vm as any
    vm.selectedDevice = '10.0.0.1'
    vm.openAdd()
    await flushPromises()
    expect(vm.submittable).toBe(false)
    expect(vm.diff.length).toBe(0)

    vm.formData.id = 100
    vm.formData.name = 'guest'
    await flushPromises()
    expect(vm.diff.map((d: any) => d.key).sort()).toEqual(['id', 'name'])
    expect(vm.submittable).toBe(true)
    w.unmount()
  })

  it('缺失必填(keyField)时 submittable=false', async () => {
    const w = mountPage()
    await flushPromises()
    const vm = w.vm as any
    vm.selectedDevice = '10.0.0.1'
    vm.openAdd()
    vm.formData.name = 'guest' // 仅填非主键
    await flushPromises()
    expect(vm.diff.length).toBe(1)
    expect(vm.submittable).toBe(false) // id(keyField) 未填
    w.unmount()
  })

  it('编辑态以已回填行为基线，仅改动字段计入 diff', async () => {
    const w = mountPage()
    await flushPromises()
    const vm = w.vm as any
    vm.selectedDevice = '10.0.0.1'
    vm.openEdit({ id: 100, name: 'old' })
    await flushPromises()
    expect(vm.diff.length).toBe(0) // 未改动
    vm.formData.name = 'new'
    await flushPromises()
    expect(vm.diff.map((d: any) => d.key)).toEqual(['name'])
    expect(vm.diff[0]).toMatchObject({ was: 'old', now: 'new' })
    w.unmount()
  })

  it('提交调用 saveItem(setConfig) 下发并重读列表', async () => {
    const w = mountPage()
    await flushPromises()
    const vm = w.vm as any
    vm.selectedDevice = '10.0.0.1'
    vm.openAdd()
    vm.formData.id = 100
    vm.formData.name = 'guest'
    await vm.submit()
    await flushPromises()
    expect(setConfig).toHaveBeenCalledWith('10.0.0.1', options.configPath, { vlans: [{ id: 100, name: 'guest' }] })
    w.unmount()
  })
})
