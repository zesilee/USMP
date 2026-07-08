import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import DeviceConfigPage from '../../src/views/DeviceConfigPage.vue'
import { getYangSchema, getConfig, setConfig, getDeviceReconcile } from '../../src/api'

vi.mock('../../src/api')

// IFM 形态 + choice bandwidth-type（两单叶 case，成员 path 扁平、与 name 同级）。
const ifmNested = {
  fields: [
    {
      path: '/ifm/interfaces',
      type: 'group',
      label: 'interfaces',
      fields: [
        {
          path: '/ifm/interfaces/interface',
          type: 'list',
          label: 'interface',
          fields: [
            { path: '/ifm/interfaces/interface/name', type: 'string', label: '接口名', required: true },
            {
              path: '/ifm/interfaces/interface/bandwidth-type',
              type: 'choice',
              label: 'bandwidth-type',
              cases: [
                { name: 'bandwidth-mbps', label: 'bandwidth-mbps', fields: [{ path: '/ifm/interfaces/interface/bandwidth', type: 'number', label: 'bandwidth' }] },
                { name: 'bandwidth-kbps', label: 'bandwidth-kbps', fields: [{ path: '/ifm/interfaces/interface/bandwidth-kbps', type: 'number', label: 'bandwidth-kbps' }] },
              ],
            },
          ],
        },
      ],
    },
  ],
}

const options = { module: 'ifm', configPath: 'ifm:ifm/ifm:interfaces', itemListSuffix: '/interface', listKey: 'interface', keyField: 'name' }
const columns = [{ prop: 'name', label: '接口名' }]

function mountPage() {
  return mount(DeviceConfigPage, {
    props: { title: '接口配置', addLabel: '新增接口', options, columns },
    global: { plugins: [createPinia(), ElementPlus] },
  })
}

describe('DeviceConfigPage · choice 成员扁平参与差异/下发/互斥清空', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: ifmNested } } as any)
    vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: { interface: [] } } } } as any)
    vi.mocked(setConfig).mockResolvedValue({ data: { data: { reconciliation: { triggered: true } } } } as any)
    vi.mocked(getDeviceReconcile).mockResolvedValue({ data: { data: { statuses: [] } } } as any)
  })

  it('choice 成员填值 → 计入差异且下发 payload 携带扁平成员键', async () => {
    vi.mocked(getDeviceReconcile)
      .mockResolvedValueOnce({ data: { data: { statuses: [] } } } as any)
      .mockResolvedValue({ data: { data: { statuses: [{ path: '/' + options.configPath, outcome: 'converged', last_run: '2026-07-06T10:00:05Z' }] } } } as any)

    const w = mountPage()
    await flushPromises()
    const vm = w.vm as any
    vm.selectedDevice = '10.0.0.1'
    vm.openAdd()
    await flushPromises()

    vm.formData.name = 'GE0/0/1'
    vm.formData.bandwidth = 1000 // 激活 mbps case 的成员（扁平键）
    await flushPromises()

    // choice 成员进入实时差异（否则下发按钮不亮）
    expect(vm.diff.map((d: any) => d.key)).toContain('bandwidth')
    expect(vm.submittable).toBe(true)

    await vm.submit()
    await flushPromises()
    const payload = vi.mocked(setConfig).mock.calls[0][2] as any
    expect(payload.interface[0]).toMatchObject({ name: 'GE0/0/1', bandwidth: 1000 })
    // 未激活 case 成员不应出现
    expect(payload.interface[0]).not.toHaveProperty('bandwidth-kbps')
    // 未出现 choice 结构键（bandwidth-type）——成员是扁平的
    expect(payload.interface[0]).not.toHaveProperty('bandwidth-type')

    w.unmount()
  })

  it('切换 case 清空非激活分支：onChoiceUpdate 按成员键 reconcile 删除', async () => {
    const w = mountPage()
    await flushPromises()
    const vm = w.vm as any
    vm.selectedDevice = '10.0.0.1'
    vm.openAdd()
    await flushPromises()

    vm.formData.name = 'GE0/0/1'
    vm.formData.bandwidth = 1000
    await flushPromises()
    expect(vm.formData.bandwidth).toBe(1000)

    // 模拟 FieldRenderer 切到 kbps case：emit 的 scope 已省略 bandwidth 键
    const choiceField = ifmNested.fields[0].fields[0].fields[1]
    vm.onChoiceUpdate(choiceField, { 'bandwidth-kbps': 64 })
    await flushPromises()

    // 旧 case 成员被删除、新 case 成员写入
    expect(vm.formData).not.toHaveProperty('bandwidth')
    expect(vm.formData['bandwidth-kbps']).toBe(64)

    w.unmount()
  })
})
