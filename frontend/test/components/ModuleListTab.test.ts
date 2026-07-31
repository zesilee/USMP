import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import ElementPlus, { ElMessageBox } from 'element-plus'
import ModuleListTab from '../../src/components/config/ModuleListTab.vue'
import ItemDetailPane from '../../src/components/config/ItemDetailPane.vue'
import { getConfig, deleteConfig, setConfig, getDeviceReconcile } from '../../src/api'
import { useChangesetStore } from '../../src/stores/changeset'
import { deriveTabs } from '../../src/utils/moduleConsole'
import { ifmNestedSchema, seedRows } from '../views/moduleConsole.fixture'

vi.mock('../../src/api')

const interfacesTab = deriveTabs(ifmNestedSchema.fields).find((t) => t.name === 'interfaces')!

let pinia: Pinia

function mountTab() {
  return mount(ModuleListTab, {
    props: { tab: interfacesTab, rootName: 'ifm', device: '10.0.0.1' },
    global: { plugins: [pinia, ElementPlus] },
  })
}

beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
  vi.resetAllMocks()
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
    expect(w.findAll('.el-table__body .el-tag').length).toBeGreaterThan(0)
    expect(w.findAll('.status-cell.ok')).toHaveLength(3)
    expect(w.findAll('.status-cell.bad')).toHaveLength(2)
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

describe('ModuleListTab · 工具区（FE-11：创建/刷新/时间戳/多选/分页）', () => {
  it('工具栏含「创建」「刷新」按钮；表格含多选框列', async () => {
    const w = mountTab()
    await flushPromises()
    expect(w.find('[data-test="list-create"]').text()).toContain('创建')
    expect(w.find('[data-test="list-refresh"]').exists()).toBe(true)
    expect(w.find('.el-table-column--selection').exists()).toBe(true)
  })

  it('查询时间戳与总记录数：加载完成后展示「查询结束，总记录数: 5」', async () => {
    const w = mountTab()
    await flushPromises()
    const line = w.find('[data-test="query-summary"]').text()
    expect(line).toContain('查询结束')
    expect(line).toContain('5')
  })

  it('点「刷新」重新拉取列表（非强制）', async () => {
    const w = mountTab()
    await flushPromises()
    const before = vi.mocked(getConfig).mock.calls.length
    await w.find('[data-test="list-refresh"]').trigger('click')
    await flushPromises()
    expect(vi.mocked(getConfig).mock.calls.length).toBe(before + 1)
    expect(vi.mocked(getConfig).mock.calls.at(-1)![2]).toBeFalsy()
  })

  it('分页含跳页（jumper）；pageSize 缩小生效', async () => {
    const w = mountTab()
    await flushPromises()
    expect(w.find('.el-pagination__jump').exists()).toBe(true)
    const vm = w.vm as any
    vm.pageSize = 2
    await flushPromises()
    expect(w.findAll('.el-table__body tr')).toHaveLength(2)
  })
})

describe('ModuleListTab · 列设置/排序/列头筛选（FE-11）', () => {
  it('列设置入口存在；取消勾选某列后该列隐藏', async () => {
    const w = mountTab()
    await flushPromises()
    expect(w.find('[data-test="column-settings"]').exists()).toBe(true)
    const vm = w.vm as any
    vm.visibleCols = vm.visibleCols.filter((p: string) => !p.endsWith('/description'))
    await flushPromises()
    const headers = w.findAll('.el-table__header th .cell').map((n) => n.text().trim())
    expect(headers).not.toContain('description')
    expect(headers).toContain('name')
  })

  it('数据列可排序（caret 存在）；enum 列有列头筛选入口', async () => {
    const w = mountTab()
    await flushPromises()
    expect(w.findAll('.el-table__header .caret-wrapper').length).toBeGreaterThan(0)
    expect(w.findAll('.el-table__header .el-table__column-filter-trigger').length).toBeGreaterThan(0)
  })
})

describe('ModuleListTab · master-detail 详情区（FE-21）', () => {
  it('点行「编辑」→ 详情区展开、编辑态、行主键入面包屑；无抽屉', async () => {
    const w = mountTab()
    await flushPromises()
    const edit = w.findAll('.el-table__body .el-button').find((b) => b.text() === '编辑')!
    await edit.trigger('click')
    await flushPromises()
    expect(w.find('[data-test="item-detail-pane"]').exists()).toBe(true)
    expect(w.find('[data-test="detail-breadcrumb"]').text()).toContain('200GE0/1/0')
    expect(w.find('.el-drawer').exists()).toBe(false)
  })

  it('点击行本体同样打开详情（行高亮）', async () => {
    const w = mountTab()
    await flushPromises()
    await w.findAll('.el-table__body tr')[1].trigger('click')
    await flushPromises()
    expect(w.find('[data-test="item-detail-pane"]').exists()).toBe(true)
    expect(w.find('[data-test="detail-breadcrumb"]').text()).toContain('200GE0/1/1')
    expect(w.find('.el-table__body tr.current-row').exists()).toBe(true)
  })

  it('「创建」→ 详情区创建态空表单', async () => {
    const w = mountTab()
    await flushPromises()
    await w.find('[data-test="list-create"]').trigger('click')
    await flushPromises()
    expect(w.find('[data-test="item-detail-pane"]').exists()).toBe(true)
    expect(w.find('[data-test="detail-breadcrumb"]').text()).toContain('创建')
  })

  it('详情区「关闭」→ 收起且列表状态不变', async () => {
    const w = mountTab()
    await flushPromises()
    await w.findAll('.el-table__body tr')[0].trigger('click')
    await flushPromises()
    await w.find('[data-test="detail-close"]').trigger('click')
    await flushPromises()
    expect(w.find('[data-test="item-detail-pane"]').exists()).toBe(false)
    expect(w.findAll('.el-table__body tr')).toHaveLength(5)
  })

  it('未提交草稿切行 → 确认框；取消停留原条目（FE-21 负路径）', async () => {
    const w = mountTab()
    await flushPromises()
    await w.findAll('.el-table__body tr')[0].trigger('click')
    await flushPromises()
    const pane = w.findComponent(ItemDetailPane)
    ;(pane.vm as any).form.formData['description'] = 'draft'
    await flushPromises()

    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm').mockRejectedValue('cancel')
    await w.findAll('.el-table__body tr')[2].trigger('click')
    await flushPromises()
    expect(confirmSpy).toHaveBeenCalled()
    expect(w.find('[data-test="detail-breadcrumb"]').text()).toContain('200GE0/1/0')
    confirmSpy.mockRestore()
  })

  it('无草稿切行 → 直接切换不弹确认', async () => {
    const w = mountTab()
    await flushPromises()
    await w.findAll('.el-table__body tr')[0].trigger('click')
    await flushPromises()
    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm')
    await w.findAll('.el-table__body tr')[2].trigger('click')
    await flushPromises()
    expect(confirmSpy).not.toHaveBeenCalled()
    expect(w.find('[data-test="detail-breadcrumb"]').text()).toContain('200GE0/1/2')
    confirmSpy.mockRestore()
  })
})

describe('ModuleListTab · 获取数据源（FE-11 force_refresh）', () => {
  it('行操作「获取数据源」→ forceRefresh=true 回读并刷新时间戳', async () => {
    const w = mountTab()
    await flushPromises()
    const fetchBtn = w.findAll('.el-table__body .el-button').find((b) => b.text() === '获取数据源')!
    await fetchBtn.trigger('click')
    await flushPromises()
    const last = vi.mocked(getConfig).mock.calls.at(-1)!
    expect(last[1]).toContain('ifm:interfaces')
    expect(last[2]).toBe(true)
  })

  it('获取数据源失败 → 错误如实展示、列表保持原状（R08/§9）', async () => {
    const w = mountTab()
    await flushPromises()
    vi.mocked(getConfig).mockRejectedValueOnce({ response: { data: { message: '设备离线' } } })
    const fetchBtn = w.findAll('.el-table__body .el-button').find((b) => b.text() === '获取数据源')!
    await fetchBtn.trigger('click')
    await flushPromises()
    expect(w.text()).toContain('设备离线')
    expect(w.findAll('.el-table__body tr')).toHaveLength(5)
  })
})

describe('ModuleListTab · 高级搜索（support-filter 驱动，FE-11）', () => {
  it('面板默认折叠，点击「高级搜索」展开；字段集仅 supportFilter 叶（class/type）', async () => {
    const w = mountTab()
    await flushPromises()
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

  it('只读 Tab：无「创建」、无操作列、点行不开详情编辑区', async () => {
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
    expect(w.findAll('.el-table__body tr')).toHaveLength(2)
    expect(w.text()).toContain('GE0/0/1')
    expect(w.find('[data-test="list-create"]').exists()).toBe(false)
    const headers = w.findAll('.el-table__header th .cell').map((n) => n.text().trim())
    expect(headers).not.toContain('操作')
    await w.findAll('.el-table__body tr')[0].trigger('click')
    await flushPromises()
    expect(w.find('[data-test="item-detail-pane"]').exists()).toBe(false)
  })

  it('可编辑 Tab 不受影响：仍有「创建」与操作列', async () => {
    const w = mountTab()
    await flushPromises()
    expect(w.find('[data-test="list-create"]').exists()).toBe(true)
    const headers = w.findAll('.el-table__header th .cell').map((n) => n.text().trim())
    expect(headers).toContain('操作')
  })
})

describe('ModuleListTab · 行删除入变更集（FE-16 攒批）', () => {
  it('确认后零请求：入变更集删除项、行现待删除标记、按钮变取消删除', async () => {
    const w = mountTab()
    await flushPromises()
    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as any)

    const delBtn = w.findAll('.el-table__body .el-button').find((b) => b.text() === '删除')!
    await delBtn.trigger('click')
    await flushPromises()

    expect(vi.mocked(deleteConfig)).not.toHaveBeenCalled()
    const cs = useChangesetStore()
    expect(cs.isPendingDelete('10.0.0.1', '/ifm:ifm/ifm:interfaces', '200GE0/1/0')).toBe(true)
    expect(w.find('[data-test="mark-delete"]').exists()).toBe(true)
    expect(w.find('[data-test="undelete-btn"]').exists()).toBe(true)
    confirmSpy.mockRestore()
  })

  it('取消删除：删除项移除、标记还原（FE-16）', async () => {
    const w = mountTab()
    await flushPromises()
    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as any)
    await w.findAll('.el-table__body .el-button').find((b) => b.text() === '删除')!.trigger('click')
    await flushPromises()

    await w.find('[data-test="undelete-btn"]').trigger('click')
    await flushPromises()
    expect(useChangesetStore().countFor('10.0.0.1')).toBe(0)
    expect(w.find('[data-test="mark-delete"]').exists()).toBe(false)
    confirmSpy.mockRestore()
  })

  it('取消确认：变更集零改动', async () => {
    const w = mountTab()
    await flushPromises()
    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm').mockRejectedValue('cancel')

    const delBtn = w.findAll('.el-table__body .el-button').find((b) => b.text() === '删除')!
    await delBtn.trigger('click')
    await flushPromises()

    expect(useChangesetStore().countFor('10.0.0.1')).toBe(0)
    confirmSpy.mockRestore()
  })

  it('删除按钮点击不触发行点击开详情（事件不冒泡）', async () => {
    const w = mountTab()
    await flushPromises()
    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm').mockRejectedValue('cancel')
    const delBtn = w.findAll('.el-table__body .el-button').find((b) => b.text() === '删除')!
    await delBtn.trigger('click')
    await flushPromises()
    expect(w.find('[data-test="item-detail-pane"]').exists()).toBe(false)
    confirmSpy.mockRestore()
  })

  it('删除待创建条目 = 直接移除且不产生删除项（FE-16 边界）', async () => {
    const w = mountTab()
    await flushPromises()
    const cs = useChangesetStore()
    cs.upsert('10.0.0.1', {
      op: 'create',
      path: '/ifm:ifm/ifm:interfaces',
      listKey: 'interface',
      keyValue: 'GE-NEW',
      payload: { name: 'GE-NEW', class: 'main-interface' },
      cleared: [],
      baseline: null,
      label: 'interface GE-NEW',
    })
    await flushPromises()
    // 合成视图：待创建行出现且带标记
    expect(w.text()).toContain('GE-NEW')
    expect(w.find('[data-test="mark-create"]').exists()).toBe(true)

    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as any)
    const rowBtns = w.findAll('.el-table__body .el-button').filter((b) => b.text() === '删除')
    await rowBtns[rowBtns.length - 1].trigger('click') // 待创建行在合成视图末尾
    await flushPromises()

    expect(cs.countFor('10.0.0.1')).toBe(0)
    expect(w.text()).not.toContain('GE-NEW')
    confirmSpy.mockRestore()
  })

  it('批量删除：多选 → 更多▾ → 确认 → 逐条入集（FE-11 二期）', async () => {
    const w = mountTab()
    await flushPromises()
    const vm = w.vm as any
    // 直接驱动 selection 状态（checkbox 交互属 F3 真浏览器域）
    vm.onSelectionChange([{ ...seedRows[0] }, { ...seedRows[1] }])
    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as any)
    await vm.onBatchCommand('batch-delete')
    await flushPromises()

    const cs = useChangesetStore()
    expect(cs.countFor('10.0.0.1')).toBe(2)
    expect(cs.isPendingDelete('10.0.0.1', '/ifm:ifm/ifm:interfaces', '200GE0/1/0')).toBe(true)
    expect(cs.isPendingDelete('10.0.0.1', '/ifm:ifm/ifm:interfaces', '200GE0/1/1')).toBe(true)
    confirmSpy.mockRestore()
  })

  it('修改标记：变更集含该行 update 条目 → 已修改标记（合成视图）', async () => {
    const w = mountTab()
    await flushPromises()
    useChangesetStore().upsert('10.0.0.1', {
      op: 'update',
      path: '/ifm:ifm/ifm:interfaces',
      listKey: 'interface',
      keyValue: '200GE0/1/0',
      payload: { name: '200GE0/1/0', description: 'x' },
      cleared: [],
      baseline: { ...seedRows[0] },
      label: 'interface 200GE0/1/0',
    })
    await flushPromises()
    expect(w.find('[data-test="mark-update"]').exists()).toBe(true)
  })
})
