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

import { Tabs, Menu, Table } from '../../src/ui/eview/components/structure'

afterEach(() => {
  cleanup()
  recv.last = {}
})

describe('Tabs 桥（key↔index + 自渲内容区）', () => {
  const items = [
    { key: 'a', label: 'Tab 甲', children: <p>content-a</p> },
    { key: 'b', label: 'Tab 乙', children: <p>content-b</p>, disabled: false },
  ]
  it('activeKey→selectedIndex、标签 title 文本化、仅渲染激活内容', () => {
    render(<Tabs items={items} activeKey="b" onChange={() => {}} data-test="console-tabs" />)
    expect(recv.last.Tab.selectedIndex).toBe(1)
    expect(document.querySelector('[data-tabitem="Tab 甲"]')).toBeTruthy()
    expect(screen.getByText('content-b')).toBeInTheDocument()
    expect(screen.queryByText('content-a')).toBeNull()
    expect(document.querySelector('[data-test="console-tabs"]')).toBeTruthy()
  })
  it('点击 index→onChange(key)', () => {
    const onChange = vi.fn()
    render(<Tabs items={items} activeKey="a" onChange={onChange} />)
    fireEvent.click(screen.getByText('click-tab2'))
    expect(onChange).toHaveBeenCalledWith('b')
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
  it('items 嵌套→data（label JSX 文本化）、openKeys→expandedKeys 受控', () => {
    render(<Menu items={items} openKeys={['a']} selectedKeys={['leaf']} onOpenChange={() => {}} onClick={() => {}} />)
    const p = recv.last.Tree
    expect(p.data[0]).toMatchObject({ id: 'a', text: '以太网交换' })
    expect(p.data[0].children[0].children[0]).toMatchObject({ id: 'leaf', isLeaf: true })
    expect(p.expandedKeys).toEqual(['a'])
    expect(p.selectedKeys).toEqual(['leaf'])
    expect(p.enableCheckbox).toBe(false)
  })
  it('onExpand 全量 keys 回写、onSelect→onClick({key})', () => {
    const onOpenChange = vi.fn()
    const onClick = vi.fn()
    render(<Menu items={items} openKeys={[]} onOpenChange={onOpenChange} onClick={onClick} />)
    fireEvent.click(screen.getByText('expand'))
    expect(onOpenChange).toHaveBeenCalledWith(['a', 'a-1'])
    fireEvent.click(screen.getByText('select-leaf'))
    expect(onClick).toHaveBeenCalledWith({ key: 'leaf' })
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
    expect(onChange).toHaveBeenCalledWith(undefined, undefined, { field: 'mtu', order: 'descend' })
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
