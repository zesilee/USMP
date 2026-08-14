import { useCallback, useEffect, useMemo, useState } from 'react'
import { Alert, Button, Empty, Popover, Checkbox, Table, Tag, icons, type TableColumnType } from '../../ui'
import { i18n } from '../../i18n'
import { getConfig } from '../../api'
import type { Field } from '../../utils/crdSchemaParser'
import {
  deriveColumns,
  deriveAllColumns,
  deriveKeyField,
  configPathFor,
  cellVisible,
  statusTone,
  leafName,
  type ConsoleTab,
} from '../../utils/moduleConsole'
import './ModuleListTab.scss'

// ModuleListTab（FE-11 切片形态，R05 命门验证件）：列表 Tab 的**运行时动态列**——
// 列集合由 schema 派生纯函数现场算出（deriveColumns 分层启发式 + 列设置显隐 +
// 可用列全集），antd Table 以 columns 配置数组承接：排序/enum·boolean 表头筛选/
// 多选/自定义单元格（when 行级显隐、状态点、enum 色板轮转 Tag、boolean Tag）
// 全部按列元数据生成，零 per-module 代码。取数走 /config/:ip/*path 回读子树契约
// （normalizeRows 语义自旧版平移）。服务端分页双模式（FE-25）随 tasks 8.5 扩展。

const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

export interface ModuleListTabProps {
  tab: ConsoleTab
  rootName: string
  device: string
  /** 行点击（详情区随 tasks 8 组接线）。 */
  onRowClick?: (row: Record<string, any>) => void
}

// 回读子树 → 行数组（语义自旧版逐分支平移）：优先 list 名，回退 tab 名；
// 值可为数组（RFC7951 list）或对象（容器键控形态，键并入 keyField 列）。
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

// enum Tag 色板轮转（按枚举值序号取色，非语义映射，R05）。
const TAG_TYPES = ['blue', 'green', 'orange', 'cyan', 'red'] as const
function tagColor(col: Field, val: string): string {
  const idx = (col.options || []).findIndex((o) => String(o.value) === val)
  return TAG_TYPES[Math.max(idx, 0) % TAG_TYPES.length]
}

function rowVal(row: Record<string, any>, col: Field): string {
  const v = row[leafName(col)]
  return v == null ? '' : String(v)
}

export default function ModuleListTab({ tab, rootName, device, onRowClick }: ModuleListTabProps) {
  const listField = tab.listField || tab.field
  const keyField = useMemo(() => deriveKeyField(listField), [listField])
  const defaultCols = useMemo(() => deriveColumns(listField), [listField])
  const allCols = useMemo(() => deriveAllColumns(listField), [listField])

  // 列设置显隐（FE-11）：默认集 = 派生默认列；用户勾选驱动展示子集。
  const [visibleCols, setVisibleCols] = useState<string[]>(() => defaultCols.map((c) => c.path))
  useEffect(() => setVisibleCols(defaultCols.map((c) => c.path)), [defaultCols])
  const shownColumns = useMemo(
    () => allCols.filter((c) => visibleCols.includes(c.path)),
    [allCols, visibleCols],
  )

  const [rows, setRows] = useState<Record<string, any>[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [queryAt, setQueryAt] = useState('')
  const [selected, setSelected] = useState<React.Key[]>([])

  const configPath = useMemo(() => configPathFor(rootName, tab.field.path), [rootName, tab.field.path])

  const load = useCallback(
    async (force = false) => {
      if (!device) {
        setRows([]) // 设备上下文清空：不留陈旧行
        setSelected([])
        return
      }
      setLoading(true)
      setError('')
      try {
        const res = await getConfig(device, configPath, force)
        const body = res.data?.data?.data ?? res.data?.data
        const { rows: r } = normalizeRows(body, listField, tab.field, keyField)
        setRows(r)
        setSelected([]) // 数据换代清选中（对齐旧 el-table 行为）
        setQueryAt(new Date().toLocaleString())
      } catch (e: any) {
        setError(e?.response?.data?.message || e?.message || t('common.loadFailed'))
        // §9：强制回读失败保留原列表（保留原配置）；常规取数失败清空。
        if (!force) setRows([])
      } finally {
        setLoading(false)
      }
    },
    [device, configPath, listField, tab.field, keyField],
  )

  useEffect(() => {
    void load()
  }, [load])

  // ===== 运行时动态列（R05 命门）：schema 元数据 → antd columns 配置 =====
  const columns: TableColumnType<Record<string, any>>[] = shownColumns.map((col) => {
    const key = leafName(col)
    const c: TableColumnType<Record<string, any>> = {
      title: col.label,
      dataIndex: key,
      key: col.path,
      // 排序：字符串/数值通用比较器（客户端模式；服务端下推随 FE-25）。
      sorter: (a, b) => {
        const av = a[key]
        const bv = b[key]
        const an = Number(av)
        const bn = Number(bv)
        if (!Number.isNaN(an) && !Number.isNaN(bn)) return an - bn
        return String(av ?? '').localeCompare(String(bv ?? ''))
      },
      render: (_: unknown, row: Record<string, any>) => {
        // when 行级单元格（以行数据为上下文求值，失败降级可见 R08）。
        if (!cellVisible(col, row)) return <span className="cell-na">-</span>
        const tone = statusTone(row[key])
        if (tone) {
          return (
            <span className={`status-cell ${tone}`}>
              <span className="dot" aria-hidden="true" />
              {row[key]}
            </span>
          )
        }
        if (col.type === 'enum' && rowVal(row, col) !== '') {
          return <Tag color={tagColor(col, rowVal(row, col))}>{rowVal(row, col)}</Tag>
        }
        if (col.type === 'boolean') {
          return <Tag color={row[key] ? 'green' : 'default'}>{row[key] ? 'true' : 'false'}</Tag>
        }
        return rowVal(row, col)
      },
    }
    // 表头筛选（FE-11）：enum 用选项集、boolean 用 true/false。
    if (col.type === 'enum' && col.options?.length) {
      c.filters = col.options.map((o) => ({ text: String(o.label ?? o.value), value: String(o.value) }))
      c.onFilter = (v, row) => String(row[key]) === String(v)
    } else if (col.type === 'boolean') {
      c.filters = [
        { text: 'true', value: 'true' },
        { text: 'false', value: 'false' },
      ]
      c.onFilter = (v, row) => String(row[key]) === String(v)
    }
    return c
  })

  return (
    <div className="module-list-tab" data-test="module-list-tab">
      <div className="list-toolbar">
        <span className="query-at" data-test="query-at">
          {queryAt && t('console.queryAt', { time: queryAt })}
        </span>
        <Button
          size="small"
          icon={<icons.RefreshIcon />}
          onClick={() => void load(true)}
          data-test="fetch-source"
        >
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
                      setVisibleCols((prev) =>
                        e.target.checked ? [...prev, c.path] : prev.filter((p) => p !== c.path),
                      )
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
        dataSource={rows}
        loading={loading}
        rowSelection={{ selectedRowKeys: selected, onChange: (keys) => setSelected(keys) }}
        onRow={(row) => ({ onClick: () => onRowClick?.(row) })}
        locale={{
          emptyText: (
            <Empty description={device ? t('console.emptyNoConfig') : t('console.emptySelectDevice')} />
          ),
        }}
        pagination={{ defaultPageSize: 10, pageSizeOptions: [10, 20, 50], showSizeChanger: true }}
      />
    </div>
  )
}
