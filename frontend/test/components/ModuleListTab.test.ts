import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus, { ElMessageBox } from 'element-plus'
import ModuleListTab from '../../src/components/config/ModuleListTab.vue'
import { getConfig, setConfig, deleteConfig, getDeviceReconcile } from '../../src/api'
import { deriveTabs } from '../../src/utils/moduleConsole'
import { ifmNestedSchema, seedRows } from '../views/moduleConsole.fixture'

vi.mock('../../src/api')

const interfacesTab = deriveTabs(ifmNestedSchema.fields).find((t) => t.name === 'interfaces')!

function mountTab() {
  return mount(ModuleListTab, {
    props: { tab: interfacesTab, rootName: 'ifm', device: '10.0.0.1' },
    global: { plugins: [createPinia(), ElementPlus] },
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: { interface: seedRows } } } } as any)
  vi.mocked(setConfig).mockResolvedValue({ data: { data: { reconciliation: { triggered: true } } } } as any)
  vi.mocked(getDeviceReconcile).mockResolvedValue({ data: { data: { statuses: [] } } } as any)
})

describe('ModuleListTab · 模型驱动列（FE-11）', () => {
  it('列按分层派生：key→identity→when 条件列→enum，无任何硬编码列名', async () => {
    const w = mountTab()
    await flushPromises()
    const headers = w.findAll('.el-table__header th .cell').map((n) => n.text().trim()).filter(Boolean)
    expect(headers.slice(0, 6)).toEqual(['name', 'class', 'type', 'parent-name', 'number', 'router-type'])
    expect(headers).toContain('admin-status')
    expect(headers).toContain('操作')
  })

  it('渲染 5 条种子数据：enum Tag、up/down 状态点、行级 when 单元格（main 行 “-”）', async () => {
    const w = mountTab()
    await flushPromises()
    const rows = w.findAll('.el-table__body tr')
    expect(rows).toHaveLength(5)

    // enum 列 Tag 化
    expect(w.findAll('.el-table__body .el-tag').length).toBeGreaterThan(0)
    // 值驱动状态点：3 up + 2 down
    expect(w.findAll('.status-cell.ok')).toHaveLength(3)
    expect(w.findAll('.status-cell.bad')).toHaveLength(2)

    // 行级 when：main-interface 行 parent-name 为 “-”，sub 行显示父接口
    const mainRow = rows[0].text()
    const subRow = rows[3].text()
    expect(mainRow).toContain('-')
    expect(subRow).toContain('200GE0/1/0')
  })

  it('读取失败降级：告警可见、表格空（R08）', async () => {
    vi.mocked(getConfig).mockRejectedValue(new Error('device offline'))
    const w = mountTab()
    await flushPromises()
    expect(w.find('.el-alert').exists()).toBe(true)
    expect(w.findAll('.el-table__body tr')).toHaveLength(0)
  })
})

describe('ModuleListTab · 高级搜索（support-filter 驱动，FE-11）', () => {
  it('面板默认折叠，点击「高级搜索」展开；字段集仅 supportFilter 叶（class/type）', async () => {
    const w = mountTab()
    await flushPromises()
    // happy-dom 下 isVisible() 对 v-show 误报，直接断言 display:none 内联样式。
    const panelStyle = () => w.find('.search-panel').attributes('style') || ''
    expect(panelStyle()).toContain('display: none')
    await w.find('.adv-toggle').trigger('click')
    expect(panelStyle()).not.toContain('display: none')
    const labels = w.find('.search-panel').findAll('.el-form-item__label').map((n) => n.text().trim())
    expect(labels).toEqual(['class', 'type'])
  })

  it('class=sub-interface 查询 → 2 行；重置 → 还原 5 行', async () => {
    const w = mountTab()
    await flushPromises()
    const vm = w.vm as any
    vm.draft.class = 'sub-interface'
    vm.applySearch()
    await flushPromises()
    expect(w.findAll('.el-table__body tr')).toHaveLength(2)
    vm.resetSearch()
    await flushPromises()
    expect(w.findAll('.el-table__body tr')).toHaveLength(5)
  })

  it('组合条件 AND：class=sub-interface + type=200GE → 0 行', async () => {
    const w = mountTab()
    await flushPromises()
    const vm = w.vm as any
    vm.draft.class = 'sub-interface'
    vm.draft.type = '200GE'
    vm.applySearch()
    await flushPromises()
    expect(w.findAll('.el-table__body tr')).toHaveLength(0)
  })
})

describe('ModuleListTab · 分页（FE-11）', () => {
  it('pageSize=10 时 5 行单页；缩小 pageSize 生效且总数正确', async () => {
    const w = mountTab()
    await flushPromises()
    const vm = w.vm as any
    expect(w.find('.el-pagination__total').text()).toContain('5')
    vm.pageSize = 2
    await flushPromises()
    expect(w.findAll('.el-table__body tr')).toHaveLength(2)
    vm.page = 3
    await flushPromises()
    expect(w.findAll('.el-table__body tr')).toHaveLength(1)
  })
})

describe('ModuleListTab · 操作门禁（operation-exclude，FE-11）', () => {
  it('编辑态：operationExclude∋update 的叶（class/type/number/parent-name）禁用；新增态可编', async () => {
    const w = mountTab()
    await flushPromises()
    const vm = w.vm as any

    // 编辑 sub-interface 行（parent-name 可见）
    vm.openEdit({ ...seedRows[3] })
    await flushPromises()
    const disabledOf = () => {
      const out: Record<string, boolean> = {}
      for (const f of vm.form.visibleFields.value) {
        out[vm.form.keyOf(f)] = vm.editing && !!f.operationExclude?.includes('update')
      }
      return out
    }
    expect(disabledOf()).toMatchObject({
      name: false,
      class: true,
      type: true,
      'parent-name': true,
      number: true,
      'router-type': true,
      description: false,
    })
    // DOM 证据：编辑抽屉内 class 的 el-select 处于禁用态
    const disabledSelects = w.findAll('.el-drawer .el-select.is-disabled, .el-drawer .el-select .is-disabled')
    expect(disabledSelects.length).toBeGreaterThan(0)

    // 新增态全部可编辑
    vm.openAdd()
    await flushPromises()
    expect(Object.values(disabledOf()).every((v) => v === false)).toBe(true)
  })

  it('list 级无 exclude → 编辑/删除按钮均可用（FE-16 起删除有真实语义）', async () => {
    const w = mountTab()
    await flushPromises()
    const ops = w.findAll('.el-table__body .el-button')
    const texts = ops.map((b) => b.text().trim())
    expect(texts).toContain('编辑')
    expect(texts).toContain('删除')
    const del = ops.find((b) => b.text().trim() === '删除')!
    expect(del.attributes('disabled')).toBeUndefined()
  })

  it('list 级 operationExclude=update|delete → 操作列整列隐藏', async () => {
    const tab = { ...interfacesTab, listField: { ...interfacesTab.listField!, operationExclude: ['update', 'delete'] } }
    const w = mount(ModuleListTab, {
      props: { tab, rootName: 'ifm', device: '10.0.0.1' },
      global: { plugins: [createPinia(), ElementPlus] },
    })
    await flushPromises()
    const headers = w.findAll('.el-table__header th .cell').map((n) => n.text().trim())
    expect(headers).not.toContain('操作')
  })
})

describe('ModuleListTab · 只读列表 Tab（FE-14）', () => {
  const roGroup = {
    path: '/ifm/remote-interfaces',
    type: 'group' as const,
    label: 'remote-interfaces',
    readonly: true,
    fields: [
      {
        path: '/ifm/remote-interfaces/remote-interface',
        type: 'list' as const,
        label: 'remote-interface',
        readonly: true,
        fields: [
          { path: '/ifm/remote-interfaces/remote-interface/index', type: 'string' as const, label: 'index', readonly: true, isKey: true },
          { path: '/ifm/remote-interfaces/remote-interface/port-name', type: 'string' as const, label: 'port-name', readonly: true },
        ],
      },
    ],
  }
  const roTab = deriveTabs([roGroup])[0]

  it('只读 Tab：无「新增」、无操作列，state 行数据照常可查看', async () => {
    vi.mocked(getConfig).mockResolvedValue({
      data: { data: { data: { 'remote-interface': [
        { index: '1', 'port-name': 'GE0/0/1' },
        { index: '2', 'port-name': 'GE0/0/2' },
      ] } } } } as any)
    const w = mount(ModuleListTab, {
      props: { tab: roTab, rootName: 'ifm', device: '10.0.0.1' },
      global: { plugins: [createPinia(), ElementPlus] },
    })
    await flushPromises()

    // 行数据照常渲染（可查看）
    expect(w.findAll('.el-table__body tr')).toHaveLength(2)
    expect(w.text()).toContain('GE0/0/1')
    // 无编辑/下发入口
    expect(w.text()).not.toContain('新增')
    const headers = w.findAll('.el-table__header th .cell').map((n) => n.text().trim())
    expect(headers).not.toContain('操作')
    expect(w.text()).not.toContain('编辑')
  })

  it('可编辑 Tab 不受影响：仍有「新增」与操作列', async () => {
    const w = mountTab()
    await flushPromises()
    expect(w.text()).toContain('新增')
    const headers = w.findAll('.el-table__header th .cell').map((n) => n.text().trim())
    expect(headers).toContain('操作')
  })
})

describe('ModuleListTab · 行删除（FE-16）', () => {
  it('门禁允许时删除按钮可用；确认后以行主键调 DELETE 并刷新', async () => {
    const w = mountTab()
    await flushPromises()

    const delBtn = w.findAll('.el-table__body .el-button').find((b) => b.text() === '删除')!
    expect(delBtn.attributes('disabled')).toBeUndefined()
    expect(delBtn.attributes('aria-disabled')).not.toBe('true')

    // 打桩确认框：直接确认
    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as any)
    vi.mocked(deleteConfig).mockResolvedValue({ data: { code: 0, success: true, data: {} } } as any)
    const loadsBefore = vi.mocked(getConfig).mock.calls.length

    await delBtn.trigger('click')
    await flushPromises()

    expect(confirmSpy).toHaveBeenCalled()
    expect(vi.mocked(deleteConfig)).toHaveBeenCalledTimes(1)
    const [ip, path, key] = vi.mocked(deleteConfig).mock.calls[0]
    expect(ip).toBe('10.0.0.1')
    expect(path).toContain('ifm:interfaces')
    expect(key).toBe('200GE0/1/0') // 首行主键（keyField=name）
    // 成功后刷新列表
    expect(vi.mocked(getConfig).mock.calls.length).toBeGreaterThan(loadsBefore)
    confirmSpy.mockRestore()
  })

  it('取消确认：零请求', async () => {
    const w = mountTab()
    await flushPromises()
    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm').mockRejectedValue('cancel')

    const delBtn = w.findAll('.el-table__body .el-button').find((b) => b.text() === '删除')!
    await delBtn.trigger('click')
    await flushPromises()

    expect(vi.mocked(deleteConfig)).not.toHaveBeenCalled()
    confirmSpy.mockRestore()
  })

  it('删除失败：错误如实可见、列表不变（R08/§9）', async () => {
    const w = mountTab()
    await flushPromises()
    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as any)
    vi.mocked(deleteConfig).mockRejectedValue({
      response: { data: { message: '设备删除失败: data-missing' } },
    })

    const delBtn = w.findAll('.el-table__body .el-button').find((b) => b.text() === '删除')!
    await delBtn.trigger('click')
    await flushPromises()

    expect(w.text()).toContain('data-missing')
    expect(w.findAll('.el-table__body tr')).toHaveLength(5)
    confirmSpy.mockRestore()
  })
})

// FE-18 二期（F2）：行删除命中归属硬锁 409 → 阻断确认 → force 重发 / 取消中止。
describe('ModuleListTab · 行删除归属硬锁 409', () => {
  const rejected409 = {
    data: { code: 409, success: false, message: '条目由业务意图管理', data: { intents: ['default/biz-100'] } },
  } as any

  it('409 → 确认覆盖 → 携 force 重发 DELETE', async () => {
    const w = mountTab()
    await flushPromises()
    const delBtn = w.findAll('.el-table__body .el-button').find((b) => b.text() === '删除')!
    // 两次确认框都点确认（删除确认 + 归属覆盖确认）
    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as any)
    vi.mocked(deleteConfig)
      .mockResolvedValueOnce(rejected409)
      .mockResolvedValueOnce({ data: { code: 0, success: true, data: {} } } as any)

    await delBtn.trigger('click')
    await flushPromises()

    expect(vi.mocked(deleteConfig)).toHaveBeenCalledTimes(2)
    expect(vi.mocked(deleteConfig).mock.calls[1][3]).toBe(true)
    // 归属确认框文案含认领意图
    const ownershipCall = confirmSpy.mock.calls.find((c) => String(c[0]).includes('default/biz-100'))
    expect(ownershipCall).toBeTruthy()
    expect(w.text()).not.toContain('条目由业务意图管理')
    confirmSpy.mockRestore()
  })

  it('409 → 取消覆盖 → 不重发、不置错误态', async () => {
    const w = mountTab()
    await flushPromises()
    const delBtn = w.findAll('.el-table__body .el-button').find((b) => b.text() === '删除')!
    // 第一次（删除确认）通过，第二次（归属覆盖）取消
    const confirmSpy = vi
      .spyOn(ElMessageBox, 'confirm')
      .mockResolvedValueOnce('confirm' as any)
      .mockRejectedValueOnce('cancel')
    vi.mocked(deleteConfig).mockResolvedValue(rejected409)

    await delBtn.trigger('click')
    await flushPromises()

    expect(vi.mocked(deleteConfig)).toHaveBeenCalledTimes(1)
    expect(w.find('.el-alert').exists()).toBe(false)
    confirmSpy.mockRestore()
  })
})

// force 重发失败（信封 success=false）→ 错误如实展示，不误报「已删除」（§9）。
describe('ModuleListTab · force 重发失败如实透出', () => {
  it('409 → 确认 → force DELETE 返回失败信封 → 展示错误', async () => {
    const w = mountTab()
    await flushPromises()
    const delBtn = w.findAll('.el-table__body .el-button').find((b) => b.text() === '删除')!
    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as any)
    vi.mocked(deleteConfig)
      .mockResolvedValueOnce({
        data: { code: 409, success: false, message: '条目由业务意图管理', data: { intents: ['default/biz-100'] } },
      } as any)
      .mockResolvedValueOnce({ data: { code: 502, success: false, message: '设备删除失败: data-missing' } } as any)

    await delBtn.trigger('click')
    await flushPromises()

    expect(vi.mocked(deleteConfig)).toHaveBeenCalledTimes(2)
    expect(w.text()).toContain('设备删除失败')
    expect(w.text()).not.toContain('已删除并触发对账')
    confirmSpy.mockRestore()
  })
})
