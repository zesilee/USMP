// EviewUI 桥 · 结构组（组 4.3）：Tabs/Menu(左树→Tree)/Table。
// 对外 props = antd 形态；映射依据 = component-matrix + gate R1/R2（勿凭空改）。
import { createElement, useEffect, useRef, useState, type ReactNode, type CSSProperties } from 'react'
import { createPortal } from 'react-dom'
import TableMod from '@nce/eview-react/Table'
import { anchorId, pickDefault, textOf } from '../../bridge'
import { i18n } from '../../../i18n'

const EvTable = pickDefault(TableMod)

interface CommonProps {
  'data-test'?: string
  className?: string
  style?: CSSProperties
}

export function Tabs(
  props: CommonProps & {
    items?: Array<{ key: string; label?: ReactNode; children?: ReactNode; disabled?: boolean }>
    activeKey?: string
    onChange?: (key: string) => void
  },
) {
  // ===== 标签栏自绘（组 7 E2E 终局定案，先例：Menu→Tree 自绘）=====
  // eview Tab 三连问题实证：①cWRP 更新路径同步死循环（key 重挂绕行过）
  // ②多标签溢出下 onClick index 错位（title 反查绕行过）③自带「可见窗口」
  // 只给部分标签 display 类——详情区几十个标签时目标标签恒不可见（CSS 横滚
  // 救不了）。自绘 div 横排：复用 ev_tab_title/active/ev_tab_bar 类承观感、
  // 补回 role=tab/aria-selected 语义（eview 原生缺失）、标签栏横向滚动全部
  // 可点。内容区自渲维持（key 按激活项防同类型组件复用残留）。
  const items = props.items ?? []
  const idx = Math.max(0, items.findIndex((i) => i.key === props.activeKey))
  return createElement(
    'div',
    { className: ['ub-tabs', props.className].filter(Boolean).join(' '), 'data-test': props['data-test'] },
    createElement(
      'div',
      { className: 'ub-tabs-nav', role: 'tablist' },
      ...items.map((i, n) =>
        createElement(
          'div',
          {
            key: i.key,
            role: 'tab',
            'aria-selected': n === idx,
            tabIndex: 0,
            className: `ev_tab_title${n === idx ? ' active' : ''}${i.disabled ? ' is-disabled' : ''}`,
            onClick: () => {
              if (!i.disabled) props.onChange?.(i.key)
            },
          },
          createElement('span', { className: 'ev_tab_bar' }),
          i.label,
        ),
      ),
    ),
    // 内容区：仅渲染激活项（与 antd 默认 lazy 语义一致）。
    createElement('div', { className: 'ub-tabs-pane', key: `pane-${items[idx]?.key ?? idx}` }, items[idx]?.children),
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
  // ===== 左树自绘（组 7 E2E 定案，替代 eview Tree）=====
  // eview TreeNode.componentWillReceiveProps 无条件 setState（编译产物实证）
  // ——生产左树 60+ 节点单轮嵌套更新超 React 19 的 50 上限（#185）→ #520
  // 恢复重渲 → 再超限 → 无限循环压崩页面；props/开关层无解（tip/滚动开关/
  // 引用稳定四轮实测均不中）。桥自绘：ul/li 递归 + 受控 openKeys/
  // selectedKeys，复用 ev_* 类名承接 eview CSS 观感；id 锚
  // ev_tree_node_id{key}（gate 定案，E2E 同锚）与 label 内 data-test 契约
  // 原样保留（label JSX 直接渲染——比 eview 文本化更完整）。零 TreeNode、
  // 零内部状态、零循环。波 C 切 openinula 后可复评回退真 Tree。
  const open = props.openKeys ?? []
  const selected = props.selectedKeys ?? []
  if (props.inlineCollapsed) return createElement('div', { className: 'ub-menu-collapsed' })
  const toggle = (key: string) => {
    const next = open.includes(key) ? open.filter((k) => k !== key) : [...open, key]
    props.onOpenChange?.(next)
  }
  const renderItems = (items: MenuLikeItem[], level: number): ReactNode =>
    createElement(
      'ul',
      { className: level === 0 ? 'ev_tree ub-tree' : 'ev_tree_sub', key: `lv${level}` },
      ...items.map((i) => {
        const hasChildren = !!i.children?.length
        const isOpen = open.includes(i.key)
        const isSel = selected.includes(i.key)
        return createElement(
          'li',
          {
            key: i.key,
            className: hasChildren ? (isOpen ? 'ev_tree_expanded' : 'ev_tree_collapsed') : 'ev_tree_leaf',
          },
          createElement(
            'div',
            { className: 'ev_tree_node_cont', style: { paddingLeft: 8 + level * 14 } },
            createElement('span', {
              className: hasChildren
                ? `ub-tree-switcher${isOpen ? ' is-open' : ''}`
                : 'ub-tree-switcher ub-tree-switcher-noop',
              onClick: hasChildren ? () => toggle(i.key) : undefined,
              'aria-expanded': hasChildren ? isOpen : undefined,
            }),
            createElement(
              'a',
              {
                id: `ev_tree_node_id${i.key}`,
                className: `ev_tree_name${isSel ? ' ub-tree-selected' : ''}${i.disabled ? ' is-disabled' : ''}`,
                onClick: () => {
                  if (i.disabled) return
                  if (hasChildren) toggle(i.key)
                  else props.onClick?.({ key: i.key })
                },
              },
              createElement(
                'span',
                { className: 'ev_tree_text', title: textOf(i.label) },
                i.icon ?? null,
                i.label ?? i.key,
              ),
            ),
          ),
          hasChildren && isOpen ? renderItems(i.children!, level + 1) : null,
        )
      }),
    )
  return createElement(
    'div',
    { className: ['ub-menu', props.className].filter(Boolean).join(' '), 'data-test': props['data-test'] },
    renderItems(props.items ?? [], 0),
  )
}

// ===== Table（矩阵全项映射）=====
/** 对外列类型（antd TableColumnType 对等面，组 5 接线：业务泛型行类型经此收口）。 */
export interface TableColumnType<T = Record<string, unknown>> {
  title?: ReactNode
  dataIndex?: string
  key?: string
  width?: number | string
  fixed?: string | boolean
  // 函数=本地比较排序（桥内执行，eview 本地排序已禁）；true=服务端下推。
  sorter?: boolean | object | ((a: T, b: T) => number)
  ellipsis?: boolean
  // 列头筛选菜单（antd filters/onFilter）：eview filter/embeddedFilter 形状
  // 未在 d.ts 暴露（仅 object）——桥自绘筛选菜单实现（ColFilter），行为对齐
  // antd 语义（谓词本地过滤 + onChange 合成 filters 快照）。
  filters?: Array<{ text: string; value: string | number | boolean }>
  onFilter?: (value: string | number | boolean, record: T) => boolean
  // value 保持 any 对齐 antd 形态（record 已强类型，单元格值类型由调用方 narrow）。
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  render?: (value: any, record: T, index: number) => ReactNode
}
type FilterVal = string | number | boolean

// ===== 列头筛选自绘（follow-up 债 5.1）=====
// eview 列 filter/embeddedFilter 形状未暴露（d.ts 仅 object、无文档），不猜
// 其 UI——桥自绘（先例：Tabs/Menu/Popover）。触发器塞进列 title（eview 列
// title 收 ReactNode），弹层 portal 到 body（fixed 定位防表头 overflow 裁剪），
// 点外关闭=放弃草稿（antd 同语义）；确定/重置回写 onApply 由 Table 桥过滤。
function ColFilter(props: {
  colKey: string
  title: ReactNode
  options: Array<{ text: string; value: FilterVal }>
  values: FilterVal[]
  onApply: (vals: FilterVal[]) => void
}) {
  const [open, setOpen] = useState(false)
  const [draft, setDraft] = useState<FilterVal[]>([])
  const wrapRef = useRef<HTMLSpanElement | null>(null)
  const posRef = useRef({ top: 0, left: 0 })
  const openRef = useRef(open)
  openRef.current = open
  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (!openRef.current) return
      const t = e.target as Element | null
      if (t && (t.closest?.('.ub-filter-popup') || wrapRef.current?.contains(t))) return
      setOpen(false)
    }
    window.addEventListener('click', handler, true)
    return () => window.removeEventListener('click', handler, true)
  }, [])
  const active = props.values.length > 0
  return createElement(
    'span',
    { className: 'ub-col-filter-wrap', ref: wrapRef },
    props.title,
    createElement(
      'button',
      {
        type: 'button',
        className: 'ub-col-filter' + (active ? ' is-active' : ''),
        'aria-label': `filter-${props.colKey}`,
        onClick: (e: { stopPropagation: () => void }) => {
          e.stopPropagation() // 防触发 eview 列头排序
          if (open) {
            setOpen(false)
            return
          }
          const r = wrapRef.current?.getBoundingClientRect()
          posRef.current = { top: (r?.bottom ?? 0) + 4, left: r?.left ?? 0 }
          setDraft(props.values)
          setOpen(true)
        },
      },
      createElement(
        'svg',
        { width: 12, height: 12, viewBox: '0 0 16 16', fill: 'currentColor', 'aria-hidden': true },
        createElement('path', { d: 'M1.5 2h13l-5 6v5.5l-3 1.5V8l-5-6z' }),
      ),
    ),
    open &&
      createPortal(
        createElement(
          'div',
          { className: 'ub-filter-popup', style: { top: posRef.current.top, left: posRef.current.left } },
          props.options.map((o) =>
            createElement(
              'label',
              { key: String(o.value), className: 'ub-filter-option' },
              createElement('input', {
                type: 'checkbox',
                checked: draft.includes(o.value),
                onChange: () =>
                  setDraft((d) => (d.includes(o.value) ? d.filter((v) => v !== o.value) : [...d, o.value])),
              }),
              o.text,
            ),
          ),
          createElement(
            'div',
            { className: 'ub-filter-actions' },
            createElement(
              'button',
              {
                type: 'button',
                className: 'ub-filter-reset',
                onClick: () => {
                  setOpen(false)
                  props.onApply([])
                },
              },
              i18n.global.t('common.reset'),
            ),
            createElement(
              'button',
              {
                type: 'button',
                className: 'ub-filter-ok',
                onClick: () => {
                  setOpen(false)
                  props.onApply(draft)
                },
              },
              i18n.global.t('common.confirm'),
            ),
          ),
        ),
        document.body,
      ),
  )
}

export function Table<T extends object = Record<string, unknown>>(
  props: CommonProps & {
    columns?: Array<TableColumnType<T>>
    dataSource?: T[]
    rowKey?: string | ((r: T, index?: number) => string | number)
    rowSelection?: { selectedRowKeys?: Array<string | number>; onChange?: (keys: Array<string | number>) => void }
    onRow?: (row: T) => { onClick?: () => void }
    onChange?: (
      pagination: { current?: number; pageSize?: number },
      filters: unknown,
      sorter: { field?: string; order?: string } | undefined,
    ) => void
    pagination?: {
      current?: number
      pageSize?: number
      total?: number
      onChange?: (page: number, size: number) => void
      pageSizeOptions?: Array<number | string>
      // eview 分页自带尺寸切换与总数文案，两项类型收下不再单独映射。
      showSizeChanger?: boolean
      showTotal?: (total: number) => ReactNode
    } | false
    rowClassName?: (row: T, index: number) => string
    /** rowClassName 类名 → 行内样式映射（eview 无函数式类名，走 customStyleRows）。 */
    classStyleMap?: Record<string, CSSProperties>
    locale?: { emptyText?: ReactNode }
    loading?: boolean
    size?: string
    // 展开行（antd expandable）：eview 展开形态待侦察——窗口期类型收下、
    // 行为暂缺（tasks 5.1 已登记）。
    expandable?: { defaultExpandAllRows?: boolean }
  },
) {
  const cols = props.columns ?? []
  // 展开行（antd expandable/children 树表格）：eview 无对应形态——
  // defaultExpandAllRows 语义下拍平渲染（子行全部铺开，E2E 实证需求：
  // 变更内容对话框树形三列）；children 字段从行对象剥除防 eview 误读。
  const rawData = props.dataSource ?? []
  const data: T[] = []
  const flatten = (rows: T[]) => {
    for (const r of rows) {
      data.push(r)
      const kids = (r as { children?: T[] }).children
      if (props.expandable?.defaultExpandAllRows && kids?.length) flatten(kids)
    }
  }
  flatten(rawData)
  // 列头筛选（follow-up 债 5.1）：filters+onFilter 列的已生效值；谓词过滤在
  // 排序之前作用于桥收到的 dataset（本地模式父组件已自行切页——与 antd 受控
  // 分页语义一致：筛选只作用于传入数据，total 不动）。
  const [colFilters, setColFilters] = useState<Record<string, FilterVal[]>>({})
  const filterCols = cols.filter((c) => c.filters?.length && c.onFilter)
  const activeFilterCols = filterCols.filter((c) => colFilters[(c.dataIndex ?? c.key)!]?.length)
  const filteredData = activeFilterCols.length
    ? data.filter((row) =>
        activeFilterCols.every((c) => colFilters[(c.dataIndex ?? c.key)!].some((v) => c.onFilter!(v, row))),
      )
    : data
  // rowKey 函数 → 预计算 __ubkey 字段（eview rowKey 仅收字段名）。
  // 本地函数排序（列 sorter 为函数时桥内执行——eview 本地排序已禁走受控流）。
  const [localSort, setLocalSort] = useState<{ key?: string; desc?: boolean }>({})
  const keyOf = (r: T, i: number): string | number =>
    typeof props.rowKey === 'function'
      ? props.rowKey(r, i)
      : (((r as Record<string, unknown>)[props.rowKey ?? 'id'] as string | number) ?? '')
  let sortedData = filteredData
  if (localSort.key) {
    const col = cols.find((c) => (c.dataIndex ?? c.key) === localSort.key)
    if (typeof col?.sorter === 'function') {
      const cmp = col.sorter
      sortedData = [...filteredData].sort((a, b) => (localSort.desc ? -cmp(a, b) : cmp(a, b)))
    }
  }
  const dataset = sortedData.map((r, i) => ({ ...(r as Record<string, unknown>), children: undefined, __ubkey: keyOf(r, i) }))

  // rowClassName → customStyleRows（行号→style）。
  let customStyleRows: Record<number, CSSProperties> | undefined
  if (props.rowClassName) {
    customStyleRows = {}
    dataset.forEach((r, i) => {
      const cls = props.rowClassName!(r as T, i)
      const style = cls && props.classStyleMap?.[cls]
      if (style) customStyleRows![i] = style
    })
  }

  const pag = props.pagination
  return createElement(EvTable, {
    id: anchorId(props['data-test']) ?? 'ub-table',
    columns: cols.map((c) => {
      const k = (c.dataIndex ?? c.key)!
      return {
      key: k,
      // filters 列：标题包自绘筛选触发器（确定/重置回写 → 过滤管线 + antd
      // onChange filters 快照合成；分页快照与排序通道同形）。
      // C1 探针定案：eview 对无显式宽度的列按标题文本测宽——组件标题量不出
      // 长度会分到 0 宽（列被挤没）；titleComponentToText 即为此供测宽文本。
      titleComponentToText: c.filters?.length && c.onFilter ? textOf(c.title) : undefined,
      title:
        c.filters?.length && c.onFilter
          ? createElement(ColFilter, {
              colKey: k,
              title: textOf(c.title),
              options: c.filters,
              values: colFilters[k] ?? [],
              onApply: (vals: FilterVal[]) => {
                setColFilters((prev) => ({ ...prev, [k]: vals }))
                const snap: Record<string, FilterVal[] | null> = {}
                for (const fc of filterCols) {
                  const fk = (fc.dataIndex ?? fc.key)!
                  const cur = fk === k ? vals : (colFilters[fk] ?? [])
                  snap[fk] = cur.length ? cur : null
                }
                props.onChange?.(
                  { current: pag ? pag.current : undefined, pageSize: pag ? pag.pageSize : undefined },
                  snap,
                  undefined,
                )
              },
            })
          : textOf(c.title),
      width: c.width,
      // fixed 不映射冻结（内网实证三连）：eview 冻结=拆分子表渲染，行高不
      // 同步致单元格跨行互相覆盖；freezeColPosition 是表级单侧属性，
      // fixed:'right' 的操作列被冻到最左；拆表还破坏横向滚动。列保持自然
      // 顺序（操作列自然排最右，宽表随表体滚动——antd 时代观感）。
      allowSort: !!c.sorter,
      // eview render(cellValue, rowData, options, row, isEdit) → antd render(value, record, index)；
      // R9 实测：不设 renderType 时 render 被忽略。R10 实测 'custom' 字面量
      // 也未生效——d.ts 字面量与运行时枚举实值可能不一致（Tag round 前科），
      // 改取真组件静态枚举 ColumnRenderType.CUSTOM，兜底才用 'custom'。
      renderType: c.render
        ? ((EvTable as { ColumnRenderType?: { CUSTOM?: string } }).ColumnRenderType?.CUSTOM ?? 'custom')
        : undefined,
      // R13 探针定案：实参=(cellValue, [cellValue], options, {id,data,rawData},
      // {isEdit}, …)——行数据在第 4 参上下文对象的 rawData/data 字段内，
      // 不是参数本身（R11/R12 渲染 R_x_undefined 的根因）。rawData 优先
      // （原始 dataset 行，含 __ubkey）；第 4 参直接为行对象时兜底兼容
      // （F2 stub 形态）。index 由 __ubkey 反查 dataset。
      render: c.render
        ? (cv: unknown, a2: unknown, _a3: unknown, a4: unknown) => {
            const isRec = (x: unknown): x is Record<string, unknown> =>
              !!x && typeof x === 'object' && !Array.isArray(x)
            const ctx = isRec(a4) ? (a4 as { data?: unknown; rawData?: unknown }) : undefined
            const rec = [ctx?.rawData, ctx?.data, a4, a2].find(isRec) ?? {}
            const idx = dataset.findIndex(
              (r) => r === rec || (rec.__ubkey != null && r.__ubkey === rec.__ubkey),
            )
            return c.render!(cv, rec as T, idx >= 0 ? idx : 0)
          }
        : undefined,
      }
    }),
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
    onRowClick: props.onRow ? (row: Record<string, unknown>) => props.onRow!(row as T).onClick?.() : undefined,
    rowClickDelay: 0,
    // 排序：服务端全权（disableEviewSort），合成 antd sorter 形态。
    disableEviewSort: true,
    delayOnColumnSort: true,
    onColumnSort: (sortColumn: { key?: string } | string, sortType: string) => {
      const field = typeof sortColumn === 'string' ? sortColumn : sortColumn?.key
      const col = cols.find((c) => (c.dataIndex ?? c.key) === field)
      if (typeof col?.sorter === 'function') {
        setLocalSort(sortType === 'origin' ? {} : { key: field, desc: sortType === 'desc' })
      }
      // antd 形态：onChange 第一参恒为分页快照（调用点直接读 current/pageSize）。
      props.onChange?.(
        { current: pag ? pag.current : undefined, pageSize: pag ? pag.pageSize : undefined },
        undefined,
        {
          field,
          order: sortType === 'desc' ? 'descend' : sortType === 'asc' ? 'ascend' : undefined,
        },
      )
    },
    // 分页：对象拆平；false → 关闭。
    enablePagination: !!pag,
    ...(pag
      ? {
          currentPage: pag.current,
          pageSize: pag.pageSize,
          recordCount: pag.total,
          pageSizeOptions: pag.pageSizeOptions?.map((n) => Number(n)),
          // 分页动作双通道回写：antd 统一 onChange（第一参分页快照）+
          // pagination.onChange（两者调用点各取所需）。
          onPageChange: (page: number) => {
            pag.onChange?.(page, pag.pageSize ?? 10)
            props.onChange?.({ current: page, pageSize: pag.pageSize }, undefined, undefined)
          },
          onPageSizeChange: (size: number) => {
            pag.onChange?.(1, size)
            props.onChange?.({ current: 1, pageSize: size }, undefined, undefined)
          },
        }
      : {}),
    customStyleRows,
    emptyTableMsg: typeof props.locale?.emptyText === 'string' ? props.locale.emptyText : undefined,
    enableLoading: props.loading,
    className: [props.className, props.size === 'small' ? 'ub-size-small' : ''].filter(Boolean).join(' ') || undefined,
  })
}
