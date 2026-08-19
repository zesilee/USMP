import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent, cleanup } from '@testing-library/react'

// 组 4.3 结构组桥 F2（替身规格 = vendor d.ts + matrix + gate 实测）。
const H = vi.hoisted(() => {
  const recv = { last: {} as Record<string, any> }
  const makeStub =
    (name: string, extra?: (p: any, h: any) => any) =>
    async () => {
      const { createElement: h } = await import('react')
      const Comp = (p: any) => {
        recv.last[name] = p
        return h('div', { 'data-stub': name, id: p.id }, extra?.(p, h), p.children)
      }
      return { default: Comp, TabItem: (p: any) => h('span', { 'data-tabitem': p.title, 'data-disabled': p.disabled ? '1' : undefined }) }
    }
  return { recv, makeStub }
})
const recv = H.recv

vi.mock('@nce/eview-react/Tab', H.makeStub('Tab', (p, h) =>
  h('button', { onClick: () => p.onClick?.(1, 't', {}) }, 'click-tab2'),
))
vi.mock('@nce/eview-react/Tree', H.makeStub('Tree', (p, h) =>
  h('div', null,
    h('button', { onClick: () => p.onExpand?.(['a', 'a-1'], { id: 'a-1' }) }, 'expand'),
    h('button', { onClick: () => p.onSelect?.(['leaf'], { id: 'leaf' }) }, 'select-leaf'),
  ),
))
vi.mock('@nce/eview-react/Table', H.makeStub('Table', (p, h) =>
  h('div', null,
    h('button', { onClick: () => p.onRowCheck?.({ __ubkey: 'r1' }, [0]) }, 'check-r1'), // R9 实测：回传行序号
    h('button', { onClick: () => p.onHeaderCheck?.([0, 1], true, []) }, 'check-all'), // R9 实测：回传行序号
    h('button', { onClick: () => p.onRowClick?.(p.dataset?.[0]) }, 'row1'),
    h('button', { onClick: () => p.onColumnSort?.({ key: 'mtu' }, 'desc') }, 'sort-desc'),
    h('button', { onClick: () => p.onPageChange?.(3) }, 'page3'),
    // 自定义 render 探针：以 eview 参数序调用列 render。
    h('output', null, String(p.columns?.[1]?.render?.('v1', null, null, p.dataset?.[0]) ?? '')),
  ),
))

import { Tabs, Menu, Table } from '@bridge/components/structure'

afterEach(() => {
  cleanup()
  recv.last = {}
})

// 组 7 终局：标签栏自绘（eview Tab 可见窗口/折叠 index 错位/cWRP 三连）——
// 断言改真实 DOM（role=tab 语义为自绘补回）。
describe('Tabs 桥（自绘标签栏 + 自渲内容区）', () => {
  const items = [
    { key: 'a', label: 'Tab 甲', children: <p>content-a</p> },
    { key: 'b', label: 'Tab 乙', children: <p>content-b</p>, disabled: false },
  ]
  it('activeKey 激活态、role=tab 语义、仅渲染激活内容', () => {
    const { container } = render(<Tabs items={items} activeKey="b" onChange={() => {}} data-test="console-tabs" />)
    const tabs = container.querySelectorAll('[role="tab"]')
    expect(tabs.length).toBe(2)
    expect(tabs[1].className).toContain('active')
    expect(tabs[1].getAttribute('aria-selected')).toBe('true')
    expect(screen.getByText('content-b')).toBeInTheDocument()
    expect(screen.queryByText('content-a')).toBeNull()
    expect(container.querySelector('[data-test="console-tabs"]')).toBeTruthy()
  })
  it('点击标签→onChange(key)；切换重挂内容区（pane key）', () => {
    const onChange = vi.fn()
    const { container, rerender } = render(<Tabs items={items} activeKey="a" onChange={onChange} />)
    fireEvent.click(screen.getByText('Tab 乙'))
    expect(onChange).toHaveBeenCalledWith('b')
    rerender(<Tabs items={items} activeKey="b" onChange={onChange} />)
    expect(container.querySelector('.ub-tabs-pane')?.textContent).toBe('content-b')
  })
})

describe('Menu→Tree 桥（左树受控）', () => {
  const items = [
    {
      key: 'a',
      label: <span data-test="lefttree-group-以太网交换">以太网交换</span>,
      children: [{ key: 'a-1', label: 'VLAN', children: [{ key: 'leaf', label: 'vlan 模块' }] }],
    },
  ]
  // 组 7 定案：左树桥自绘（eview TreeNode cWRP 无条件 setState 在 React 19
  // 60+ 节点必超嵌套上限 #185 循环压崩页面）——断言改真实 DOM。
  it('自绘树：嵌套渲染 + openKeys 受控展开 + 选中态 + id/data-test 锚', () => {
    const { container } = render(
      <Menu items={items} openKeys={['a']} selectedKeys={['leaf']} onOpenChange={() => {}} onClick={() => {}} />,
    )
    // 一层展开（a 展开→a-1 可见；a-1 未展开→leaf 不可见）
    expect(container.querySelector('#ev_tree_node_ida')).toBeTruthy()
    expect(container.querySelector('#ev_tree_node_ida-1')).toBeTruthy()
    expect(container.querySelector('#ev_tree_node_idleaf')).toBeNull()
    // label JSX 原样渲染（data-test 锚保留——比 eview 文本化更完整）
    expect(container.querySelector('[data-test="lefttree-group-以太网交换"]')).toBeTruthy()
  })
  it('自绘树：箭头/名称点击展开回写、叶子点击 onClick({key})', () => {
    const onOpenChange = vi.fn()
    const onClick = vi.fn()
    const { container, rerender } = render(
      <Menu items={items} openKeys={[]} onOpenChange={onOpenChange} onClick={onClick} />,
    )
    fireEvent.click(container.querySelector('#ev_tree_node_ida')!)
    expect(onOpenChange).toHaveBeenCalledWith(['a'])
    rerender(<Menu items={items} openKeys={['a', 'a-1']} onOpenChange={onOpenChange} onClick={onClick} />)
    fireEvent.click(container.querySelector('#ev_tree_node_idleaf')!)
    expect(onClick).toHaveBeenCalledWith({ key: 'leaf' })
    // 展开态收起回写（去除 a-1）
    fireEvent.click(container.querySelector('.ub-tree-switcher.is-open')!)
    expect(onOpenChange).toHaveBeenLastCalledWith(['a-1'])
  })
  it('inlineCollapsed 收起时不渲染树体', () => {
    const { container } = render(<Menu items={items} inlineCollapsed />)
    expect(container.querySelector('.ub-menu-collapsed')).toBeTruthy()
    expect(recv.last.Tree).toBeUndefined()
  })
})

describe('Table 桥（矩阵全项）', () => {
  const columns = [
    { title: '名称', dataIndex: 'name', width: 120, sorter: true },
    { title: '值', dataIndex: 'val', render: (v: unknown, r: Record<string, unknown>) => `R_${v}_${r.name}` },
  ]
  const data = [{ name: 'r1', val: 'x' }, { name: 'r2', val: 'y' }]
  const base = { columns, dataSource: data, rowKey: 'name' as const }

  it('列映射（allowSort/width/render 参数序换位）与 rowKey 预计算', () => {
    render(<Table {...base} data-test="module-table" />)
    const p = recv.last.Table
    expect(p.columns[0]).toMatchObject({ key: 'name', title: '名称', allowSort: true, width: 120 })
    expect(p.dataset[0].__ubkey).toBe('r1')
    expect(p.rowKey).toBe('__ubkey')
    expect(p.disableEviewSort).toBe(true)
    expect(p.rowClickDelay).toBe(0)
    // eview render(cv,rowData,options,row) → antd render(value,record)
    expect(screen.getByRole('status')).toHaveTextContent('R_v1_r1')
  })

  it('rowKey 函数形态预计算 __ubkey', () => {
    render(<Table {...base} rowKey={(r) => `k-${r.name}`} />)
    expect(recv.last.Table.dataset[1].__ubkey).toBe('k-r2')
  })

  it('勾选受控：行勾/全选统一回写 keys；checkedRowsForceUpdate 强刷', () => {
    const onChange = vi.fn()
    render(<Table {...base} rowSelection={{ selectedRowKeys: ['r1'], onChange }} />)
    expect(recv.last.Table.enableCheckBox).toBe(true)
    expect(recv.last.Table.checkedRows).toEqual([0]) // 桥已映射 rowKey→行序号（R9）
    expect(recv.last.Table.checkedRowsForceUpdate).toBe(true)
    fireEvent.click(screen.getByText('check-r1'))
    expect(onChange).toHaveBeenLastCalledWith(['r1'])
    fireEvent.click(screen.getByText('check-all'))
    expect(onChange).toHaveBeenLastCalledWith(['r1', 'r2'])
  })

  it('行点击、排序合成 antd sorter、分页拆平', () => {
    const onRowClick = vi.fn()
    const onChange = vi.fn()
    const pageChange = vi.fn()
    render(
      <Table
        {...base}
        onRow={(row) => ({ onClick: () => onRowClick(row.name) })}
        onChange={onChange}
        pagination={{ current: 1, pageSize: 10, total: 42, onChange: pageChange }}
      />,
    )
    fireEvent.click(screen.getByText('row1'))
    expect(onRowClick).toHaveBeenCalledWith('r1')
    fireEvent.click(screen.getByText('sort-desc'))
    // 组 5：排序回调第一参合成分页快照（antd 形态，调用点直接读 current/pageSize）。
    expect(onChange).toHaveBeenCalledWith({ current: 1, pageSize: 10 }, undefined, { field: 'mtu', order: 'descend' })
    fireEvent.click(screen.getByText('page3'))
    expect(pageChange).toHaveBeenCalledWith(3, 10)
    expect(recv.last.Table.recordCount).toBe(42)
  })

  it('pagination=false 关闭、rowClassName→customStyleRows、emptyText/loading', () => {
    render(
      <Table
        {...base}
        pagination={false}
        rowClassName={(r) => (r.name === 'r1' ? 'row-create' : '')}
        classStyleMap={{ 'row-create': { background: '#f0fff0' } }}
        locale={{ emptyText: '暂无数据' }}
        loading
      />,
    )
    const p = recv.last.Table
    expect(p.enablePagination).toBe(false)
    expect(p.customStyleRows).toEqual({ 0: { background: '#f0fff0' } })
    expect(p.emptyTableMsg).toBe('暂无数据')
    expect(p.enableLoading).toBe(true)
  })
})
