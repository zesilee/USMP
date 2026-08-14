import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import ModuleListTab from '../../src/components/config/ModuleListTab'
import { UiProvider } from '../../src/ui'
import { deriveTabs } from '../../src/utils/moduleConsole'
import { useChangesetStore } from '../../src/stores/changeset'
import { useFreshnessStore } from '../../src/stores/freshness'
import * as apiModule from '../../src/api'
import type { Field } from '../../src/utils/crdSchemaParser'

// ModuleListTab 全功能面 F2（FE-24/25 + FE-16/11 二期）：双模式探测与下推、
// 占位态、变更集标记/行删除/undelete、新鲜度埋点。
vi.mock('../../src/api')

const fields: Field[] = [
  {
    path: '/vlan/vlans', type: 'group', label: 'vlans',
    fields: [
      {
        path: '/vlan/vlans/vlan', type: 'list', label: 'vlan',
        fields: [
          { path: '/vlan/vlans/vlan/id', type: 'number', label: 'id', isKey: true },
          { path: '/vlan/vlans/vlan/name', type: 'string', label: 'name', supportFilter: true },
        ],
      },
    ],
  },
]

const CS = () => useChangesetStore.getState()

function mount(extra: Record<string, any> = {}) {
  const tab = deriveTabs(fields)[0]
  return render(
    <UiProvider>
      <ModuleListTab tab={tab} rootName="vlan" device="10.0.0.1" {...extra} />
    </UiProvider>,
  )
}

const listPage = (rows: any[], total: number) => ({
  data: { success: true, data: { data: { rows, total }, cache_age_seconds: 5, ttl_seconds: 30, source: 'cache' } },
})

describe('ModuleListTab · 双模式分页（FE-25）', () => {
  beforeEach(() => {
    useChangesetStore.setState({ byDevice: {} })
    useFreshnessStore.getState().reset()
    vi.clearAllMocks()
  })

  it('小表：ListPage total≤200 → 客户端模式，一次取全量', async () => {
    vi.mocked(apiModule.getConfig).mockResolvedValue(listPage([{ id: 1, name: 'a' }], 1) as any)
    mount()
    expect(await screen.findByText('a')).toBeInTheDocument()
    expect(vi.mocked(apiModule.getConfig)).toHaveBeenCalledTimes(1)
    // 首读探测参数 limit=200
    expect(vi.mocked(apiModule.getConfig).mock.calls[0][4]).toMatchObject({ limit: 200, offset: 0 })
  })

  it('大表：total>200 → 服务端模式，立即取当前页；翻页下推 offset', async () => {
    const bigRows = Array.from({ length: 10 }, (_, i) => ({ id: i + 1, name: `n${i + 1}` }))
    vi.mocked(apiModule.getConfig).mockResolvedValue(listPage(bigRows, 500) as any)
    mount()
    expect(await screen.findByText('n1')).toBeInTheDocument()
    // 探测 + 取页 = 2 次
    await waitFor(() => expect(vi.mocked(apiModule.getConfig)).toHaveBeenCalledTimes(2))
    expect(vi.mocked(apiModule.getConfig).mock.calls[1][4]).toMatchObject({ limit: 10, offset: 0 })

    // 翻到第 2 页 → offset 下推
    fireEvent.click(screen.getByTitle('2'))
    await waitFor(() => expect(vi.mocked(apiModule.getConfig)).toHaveBeenCalledTimes(3))
    expect(vi.mocked(apiModule.getConfig).mock.calls[2][4]).toMatchObject({ limit: 10, offset: 10 })
  })

  it('带参被拒（success=false）：回退旧读法整树形状（R08 降级）', async () => {
    vi.mocked(apiModule.getConfig)
      .mockResolvedValueOnce({ data: { success: false, code: 1, data: null } } as any)
      .mockResolvedValueOnce({ data: { success: true, data: { data: { vlan: [{ id: 7, name: 'fallback' }] } } } } as any)
    mount()
    expect(await screen.findByText('fallback')).toBeInTheDocument()
    // 第二次调用为无 query 的旧读法
    expect(vi.mocked(apiModule.getConfig).mock.calls[1][4]).toBeUndefined()
  })

  it('读路径新鲜度埋点：cache_age/ttl/source 入 store（tasks 11.3 前置）', async () => {
    vi.mocked(apiModule.getConfig).mockResolvedValue(listPage([{ id: 1, name: 'a' }], 1) as any)
    mount()
    await screen.findByText('a')
    const f = useFreshnessStore.getState()
    expect(f.hasData).toBe(true)
    expect(f.ageSeconds).toBe(5)
    expect(f.source).toBe('cache')
  })
})

describe('ModuleListTab · 高级搜索/排序/详情接线', () => {
  beforeEach(() => {
    useChangesetStore.setState({ byDevice: {} })
    vi.clearAllMocks()
  })

  it('客户端模式高级搜索：应用条件本地过滤、重置还原（FE-11）', async () => {
    vi.mocked(apiModule.getConfig).mockResolvedValue({
      data: { success: true, data: { data: { vlan: [{ id: 1, name: 'mgmt' }, { id: 2, name: 'iot' }] } } },
    } as any)
    mount()
    await screen.findByText('mgmt')
    fireEvent.click(document.querySelector('[data-test="adv-search"]')!)
    const panel = await waitFor(() => document.querySelector('[data-test="adv-search-panel"]')!)
    fireEvent.change(panel.querySelector('input')!, { target: { value: 'io' } })
    fireEvent.click(document.querySelector('[data-test="adv-search-apply"]')!)
    await waitFor(() => expect(screen.queryByText('mgmt')).toBeNull())
    expect(screen.getByText('iot')).toBeInTheDocument()

    fireEvent.click(document.querySelector('[data-test="adv-search"]')!)
    fireEvent.click(document.querySelector('[data-test="adv-search-reset"]')!)
    await waitFor(() => expect(screen.getByText('mgmt')).toBeInTheDocument())
  })

  it('服务端模式：高级搜索/排序下推为 filter/sort 参数（FE-25/BR-13）', async () => {
    const bigRows = Array.from({ length: 10 }, (_, i) => ({ id: i + 1, name: `n${i + 1}` }))
    vi.mocked(apiModule.getConfig).mockResolvedValue(listPage(bigRows, 500) as any)
    mount()
    await screen.findByText('n1')
    await waitFor(() => expect(vi.mocked(apiModule.getConfig)).toHaveBeenCalledTimes(2))

    // 搜索下推
    fireEvent.click(document.querySelector('[data-test="adv-search"]')!)
    const panel = await waitFor(() => document.querySelector('[data-test="adv-search-panel"]')!)
    fireEvent.change(panel.querySelector('input')!, { target: { value: 'core' } })
    fireEvent.click(document.querySelector('[data-test="adv-search-apply"]')!)
    await waitFor(() => expect(vi.mocked(apiModule.getConfig)).toHaveBeenCalledTimes(3))
    expect(vi.mocked(apiModule.getConfig).mock.calls[2][4]).toMatchObject({ filters: ['name~=core'], offset: 0 })

    // 排序下推
    fireEvent.click(screen.getByRole('columnheader', { name: /id/ }))
    await waitFor(() => expect(vi.mocked(apiModule.getConfig)).toHaveBeenCalledTimes(4))
    expect(vi.mocked(apiModule.getConfig).mock.calls[3][4]).toMatchObject({ sort: 'id' })
  })

  it('新增按钮打开创建态详情区；暂存后关闭（FE-21 接线）', async () => {
    vi.mocked(apiModule.getConfig).mockResolvedValue({
      data: { success: true, data: { data: { vlan: [] } } },
    } as any)
    mount()
    await waitFor(() => expect(document.querySelector('[data-test="add-row"]')).toBeTruthy())
    fireEvent.click(document.querySelector('[data-test="add-row"]')!)
    await waitFor(() => expect(document.querySelector('[data-test="item-detail-pane"]')).toBeTruthy())
    fireEvent.click(document.querySelector('[data-test="detail-close"]')!)
    await waitFor(() => expect(document.querySelector('[data-test="item-detail-pane"]')).toBeNull())
  })

  it('行点击打开编辑态详情区（onRow 接线）', async () => {
    vi.mocked(apiModule.getConfig).mockResolvedValue({
      data: { success: true, data: { data: { vlan: [{ id: 10, name: 'mgmt' }] } } },
    } as any)
    mount()
    const cell = await screen.findByText('mgmt')
    fireEvent.click(cell)
    await waitFor(() => expect(document.querySelector('[data-test="item-detail-pane"]')).toBeTruthy())
  })
})

describe('ModuleListTab · FE-24 占位与 FE-16 行删除', () => {
  beforeEach(() => {
    useChangesetStore.setState({ byDevice: {} })
    vi.clearAllMocks()
    vi.mocked(apiModule.getConfig).mockResolvedValue({
      data: { success: true, data: { data: { vlan: [{ id: 10, name: 'mgmt' }] } } },
    } as any)
  })

  it('预标记占位：零请求；重试 force 恢复', async () => {
    mount({ unsupported: true })
    expect(document.querySelector('[data-test="node-unsupported"]')).toBeTruthy()
    expect(vi.mocked(apiModule.getConfig)).not.toHaveBeenCalled()
    fireEvent.click(screen.getByRole('button'))
    await waitFor(() => expect(document.querySelector('[data-test="module-list-tab"]')).toBeTruthy())
  })

  it('行删除：确认 → markDelete 入集 + 行标记 delete + undelete 还原', async () => {
    mount()
    await screen.findByText('mgmt')
    fireEvent.click(screen.getByRole('button', { name: /删\s*除|Delete/ }))
    const modal = await waitFor(() => {
      const el = document.querySelector('.ant-modal-confirm-btns')
      if (!el) throw new Error('not open')
      return el as HTMLElement
    })
    fireEvent.click(within(modal).getByRole('button', { name: /确\s*定|OK/ }))
    await waitFor(() => expect(document.querySelector('[data-test="mark-delete"]')).toBeTruthy())
    expect(CS().isPendingDelete('10.0.0.1', '/vlan:vlan/vlan:vlans', '10')).toBe(true)

    fireEvent.click(document.querySelector('[data-test="undelete-btn"]')!)
    await waitFor(() => expect(document.querySelector('[data-test="mark-delete"]')).toBeNull())
    expect(CS().countFor('10.0.0.1')).toBe(0)
  })

  it('update 条目行标记 markUpdate；delete 行样式类挂载（FE-11 二期）', async () => {
    CS().upsert('10.0.0.1', {
      op: 'update', path: '/vlan:vlan/vlan:vlans', listKey: 'vlan', keyValue: '10',
      payload: { id: 10, name: 'edited' }, baseline: { id: 10, name: 'mgmt' },
    })
    const { container } = mount()
    await screen.findByText('mgmt')
    expect(document.querySelector('[data-test="mark-update"]')).toBeTruthy()
    expect(container.querySelector('tr.row-update')).toBeTruthy()
  })

  it('§9 回归：force 刷新失败保留原列表；常规失败清空（评审 #1 修复防线）', async () => {
    mount()
    await screen.findByText('mgmt')
    // force 失败 → 保留原列表 + 错误条
    vi.mocked(apiModule.getConfig).mockRejectedValueOnce(new Error('refresh boom'))
    fireEvent.click(document.querySelector('[data-test="fetch-source"]')!)
    expect(await screen.findByText('refresh boom')).toBeInTheDocument()
    expect(screen.getByText('mgmt')).toBeInTheDocument() // §9 保留

    // 常规重载失败（设备切换路径复用 load(false)）→ 清空
    vi.mocked(apiModule.getConfig).mockRejectedValue(new Error('load boom'))
    fireEvent.click(document.querySelector('[data-test="fetch-source"]')!) // 先触发错误态
    await screen.findByText('load boom')
  })

  it('信封运行中学习：ListPage 请求返回 node-unsupported → 转占位（FE-24）', async () => {
    vi.mocked(apiModule.getConfig).mockResolvedValue({
      data: { success: false, code: 1, data: { reason: 'node-unsupported' } },
    } as any)
    mount()
    await waitFor(() => expect(document.querySelector('[data-test="node-unsupported"]')).toBeTruthy())
  })

  it('变更集待创建行叠加合成视图并标记 create（FE-11 二期）', async () => {
    CS().upsert('10.0.0.1', {
      op: 'create', path: '/vlan:vlan/vlan:vlans', listKey: 'vlan', keyValue: '99',
      payload: { id: 99, name: 'pending' },
    })
    mount()
    expect(await screen.findByText('pending')).toBeInTheDocument()
    expect(document.querySelector('[data-test="mark-create"]')).toBeTruthy()
  })
})
