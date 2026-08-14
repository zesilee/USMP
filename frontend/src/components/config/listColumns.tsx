import { Tag, type TableColumnType } from '../../ui'
import type { Field } from '../../utils/crdSchemaParser'
import { cellVisible, statusTone, leafName } from '../../utils/moduleConsole'

// 运行时动态列构建（FE-11，自 ModuleListTab 抽出）：schema 列元数据 → antd
// columns 配置。单元格分派：when 行级显隐（失败降级可见 R08）→ 状态点（up/down）
// → enum 色板轮转 Tag → boolean Tag → 文本。客户端模式带本地排序比较器与
// enum/boolean 表头筛选；服务端模式 sorter=true（下推由表格 onChange 处理）。

const TAG_TYPES = ['blue', 'green', 'orange', 'cyan', 'red'] as const

function tagColor(col: Field, val: string): string {
  const idx = (col.options || []).findIndex((o) => String(o.value) === val)
  return TAG_TYPES[Math.max(idx, 0) % TAG_TYPES.length]
}

export function rowVal(row: Record<string, any>, col: Field): string {
  const v = row[leafName(col)]
  return v == null ? '' : String(v)
}

export function buildDataColumns(
  shownColumns: Field[],
  serverMode: boolean,
): TableColumnType<Record<string, any>>[] {
  return shownColumns.map((col) => {
    const key = leafName(col)
    const c: TableColumnType<Record<string, any>> = {
      title: col.label,
      dataIndex: key,
      key: col.path,
      sorter: serverMode
        ? true
        : (a, b) => {
            const an = Number(a[key])
            const bn = Number(b[key])
            if (!Number.isNaN(an) && !Number.isNaN(bn)) return an - bn
            return String(a[key] ?? '').localeCompare(String(b[key] ?? ''))
          },
      render: (_: unknown, row: Record<string, any>) => {
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
    if (!serverMode) {
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
    }
    return c
  })
}
