import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import DeviceConfigPage from '../../src/views/DeviceConfigPage.vue'
import { getYangSchema, getConfig, setConfig, getDeviceReconcile } from '../../src/api'

vi.mock('../../src/api')

// interface `number` 叶：string + pattern（真实 IFM 接口编号正则的简化：数字与斜杠）。
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
            { path: '/ifm/interfaces/interface/number', type: 'string', label: '编号', pattern: '(\\d+/\\d+)|(\\d+)' },
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

describe('DeviceConfigPage · pattern 正则校验拦截下发（数据驱动）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: nested } } as any)
    vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: { interface: [] } } } } as any)
    vi.mocked(setConfig).mockResolvedValue({ data: { data: { reconciliation: { triggered: true } } } } as any)
    vi.mocked(getDeviceReconcile)
      .mockResolvedValueOnce({ data: { data: { statuses: [] } } } as any)
      .mockResolvedValue({ data: { data: { statuses: [{ path: '/' + options.configPath, outcome: 'converged', last_run: '2026-07-06T10:00:05Z' }] } } } as any)
  })

  it('不匹配 pattern → submittable=false，提交被拦截', async () => {
    const w = mountPage()
    await flushPromises()
    const vm = w.vm as any
    vm.selectedDevice = '10.0.0.1'
    vm.openAdd()
    await flushPromises()
    vm.formData.name = 'GE0/0/1'
    vm.formData.number = 'abc' // 不匹配 (\d+/\d+)|(\d+)
    await flushPromises()
    expect(vm.submittable).toBe(false)
    await vm.submit()
    await flushPromises()
    expect(setConfig).not.toHaveBeenCalled()
    w.unmount()
  })

  it('匹配 pattern → 正常下发', async () => {
    const w = mountPage()
    await flushPromises()
    const vm = w.vm as any
    vm.selectedDevice = '10.0.0.1'
    vm.openAdd()
    await flushPromises()
    vm.formData.name = 'GE0/0/1'
    vm.formData.number = '0/0' // 匹配 \d+/\d+
    await flushPromises()
    expect(vm.submittable).toBe(true)
    await vm.submit()
    await flushPromises()
    expect(setConfig).toHaveBeenCalledTimes(1)
    w.unmount()
  })
})
