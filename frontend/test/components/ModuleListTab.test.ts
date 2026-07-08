import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import ModuleListTab from '../../src/components/config/ModuleListTab.vue'
import { getConfig, setConfig, getDeviceReconcile } from '../../src/api'
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

  it('list 级无 exclude → 编辑按钮可见；删除按钮存在但禁用（后端暂无删除语义）', async () => {
    const w = mountTab()
    await flushPromises()
    const ops = w.findAll('.el-table__body .el-button')
    const texts = ops.map((b) => b.text().trim())
    expect(texts).toContain('编辑')
    expect(texts).toContain('删除')
    const del = ops.find((b) => b.text().trim() === '删除')!
    expect(del.attributes('disabled')).toBeDefined()
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
