import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import DeviceConfigPage from '../../src/views/DeviceConfigPage.vue'
import { getYangSchema, getConfig, setConfig, getDeviceReconcile } from '../../src/api'

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

describe('DeviceConfigPage · 下发对账编排接线', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: vlanNested } } as any)
    vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: { vlans: [] } } } } as any)
    vi.mocked(setConfig).mockResolvedValue({ data: { data: { reconciliation: { triggered: true } } } } as any)
    vi.mocked(getDeviceReconcile).mockResolvedValue({ data: { data: { outcome: 'converged' } } } as any)
  })

  it('提交后展示对账进度并到达已收敛，setConfig 被调用、列表重读', async () => {
    const w = mountPage()
    await flushPromises() // onMounted 拉 schema

    const vm = w.vm as any
    vm.selectedDevice = '10.0.0.1'
    vm.openAdd()
    await flushPromises()

    // 填入必填 + 一个改动
    vm.formData.id = 100
    vm.formData.name = 'guest'
    await flushPromises()
    expect(vm.submittable).toBe(true)
    expect(vm.diff.length).toBe(2)

    await vm.submit()
    await flushPromises()

    expect(setConfig).toHaveBeenCalledWith('10.0.0.1', options.configPath, { vlans: [{ id: 100, name: 'guest' }] })
    expect(getConfig).toHaveBeenCalledWith('10.0.0.1', options.configPath, true) // force_refresh 回读
    expect(vm.submitFlow.phase.value).toBe('converged')
    // 下发成功后重读列表（getConfig 至少被普通读+强制读调用）
    expect(vi.mocked(getConfig).mock.calls.some((c) => c[2] === undefined || c[2] === false)).toBe(true)

    // 抽屉切到对账进度视图
    expect(w.findComponent({ name: 'ReconcileSteps' }).exists() || w.find('.reconcile-steps').exists()).toBe(true)
    w.unmount()
  })

  it('无改动时下发按钮禁用（submittable=false）', async () => {
    const w = mountPage()
    await flushPromises()
    const vm = w.vm as any
    vm.selectedDevice = '10.0.0.1'
    vm.openAdd()
    await flushPromises()
    expect(vm.submittable).toBe(false) // 空表单无改动
    w.unmount()
  })

  it('setConfig 失败 → phase=error，不重读列表', async () => {
    vi.mocked(setConfig).mockRejectedValue({ message: '会话超时' })
    const w = mountPage()
    await flushPromises()
    const vm = w.vm as any
    vm.selectedDevice = '10.0.0.1'
    vm.openAdd()
    vm.formData.id = 200
    await vm.submit()
    await flushPromises()
    expect(vm.submitFlow.phase.value).toBe('error')
    expect(getDeviceReconcile).not.toHaveBeenCalled()
    w.unmount()
  })
})
