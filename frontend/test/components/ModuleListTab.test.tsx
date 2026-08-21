import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import ModuleListTab, { normalizeRows } from '../../src/components/config/ModuleListTab'
import { UiProvider } from '../../src/ui'
import { deriveTabs, deriveColumns } from '../../src/utils/moduleConsole'
import * as apiModule from '../../src/api'
import type { Field } from '../../src/utils/crdSchemaParser'

// ModuleListTab F2（FE-11 切片）：Table 运行时动态列——列由 schema 派生现场生成、
// 单元格按列元数据分派、表头筛选/排序/多选/列设置全走列配置。零 per-module 代码。
vi.mock('../../src/api')

const vlanFields: Field[] = [
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
          { path: '/vlan/vlans/vlan/id', type: 'number', label: 'id', isKey: true },
          { path: '/vlan/vlans/vlan/name', type: 'string', label: 'name' },
          {
            path: '/vlan/vlans/vlan/type', type: 'enum', label: 'type',
            options: [{ label: 'common', value: 'common' }, { label: 'dynamic', value: 'dynamic' }],
          },
          { path: '/vlan/vlans/vlan/enable', type: 'boolean', label: 'enable' },
          { path: '/vlan/vlans/vlan/status', type: 'string', label: 'status', readonly: true },
        ],
      },
    ],
  },
]

const rows = [
  { id: 10, name: 'mgmt', type: 'common', enable: true, status: 'up' },
  { id: 20, name: 'iot', type: 'dynamic', enable: false, status: 'down' },
]

function mockConfig(payload: any) {
  vi.mocked(apiModule.getConfig).mockResolvedValue({ data: { success: true, data: payload } } as any)
}

function mountTab() {
  const tab = deriveTabs(vlanFields)[0]
  return render(
    <UiProvider>
      <ModuleListTab tab={tab} rootName="vlan" device="10.0.0.1" />
    </UiProvider>,
  )
}

describe('ModuleListTab · 运行时动态列（R05 闸门）', () => {
  beforeEach(() => vi.clearAllMocks())

  it('列由 schema 派生现场生成（与 deriveColumns 结论一致），行数据渲染', async () => {
    mockConfig({ vlan: rows })
    mountTab()
    const tab = deriveTabs(vlanFields)[0]
    const expected = deriveColumns(tab.listField!).map((c) => c.label)
    for (const label of expected) {
      expect(await screen.findByRole('columnheader', { name: new RegExp(label) })).toBeInTheDocument()
    }
    expect(await screen.findByText('mgmt')).toBeInTheDocument()
    expect(screen.getByText('iot')).toBeInTheDocument()
  })

  it('单元格分派：enum→Tag、boolean→Tag、up/down→状态点', async () => {
    mockConfig({ vlan: rows })
    const { container } = mountTab()
    await screen.findByText('mgmt')
    expect(screen.getByText('common').closest('.ant-tag')).toBeTruthy()
    expect(screen.getByText('true').closest('.ant-tag')).toBeTruthy()
    expect(container.querySelector('.status-cell.ok')).toBeTruthy()
    expect(container.querySelector('.status-cell.bad')).toBeTruthy()
  })

  it('enum 表头筛选按选项集生成并可过滤行', async () => {
    mockConfig({ vlan: rows })
    const { container } = mountTab()
    await screen.findByText('mgmt')
    const trigger = container.querySelectorAll('.ant-table-filter-trigger')
    expect(trigger.length).toBeGreaterThan(0)
    fireEvent.click(trigger[0])
    fireEvent.click(await screen.findByText('dynamic', { selector: '.ant-dropdown *' }))
    // 确认钮文案随 locale（zh「确 定」/en「OK」），按钮类名定位更稳。
    fireEvent.click(document.querySelector('.ant-table-filter-dropdown-btns .ant-btn-primary')!)
    await waitFor(() => expect(screen.queryByText('mgmt')).toBeNull())
    expect(screen.getByText('iot')).toBeInTheDocument()
  })

  it('列设置：取消勾选后该列消失（可用列全集≠默认集）', async () => {
    mockConfig({ vlan: rows })
    mountTab()
    await screen.findByText('mgmt')
    fireEvent.click(document.querySelector('[data-test="column-settings"]')!)
    const panel = await waitFor(() => document.querySelector('[data-test="column-settings-panel"]')!)
    const nameBox = Array.from(panel.querySelectorAll('label')).find((l) => l.textContent?.includes('name'))!
    fireEvent.click(nameBox.querySelector('input')!)
    await waitFor(() =>
      expect(screen.queryByRole('columnheader', { name: /name/ })).toBeNull(),
    )
  })

  it('列头点击排序：数值列按数值序（非字典序）', async () => {
    mockConfig({ vlan: [...rows, { id: 3, name: 'z3', type: 'common', enable: true, status: 'up' }] })
    mountTab()
    await screen.findByText('z3')
    fireEvent.click(screen.getByRole('columnheader', { name: /id/ }))
    await waitFor(() => {
      // 列序：selection(1) id(2)——标记列改为仅攒批有标记时插位（NCE 空列对齐），
      // 无变更场景 id 在第 2 列。
      const cells = screen.getAllByRole('row').slice(1).map((r) => r.querySelector('td:nth-child(2)')?.textContent)
      expect(cells).toEqual(['3', '10', '20']) // 数值序：3<10<20（字典序会是 10,20,3）
    })
    // 字符串列排序路径
    fireEvent.click(screen.getByRole('columnheader', { name: /name/ }))
  })

  it('多选可用；获取数据源触发 force_refresh 重取', async () => {
    mockConfig({ vlan: rows })
    mountTab()
    await screen.findByText('mgmt')
    const checkboxes = screen.getAllByRole('checkbox')
    fireEvent.click(checkboxes[1])
    expect((checkboxes[1] as HTMLInputElement).checked).toBe(true)

    fireEvent.click(document.querySelector('[data-test="fetch-source"]')!)
    await waitFor(() =>
      // 全功能形态带 includeState/query 参（FE-25 探测），force=true 为第 3 参。
      expect(vi.mocked(apiModule.getConfig)).toHaveBeenLastCalledWith(
        '10.0.0.1', expect.any(String), true, false, expect.anything(),
      ),
    )
  })

  it('无攒批变更时不渲染 __mark__ 标记列（NCE 对齐：无空列）', async () => {
    mockConfig({ vlan: rows })
    mountTab()
    await screen.findByText('mgmt')
    const headTexts = Array.from(document.querySelectorAll('th')).map((th) => th.textContent?.trim())
    // 首个非勾选列应为数据列（VLAN 标识），不存在无标题空列
    expect(headTexts.filter((t2) => t2 === '').length).toBeLessThanOrEqual(1) // 仅勾选列头
    expect(document.querySelector('[data-test="mark-create"]')).toBeNull()
  })

  it('行内获取数据源触发 force_refresh（NCE 操作列对齐）', async () => {
    mockConfig({ vlan: rows })
    mountTab()
    await screen.findByText('mgmt')
    fireEvent.click(document.querySelectorAll('[data-test="row-fetch"]')[0]!)
    await waitFor(() =>
      expect(vi.mocked(apiModule.getConfig)).toHaveBeenLastCalledWith(
        '10.0.0.1', expect.any(String), true, false, expect.anything(),
      ),
    )
  })

  it('数据列统一宽度（NCE 等宽对齐）', async () => {
    const { buildDataColumns } = await import('../../src/components/config/listColumns')
    const cols = buildDataColumns(
      [
        { path: 'a', label: 'A', type: 'string' },
        { path: 'b', label: 'B', type: 'boolean' },
      ] as never,
      false,
    )
    expect(cols.every((c: { width?: number }) => c.width === 160)).toBe(true)
  })

  it('取数失败：错误条展示、表格空态（R08 负路径）', async () => {
    vi.mocked(apiModule.getConfig).mockRejectedValue(new Error('device offline'))
    mountTab()
    expect(await screen.findByText('device offline')).toBeInTheDocument()
  })
})

describe('normalizeRows · 回读子树剥形（沿用契约）', () => {
  const tab = deriveTabs(vlanFields)[0]
  const lf = tab.listField!

  it('list 数组直取', () => {
    expect(normalizeRows({ vlan: rows }, lf, tab.field, 'id').rows).toHaveLength(2)
  })

  it('容器键控对象 → 键并入 keyField 列（数字键转数字）', () => {
    const r = normalizeRows({ vlan: { '10': { name: 'a' }, '20': { name: 'b' } } }, lf, tab.field, 'id')
    expect(r.rows).toEqual([
      { id: 10, name: 'a' },
      { id: 20, name: 'b' },
    ])
  })

  it('回退 tab 名键；无命中空行（症状速查锚点：一行且主键列=list 名 = 上游剥层坏）', () => {
    expect(normalizeRows({ vlans: rows }, lf, tab.field, 'id').rows).toHaveLength(2)
    expect(normalizeRows({}, lf, tab.field, 'id').rows).toEqual([])
    expect(normalizeRows(undefined, lf, tab.field, 'id').rows).toEqual([])
  })
})
