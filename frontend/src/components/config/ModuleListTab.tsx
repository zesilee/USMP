import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  Alert,
  Button,
  Checkbox,
  Empty,
  Popover,
  Select,
  Table,
  Tag,
  confirm,
  icons,
  type TableColumnType,
} from '../../ui'
import { i18n } from '../../i18n'
import { useChangesetStore } from '../../stores/changeset'
import { useListQuery } from '../../hooks/useListQuery'
import type { Field } from '../../utils/crdSchemaParser'
import {
  deriveColumns,
  deriveAllColumns,
  deriveKeyField,
  configPathFor,
  leafName,
  filterableFields,
  filterRows,
  type ConsoleTab,
} from '../../utils/moduleConsole'
import ItemDetailPane from './ItemDetailPane'
import { buildDataColumns } from './listColumns'
import './ModuleListTab.scss'

// ModuleListTab（FE-11/24/25 + FE-16/21 接线）：运行时动态列表格全功能形态——
// 双模式分页（首读 limit=200 探测：total>阈值转服务端，翻页/搜索/排序下推 BR-13；
// 否则纯客户端）、FE-24 节点不支持占位（预标记零请求/信封学习/重试逃生）、
// 变更集标记合成视图（create 行叠加/update·delete 标记）、行删除入变更集
// （待创建直接移除）、详情同屏（未提交草稿守卫）、新鲜度埋点。零 per-module 代码。
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

export interface ModuleListTabProps {
  tab: ConsoleTab
  rootName: string
  device: string
  /** schema 预标记不支持（CN-05）。 */
  unsupported?: boolean
  onUnsupportedChange?: (unsupported: boolean) => void
}

export function normalizeRows(
  subtree: any,
  listField: Field,
  tabField: Field,
  keyField: string,
): { rows: Record<string, any>[]; key: string } {
  const candidates = [leafName(listField), leafName(tabField)]
  for (const k of candidates) {
    const v = subtree?.[k]
    if (Array.isArray(v)) return { rows: v, key: k }
    if (v && typeof v === 'object') {
      return {
        rows: Object.entries(v).map(([kk, vv]) =>
          typeof vv === 'object' && vv !== null
            ? { [keyField]: isNaN(Number(kk)) ? kk : Number(kk), ...(vv as object) }
            : { [keyField]: kk },
        ),
        key: k,
      }
    }
  }
  if (Array.isArray(subtree)) return { rows: subtree, key: candidates[0] }
  return { rows: [], key: candidates[0] }
}

export default function ModuleListTab(props: ModuleListTabProps) {
  const { tab, rootName, device, unsupported, onUnsupportedChange } = props
  const listField = tab.listField || tab.field
  const keyField = useMemo(() => deriveKeyField(listField), [listField])
  const defaultCols = useMemo(() => deriveColumns(listField), [listField])
  const allCols = useMemo(() => deriveAllColumns(listField), [listField])
  const searchFields = useMemo(() => filterableFields(listField), [listField])
  const configPath = useMemo(() => configPathFor(rootName, tab.field.path), [rootName, tab.field.path])
  // 变更集条目路径（带前导斜杠，与 ItemDetailPane 同源）。
  const entryPath = '/' + configPath

  const changeset = useChangesetStore()

  const [visibleCols, setVisibleCols] = useState<string[]>(() => defaultCols.map((c) => c.path))
  useEffect(() => setVisibleCols(defaultCols.map((c) => c.path)), [defaultCols])
  const shownColumns = useMemo(() => allCols.filter((c) => visibleCols.includes(c.path)), [allCols, visibleCols])

  const [selectedKeys, setSelectedKeys] = useState<Array<string | number>>([])

  // 取数编排（FE-24/25）：requestRows 收口 + 双模式全部收敛在 useListQuery。
  const normalize = useCallback(
    (subtree: any) => normalizeRows(subtree, listField, tab.field, keyField),
    [listField, tab.field, keyField],
  )
  const {
    items,
    postKey,
    loading,
    error,
    queryAt,
    nodeUnsupported,
    serverMode,
    serverTotal,
    page,
    setPage,
    pageSize,
    setPageSize,
    setSortState,
    applied,
    setApplied,
    load,
    pageLoad,
  } = useListQuery({
    device,
    configPath,
    readonlyTab: !!tab.readonly,
    listField,
    searchFields,
    normalize,
    unsupported,
  })
  useEffect(() => onUnsupportedChange?.(nodeUnsupported), [nodeUnsupported, onUnsupportedChange])
  useEffect(() => setSelectedKeys([]), [items])

  // ===== 高级搜索（FE-11）=====
  const [draft, setDraft] = useState<Record<string, any>>({})
  const [searchOpen, setSearchOpen] = useState(false)

  // ===== 详情同屏（FE-21）=====
  const [selectedRow, setSelectedRow] = useState<Record<string, any> | null>(null)
  const [detailMode, setDetailMode] = useState<'edit' | 'create' | null>(null)
  const dirtyRef = useRef(false) // ItemDetailPane 报告的未提交草稿态

  useEffect(() => {
    setSortState(null)
    setDetailMode(null)
    setSelectedRow(null)
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [device, configPath])

  // ===== 变更集标记合成视图（FE-11 二期）=====
  const mergedItems = useMemo(() => {
    const existing = new Set(items.map((r) => String(r[keyField])))
    const pendingCreates = changeset
      .entriesFor(device)
      .filter((e) => e.path === entryPath && e.op === 'create' && !existing.has(String(e.keyValue)))
      .map((e) => ({ ...(e.payload ?? {}) }))
    return [...items, ...pendingCreates]
  }, [items, changeset, device, entryPath, keyField])

  const rowMark = useCallback(
    (row: Record<string, any>): '' | 'create' | 'update' | 'delete' => {
      const e = changeset.entryFor(device, entryPath, String(row[keyField]))
      return (e?.op as any) ?? ''
    },
    [changeset, device, entryPath, keyField],
  )

  // 服务端模式过滤已在后端完成；客户端模式本地过滤（pending create 行仍叠加）。
  const filteredRows = serverMode ? mergedItems : filterRows(mergedItems, applied, searchFields)
  const totalCount = serverMode ? serverTotal : filteredRows.length
  const pagedRows = serverMode ? filteredRows : filteredRows.slice((page - 1) * pageSize, page * pageSize)

  // ===== 操作门禁（FE-11 list 级 operation-exclude）=====
  const canUpdate = !tab.field.operationExclude?.includes('update') && !listField.operationExclude?.includes('update')
  const canDelete = !tab.field.operationExclude?.includes('delete') && !listField.operationExclude?.includes('delete')

  // 未提交草稿守卫（FE-21 负路径）：取消则停留原条目、草稿保留。
  const ensureNoDraft = async (): Promise<boolean> => {
    if (!detailMode || !dirtyRef.current) return true
    return confirm(t('console.unsavedSwitch'), { title: t('console.unsavedTitle') })
  }

  const openEdit = async (row: Record<string, any>) => {
    if (tab.readonly) return
    const same = detailMode === 'edit' && selectedRow?.[keyField] === row[keyField]
    if (same) return
    if (!(await ensureNoDraft())) return
    setSelectedRow(row)
    setDetailMode('edit')
  }

  const openCreate = async () => {
    if (!(await ensureNoDraft())) return
    setSelectedRow(null)
    setDetailMode('create')
  }

  // 行删除（FE-16）：确认 → 入变更集（待创建行直接移除，不产生删除报文）。
  const onDelete = async (row: Record<string, any>) => {
    const keyValue = String(row[keyField] ?? '')
    const ok = await confirm(t('console.deleteConfirm', { key: keyValue }), {
      title: t('console.deleteTitle'),
      danger: true,
    })
    if (!ok) return
    changeset.markDelete(device, {
      path: entryPath,
      listKey: postKey || leafName(listField),
      keyValue,
      label: `${tab.label} ${keyValue}`,
    })
  }

  const onUndelete = (row: Record<string, any>) => {
    changeset.unmarkDelete(device, entryPath, String(row[keyField] ?? ''))
  }

  // ===== 运行时动态列 =====
  const columns: TableColumnType<Record<string, any>>[] = [
    {
      title: '',
      key: '__mark__',
      width: 72,
      render: (_: unknown, row: Record<string, any>) => {
        const m = rowMark(row)
        if (m === 'create') return <Tag color="green" data-test="mark-create">{t('console.markCreate')}</Tag>
        if (m === 'update') return <Tag color="orange" data-test="mark-update">{t('console.markUpdate')}</Tag>
        if (m === 'delete') return <Tag color="red" data-test="mark-delete">{t('console.markDelete')}</Tag>
        return null
      },
    },
    ...buildDataColumns(shownColumns, serverMode),
  ]

  if (!tab.readonly && (canUpdate || canDelete)) {
    columns.push({
      title: t('common.actions'),
      key: '__actions__',
      width: 200,
      fixed: 'right',
      render: (_: unknown, row: Record<string, any>) => (
        <span onClick={(e) => e.stopPropagation()}>
          {canUpdate && (
            <Button type="link" size="small" onClick={() => void openEdit(row)}>
              {t('common.edit')}
            </Button>
          )}
          {canDelete &&
            (rowMark(row) === 'delete' ? (
              <Button type="link" size="small" data-test="undelete-btn" onClick={() => onUndelete(row)}>
                {t('console.undelete')}
              </Button>
            ) : (
              <Button type="link" size="small" danger onClick={() => void onDelete(row)}>
                {t('common.delete')}
              </Button>
            ))}
        </span>
      ),
    })
  }

  if (nodeUnsupported) {
    return (
      <div className="module-list-tab" data-test="node-unsupported">
        <Empty description={t('console.nodeUnsupported')}>
          <Button size="small" icon={<icons.RefreshIcon />} onClick={() => void load(true)}>
            {t('common.retry')}
          </Button>
        </Empty>
      </div>
    )
  }

  return (
    <div className="module-list-tab" data-test="module-list-tab">
      <div className="list-toolbar">
        <span className="query-at" data-test="query-at">
          {queryAt && t('console.queryAt', { time: queryAt })}
        </span>
        {!tab.readonly && canUpdate && (
          <Button type="primary" size="small" icon={<icons.PlusIcon />} onClick={() => void openCreate()} data-test="add-row">
            {t('common.create')}
          </Button>
        )}
        {searchFields.length > 0 && (
          <Popover
            trigger="click"
            open={searchOpen}
            onOpenChange={setSearchOpen}
            content={
              <div className="adv-search search-panel" data-test="adv-search-panel">
                {searchFields.map((f) => (
                  <label key={f.path} className="adv-search-item">
                    <span>{f.label}</span>
                    {f.options?.length ? (
                      // 枚举查询条件用下拉（与详情区控件同型，FE-11）。
                      <Select
                        size="small"
                        allowClear
                        style={{ minWidth: 160 }}
                        options={f.options}
                        value={draft[leafName(f)] || undefined}
                        onChange={(v) => setDraft((prev) => ({ ...prev, [leafName(f)]: v ?? '' }))}
                      />
                    ) : (
                      <input
                        value={draft[leafName(f)] ?? ''}
                        onChange={(e) => setDraft((prev) => ({ ...prev, [leafName(f)]: e.target.value }))}
                      />
                    )}
                  </label>
                ))}
                <div className="adv-search-actions">
                  <Button
                    size="small"
                    type="primary"
                    data-test="adv-search-apply"
                    onClick={() => {
                      setApplied({ ...draft })
                      setPage(1)
                      setSearchOpen(false)
                      if (serverMode) void pageLoad(false, { page: 1, applied: { ...draft } })
                    }}
                  >
                    {t('common.apply')}
                  </Button>
                  <Button
                    size="small"
                    data-test="adv-search-reset"
                    onClick={() => {
                      setDraft({})
                      setApplied({})
                      setPage(1)
                      if (serverMode) void pageLoad(false, { page: 1, applied: {} })
                    }}
                  >
                    {t('common.reset')}
                  </Button>
                </div>
              </div>
            }
          >
            <Button size="small" icon={<icons.SearchIcon />} data-test="adv-search">
              {t('console.advancedSearch')}
            </Button>
          </Popover>
        )}
        <Button size="small" icon={<icons.RefreshIcon />} onClick={() => void load(true)} data-test="fetch-source">
          {t('console.fetchSource')}
        </Button>
        <Popover
          trigger="click"
          content={
            <div className="col-settings" data-test="column-settings-panel">
              {allCols.map((c) => (
                <label key={c.path} className="col-settings-item">
                  <Checkbox
                    checked={visibleCols.includes(c.path)}
                    onChange={(e) =>
                      setVisibleCols((prev) => (e.target.checked ? [...prev, c.path] : prev.filter((p) => p !== c.path)))
                    }
                  />
                  <span>{c.label}</span>
                </label>
              ))}
            </div>
          }
        >
          <Button size="small" icon={<icons.SettingIcon />} data-test="column-settings" title={t('console.columnSettings')} />
        </Popover>
      </div>

      {error && <Alert type="error" showIcon message={error} className="list-error" />}

      <Table
        size="small"
        rowKey={(r) => String(r[keyField] ?? JSON.stringify(r))}
        columns={columns}
        dataSource={pagedRows}
        loading={loading}
        rowClassName={(row) => {
          const m = rowMark(row)
          return m ? `row-${m}` : ''
        }}
        rowSelection={{ selectedRowKeys: selectedKeys, onChange: (keys) => setSelectedKeys(keys) }}
        onRow={(row) => ({ onClick: () => void openEdit(row) })}
        onChange={(pg, _filters, sorter: any) => {
          const nextPage = pg.current ?? 1
          const nextSize = pg.pageSize ?? pageSize
          const nextSort = sorter?.order
            ? { prop: String(sorter.field), desc: sorter.order === 'descend' }
            : null
          setPage(nextPage)
          setPageSize(nextSize)
          setSortState(nextSort)
          if (serverMode) void pageLoad(false, { page: nextPage, pageSize: nextSize, sort: nextSort })
        }}
        locale={{
          emptyText: <Empty description={device ? t('console.emptyNoConfig') : t('console.emptySelectDevice')} />,
        }}
        pagination={{
          current: page,
          pageSize,
          total: totalCount,
          pageSizeOptions: [10, 20, 50],
          showSizeChanger: true,
          showTotal: (tot) => t('console.totalCount', { total: tot }),
        }}
      />

      {detailMode && (
        <ItemDetailPane
          tab={tab}
          rootName={rootName}
          device={device}
          mode={detailMode}
          row={detailMode === 'edit' ? selectedRow : null}
          postKey={postKey || leafName(listField)}
          onDirtyChange={(d) => {
            dirtyRef.current = d
          }}
          onClose={() => {
            setDetailMode(null)
            setSelectedRow(null)
            dirtyRef.current = false
          }}
          onStaged={() => {
            dirtyRef.current = false
            setDetailMode(null)
            setSelectedRow(null)
          }}
        />
      )}
    </div>
  )
}
