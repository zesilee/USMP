// EviewUI 桥 · 结构组（组 4.3）：Tabs/Menu(左树→Tree)/Table。
// 对外 props = antd 形态；映射依据 = component-matrix + gate R1/R2（勿凭空改）。
import { createElement, useState, type ReactNode, type CSSProperties } from 'react'
import * as TabNS from '@nce/eview-react/Tab'
import TableMod from '@nce/eview-react/Table'
import { anchorId, pickDefault, textOf } from '../../bridge'

const EvTab = pickDefault(TabNS)
// TabItem 挂在模块具名导出（default 导入拿不到——须命名空间导入；实录坑）。
// F3-R4：真浏览器 interop 下具名可能藏在任意一层 default 内——候选链逐层找。
const EvTabItem = pickDefault(
  [
    (TabNS as { TabItem?: unknown }).TabItem,
    (TabNS as { default?: { TabItem?: unknown } }).default?.TabItem,
    (EvTab as unknown as { TabItem?: unknown }).TabItem,
  ].find((x) => x != null) ?? TabNS,
)
const EvTable = pickDefault(TableMod)

interface CommonProps {
  'data-test'?: string
  className?: string
  style?: CSSProperties
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
        // 更新路径绕行（CAL-R7/F3-R2 定案）：eview Tab/Tree 收新 props 的
        // cWRP 更新路径同步死循环（happy-dom 与真浏览器均实证）——受控值
        // 变化即整体重挂（key），组件恒走首渲路径；受控数据全由 props 喂回，
        // 语义无损（代价=切换无过渡动画，窗口期可接受）。
        key: `ub-tab-${idx}`,
        selectedIndex: idx,
        // eview onClick(index, title, e)。E2E 实录（ifm 7 tab）：溢出折叠场景
        // index 与 items 原始序错位（点「接口列表」内容区仍是首 tab）——
        // 以 title 反查为主、index 兜底。
        onClick: (index: number, title?: string) => {
          const target =
            (title != null ? items.find((i) => textOf(i.label) === title) : undefined) ?? items[index]
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
  // 列头筛选菜单（antd filters/onFilter）：eview 需 embeddedFilter 侦察后映射
  // ——窗口期类型收下、行为暂缺（tasks 5.1 已登记，本地小表次要功能）。
  filters?: Array<{ text: string; value: string | number | boolean }>
  onFilter?: (value: string | number | boolean, record: T) => boolean
  // value 保持 any 对齐 antd 形态（record 已强类型，单元格值类型由调用方 narrow）。
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  render?: (value: any, record: T, index: number) => ReactNode
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
  // rowKey 函数 → 预计算 __ubkey 字段（eview rowKey 仅收字段名）。
  // 本地函数排序（列 sorter 为函数时桥内执行——eview 本地排序已禁走受控流）。
  const [localSort, setLocalSort] = useState<{ key?: string; desc?: boolean }>({})
  const keyOf = (r: T, i: number): string | number =>
    typeof props.rowKey === 'function'
      ? props.rowKey(r, i)
      : (((r as Record<string, unknown>)[props.rowKey ?? 'id'] as string | number) ?? '')
  let sortedData = data
  if (localSort.key) {
    const col = cols.find((c) => (c.dataIndex ?? c.key) === localSort.key)
    if (typeof col?.sorter === 'function') {
      const cmp = col.sorter
      sortedData = [...data].sort((a, b) => (localSort.desc ? -cmp(a, b) : cmp(a, b)))
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
