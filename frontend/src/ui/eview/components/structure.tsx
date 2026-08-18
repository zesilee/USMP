// EviewUI 桥 · 结构组（组 4.3）：Tabs/Menu(左树→Tree)/Table。
// 对外 props = antd 形态；映射依据 = component-matrix + gate R1/R2（勿凭空改）。
import { Children, createElement, isValidElement, type ReactNode, type CSSProperties } from 'react'
import * as TabNS from '@nce/eview-react/Tab'
import TreeMod from '@nce/eview-react/Tree'
import TableMod from '@nce/eview-react/Table'
import { anchorId, pickDefault } from '../../bridge'

const EvTab = pickDefault(TabNS)
// TabItem 挂在模块具名导出（default 导入拿不到——须命名空间导入；实录坑）。
const EvTabItem = pickDefault((TabNS as { TabItem?: unknown }).TabItem ?? (pickDefault(TabNS) as { TabItem?: unknown }).TabItem ?? TabNS)
const EvTree = pickDefault(TreeMod)
const EvTable = pickDefault(TableMod)

interface CommonProps {
  'data-test'?: string
  className?: string
  style?: CSSProperties
}

const textOf = (n: ReactNode): string => {
  if (typeof n === 'string' || typeof n === 'number') return String(n)
  if (isValidElement(n)) {
    const kids = (n.props as { children?: ReactNode }).children
    return Children.toArray(kids ?? [])
      .map((c) => textOf(c))
      .join('')
  }
  return n == null ? '' : String(n)
}

// ===== Tabs → Tab（key↔index 桥；内容区桥自渲，绕开 TabContent 形态不确定性）=====
export function Tabs(
  props: CommonProps & {
    items?: Array<{ key: string; label?: ReactNode; children?: ReactNode; disabled?: boolean }>
    activeKey?: string
    onChange?: (key: string) => void
  },
) {
  const items = props.items ?? []
  const idx = Math.max(0, items.findIndex((i) => i.key === props.activeKey))
  return createElement(
    'div',
    { className: ['ub-tabs', props.className].filter(Boolean).join(' '), 'data-test': props['data-test'] },
    createElement(
      EvTab,
      {
        selectedIndex: idx,
        // eview onClick(index, title, e)；半受控（内切先行）——业务恒回写
        // activeKey（cWRP 同步），无拒写场景不上重挂（YAGNI，集成点复核）。
        onClick: (index: number) => {
          const target = items[index]
          if (target && !target.disabled) props.onChange?.(target.key)
        },
        observerWidthChange: true, // 溢出折叠随容器宽度自动重排（matrix）
      },
      items.map((i) =>
        createElement(EvTabItem, { key: i.key, title: textOf(i.label), disabled: i.disabled }),
      ),
    ),
    // 内容区：仅渲染激活项（与 antd 默认 lazy 语义一致）。
    createElement('div', { className: 'ub-tabs-pane' }, items[idx]?.children),
  )
}

// ===== Menu（左树）→ Tree：items 嵌套 → data；openKeys/selectedKeys 受控桥 =====
export interface MenuLikeItem {
  key: string
  label?: ReactNode
  children?: MenuLikeItem[]
  disabled?: boolean
  type?: string // 'group' → 拍平为不可选父节点
  icon?: ReactNode
}

function toTreeData(items: MenuLikeItem[]): Array<Record<string, unknown>> {
  return items.map((i) => ({
    id: i.key,
    text: textOf(i.label),
    // label 里的 data-test 锚点（现左树形态）挖出 → Tree 节点 DOM id 为
    // ev_tree_node_id<id>，E2E 以 id 锚（FA-05 三路之 id 路）。
    disabled: i.disabled,
    show: true,
    children: i.children?.length ? toTreeData(i.children) : undefined,
    isLeaf: !i.children?.length,
  }))
}

export function Menu(
  props: CommonProps & {
    items?: MenuLikeItem[]
    mode?: string
    openKeys?: string[]
    onOpenChange?: (keys: string[]) => void
    selectedKeys?: string[]
    onClick?: (info: { key: string }) => void
    inlineCollapsed?: boolean
  },
) {
  // inlineCollapsed（整面板收起）由宿主容器样式处理（Sidebar 既有 collapsed 类），
  // 桥仅在收起时隐藏树体。
  if (props.inlineCollapsed) return createElement('div', { className: 'ub-menu-collapsed' })
  return createElement(EvTree, {
    id: anchorId(props['data-test']),
    data: toTreeData(props.items ?? []),
    nodeKey: 'id',
    expandedKeys: props.openKeys ?? [],
    selectedKeys: props.selectedKeys ?? [],
    enableCheckbox: false,
    enableMultiExpand: true,
    // gate 定案：onExpand 回传全量 expandedKeys 数组 → 直接回写。
    onExpand: (keys: string[]) => props.onOpenChange?.(keys ?? []),
    onSelect: (_keys: string[], node: { id?: string }) => {
      if (node?.id != null) props.onClick?.({ key: String(node.id) })
    },
    className: props.className,
  })
}

// ===== Table（矩阵全项映射）=====
interface AntdColumn {
  title?: ReactNode
  dataIndex?: string
  key?: string
  width?: number | string
  fixed?: string | boolean
  sorter?: boolean | object
  ellipsis?: boolean
  render?: (value: unknown, record: Record<string, unknown>, index: number) => ReactNode
}

export function Table(
  props: CommonProps & {
    columns?: AntdColumn[]
    dataSource?: Array<Record<string, unknown>>
    rowKey?: string | ((r: Record<string, unknown>) => string | number)
    rowSelection?: { selectedRowKeys?: Array<string | number>; onChange?: (keys: Array<string | number>) => void }
    onRow?: (row: Record<string, unknown>) => { onClick?: () => void }
    onChange?: (pagination: unknown, filters: unknown, sorter: { field?: string; order?: string } | undefined) => void
    pagination?: { current?: number; pageSize?: number; total?: number; onChange?: (page: number, size: number) => void; pageSizeOptions?: Array<number | string> } | false
    rowClassName?: (row: Record<string, unknown>, index: number) => string
    /** rowClassName 类名 → 行内样式映射（eview 无函数式类名，走 customStyleRows）。 */
    classStyleMap?: Record<string, CSSProperties>
    locale?: { emptyText?: ReactNode }
    loading?: boolean
    size?: string
  },
) {
  const cols = props.columns ?? []
  const data = props.dataSource ?? []
  // rowKey 函数 → 预计算 __ubkey 字段（eview rowKey 仅收字段名）。
  const keyOf = (r: Record<string, unknown>): string | number =>
    typeof props.rowKey === 'function' ? props.rowKey(r) : ((r[props.rowKey ?? 'id'] as string | number) ?? '')
  const dataset = data.map((r) => ({ ...r, __ubkey: keyOf(r) }))

  // rowClassName → customStyleRows（行号→style）。
  let customStyleRows: Record<number, CSSProperties> | undefined
  if (props.rowClassName) {
    customStyleRows = {}
    dataset.forEach((r, i) => {
      const cls = props.rowClassName!(r, i)
      const style = cls && props.classStyleMap?.[cls]
      if (style) customStyleRows![i] = style
    })
  }

  const pag = props.pagination
  return createElement(EvTable, {
    id: anchorId(props['data-test']) ?? 'ub-table',
    columns: cols.map((c) => ({
      key: c.dataIndex ?? c.key,
      title: textOf(c.title),
      width: c.width,
      freezeCol: !!c.fixed,
      allowSort: !!c.sorter,
      // eview render(cellValue, rowData, options, row, isEdit) → antd render(value, record, index)；
      // R9 实测：不设 renderType 时 render 被忽略。R10 实测 'custom' 字面量
      // 也未生效——d.ts 字面量与运行时枚举实值可能不一致（Tag round 前科），
      // 改取真组件静态枚举 ColumnRenderType.CUSTOM，兜底才用 'custom'。
      renderType: c.render
        ? ((EvTable as { ColumnRenderType?: { CUSTOM?: string } }).ColumnRenderType?.CUSTOM ?? 'custom')
        : undefined,
      // R11 实测：render 已被调用但第 4 参非行对象（渲染出 R_x_undefined）；
      // d.ts 第 2 参名 rowData 类型却标 Array——参数位不可信，自适应挑
      // 「像行对象」的参（对象非数组优先 a2→a4；a2 为数组且 a4 为行号则取
      // a2[a4]），行号取数字参。
      render: c.render
        ? (cv: unknown, a2: unknown, _a3: unknown, a4: unknown) => {
            const idx = typeof a4 === 'number' ? a4 : typeof a2 === 'number' ? a2 : 0
            const isRec = (x: unknown): x is Record<string, unknown> =>
              !!x && typeof x === 'object' && !Array.isArray(x)
            const rec = isRec(a2) ? a2 : isRec(a4) ? a4 : Array.isArray(a2) && isRec(a2[idx]) ? a2[idx] : {}
            return c.render!(cv, rec, idx)
          }
        : undefined,
    })),
    dataset,
    rowKey: '__ubkey',
    // 勾选：受控 checkedRows + 强刷（matrix）。R9 实测：不设 keyIndex 时
    // checkedRows 双向都是「行序号」语义（keyIndex 又是数字列序号、对不上
    // 对象行数据）——桥内做行序号↔rowKey 双向映射闭环，对外保持 antd
    // selectedRowKeys 形态。
    enableCheckBox: !!props.rowSelection,
    checkedRows: (props.rowSelection?.selectedRowKeys ?? [])
      .map((k) => dataset.findIndex((r) => r.__ubkey === k))
      .filter((i) => i >= 0),
    checkedRowsForceUpdate: true,
    onRowCheck: (_row: unknown, checkedRows: Array<string | number>) =>
      props.rowSelection?.onChange?.((checkedRows ?? []).map((i) => dataset[Number(i)]?.__ubkey ?? i)),
    onHeaderCheck: (checkedRows: Array<string | number>) =>
      props.rowSelection?.onChange?.((checkedRows ?? []).map((i) => dataset[Number(i)]?.__ubkey ?? i)),
    // 行点击（rowClickDelay:0 关单双击去抖，matrix）。
    onRowClick: props.onRow ? (row: Record<string, unknown>) => props.onRow!(row).onClick?.() : undefined,
    rowClickDelay: 0,
    // 排序：服务端全权（disableEviewSort），合成 antd sorter 形态。
    disableEviewSort: true,
    delayOnColumnSort: true,
    onColumnSort: (sortColumn: { key?: string } | string, sortType: string) => {
      const field = typeof sortColumn === 'string' ? sortColumn : sortColumn?.key
      props.onChange?.(undefined, undefined, {
        field,
        order: sortType === 'desc' ? 'descend' : sortType === 'asc' ? 'ascend' : undefined,
      })
    },
    // 分页：对象拆平；false → 关闭。
    enablePagination: !!pag,
    ...(pag
      ? {
          currentPage: pag.current,
          pageSize: pag.pageSize,
          recordCount: pag.total,
          pageSizeOptions: pag.pageSizeOptions?.map((n) => Number(n)),
          onPageChange: (page: number) => pag.onChange?.(page, pag.pageSize ?? 10),
          onPageSizeChange: (size: number) => pag.onChange?.(1, size),
        }
      : {}),
    customStyleRows,
    emptyTableMsg: typeof props.locale?.emptyText === 'string' ? props.locale.emptyText : undefined,
    enableLoading: props.loading,
    className: [props.className, props.size === 'small' ? 'ub-size-small' : ''].filter(Boolean).join(' ') || undefined,
  })
}
