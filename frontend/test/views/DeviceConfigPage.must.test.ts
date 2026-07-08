import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import DeviceConfigPage from '../../src/views/DeviceConfigPage.vue'
import { getYangSchema, getConfig, setConfig, getDeviceReconcile } from '../../src/api'

vi.mock('../../src/api')

// IFM 阻尼约束形态：reuse 受 must `(../suppress>../reuse)` 约束（reuse 必须小于 suppress）。
const nested = {
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
            { path: '/ifm/interfaces/interface/suppress', type: 'number', label: 'suppress' },
            {
              path: '/ifm/interfaces/interface/reuse',
              type: 'number',
              label: 'reuse',
              must: [{ expr: '(../suppress>../reuse)', message: 'reuse 必须小于 suppress' }],
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

describe('DeviceConfigPage · must 跨字段校验拦截下发（数据驱动，无硬编码）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: nested } } as any)
    vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: { interface: [] } } } } as any)
    vi.mocked(setConfig).mockResolvedValue({ data: { data: { reconciliation: { triggered: true } } } } as any)
    vi.mocked(getDeviceReconcile)
      .mockResolvedValueOnce({ data: { data: { statuses: [] } } } as any)
      .mockResolvedValue({ data: { data: { statuses: [{ path: '/' + options.configPath, outcome: 'converged', last_run: '2026-07-06T10:00:05Z' }] } } } as any)
  })

  it('reuse>suppress 违反 must → submittable=false，提交被拦截、setConfig 不调用', async () => {
    const w = mountPage()
    await flushPromises()
    const vm = w.vm as any
    vm.selectedDevice = '10.0.0.1'
    vm.openAdd()
    await flushPromises()

    vm.formData.name = 'GE0/0/1'
    vm.formData.suppress = 2000
    vm.formData.reuse = 3000 // 违反 (../suppress>../reuse)
    await flushPromises()

    expect(vm.submittable).toBe(false)
    await vm.submit()
    await flushPromises()
    expect(setConfig).not.toHaveBeenCalled()

    w.unmount()
  })

  it('reuse<suppress 满足 must → submittable=true，正常下发', async () => {
    const w = mountPage()
    await flushPromises()
    const vm = w.vm as any
    vm.selectedDevice = '10.0.0.1'
    vm.openAdd()
    await flushPromises()

    vm.formData.name = 'GE0/0/1'
    vm.formData.suppress = 2000
    vm.formData.reuse = 750 // 满足约束
    await flushPromises()

    expect(vm.submittable).toBe(true)
    await vm.submit()
    await flushPromises()
    expect(setConfig).toHaveBeenCalledTimes(1)
    const payload = vi.mocked(setConfig).mock.calls[0][2] as any
    expect(payload.interface[0]).toMatchObject({ name: 'GE0/0/1', suppress: 2000, reuse: 750 })

    w.unmount()
  })
})
