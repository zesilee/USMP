import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import DeviceConfigPage from '../../src/views/DeviceConfigPage.vue'
import { getYangSchema, getConfig, setConfig, getDeviceReconcile } from '../../src/api'

vi.mock('../../src/api')

// IFM 形态嵌套 schema：interface list 含 name(key)、class(enum)、parent-name(when 门控)。
// when 表达式为真实 IFM 契约：parent-name 仅在 class='sub-interface' 时可见。
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
            { path: '/ifm/interfaces/interface/class', type: 'enum', label: '类别' },
            {
              path: '/ifm/interfaces/interface/parent-name',
              type: 'string',
              label: '父接口',
              when: "../class='sub-interface'",
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

// 表单标签（隔离于左侧 SchemaTree —— 后者用不同 class 展示完整架构树，恒含所有叶名）。
// el-form-item__label 只出现在配置表单里，故按此 class 精确取表单字段标签。
function formLabels(w: ReturnType<typeof mountPage>): string[] {
  return w.findAll('.el-form-item__label').map((n) => n.text().trim())
}

describe('DeviceConfigPage · when 驱动的字段显隐（数据驱动，无硬编码）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: ifmNested } } as any)
    vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: { interface: [] } } } } as any)
    vi.mocked(setConfig).mockResolvedValue({ data: { data: { reconciliation: { triggered: true } } } } as any)
    vi.mocked(getDeviceReconcile).mockResolvedValue({ data: { data: { statuses: [] } } } as any)
  })

  it('class≠sub-interface 时 parent-name 隐藏，切到 sub-interface 后显现（无 if(type===) 硬编码）', async () => {
    const w = mountPage()
    await flushPromises()
    const vm = w.vm as any
    vm.selectedDevice = '10.0.0.1'
    vm.openAdd()
    await flushPromises()

    // 初始 class 为空 → parent-name 不在可见字段、不渲染其表单项
    vm.formData.name = 'GE0/0/1'
    vm.formData.class = 'main-interface'
    await flushPromises()
    expect(vm.visibleFields.map((f: any) => f.label)).not.toContain('父接口')
    expect(formLabels(w)).not.toContain('父接口')

    // 切 class=sub-interface → when 求值为真 → parent-name 显现
    vm.formData.class = 'sub-interface'
    await flushPromises()
    expect(vm.visibleFields.map((f: any) => f.label)).toContain('父接口')
    expect(formLabels(w)).toContain('父接口')

    w.unmount()
  })

  it('隐藏字段不进下发 payload；显现后才随下发', async () => {
    const w = mountPage()
    await flushPromises()
    const vm = w.vm as any
    vm.selectedDevice = '10.0.0.1'
    vm.openAdd()
    await flushPromises()

    // 下发后对账收敛（baseline 空 → converged），使 submit 流程快速结束。
    vi.mocked(getDeviceReconcile)
      .mockResolvedValueOnce({ data: { data: { statuses: [] } } } as any)
      .mockResolvedValue({ data: { data: { statuses: [{ path: '/' + options.configPath, outcome: 'converged', last_run: '2026-07-06T10:00:05Z' }] } } } as any)

    // class=main-interface：即便 parent-name 有残留值也不应下发
    vm.formData.name = 'GE0/0/1'
    vm.formData.class = 'main-interface'
    vm.formData['parent-name'] = 'stale'
    await flushPromises()
    await vm.submit()
    await flushPromises()
    const firstPayload = vi.mocked(setConfig).mock.calls[0][2] as any
    expect(firstPayload.interface[0]).not.toHaveProperty('parent-name')
    expect(firstPayload.interface[0]).toMatchObject({ name: 'GE0/0/1', class: 'main-interface' })

    w.unmount()
  })
})
