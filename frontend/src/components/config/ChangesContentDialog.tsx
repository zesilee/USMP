import { Empty, Modal, Table } from '../../ui'
import { i18n } from '../../i18n'
import { useChangesetStore, type ChangesetEntry } from '../../stores/changeset'
import './ChangesContentDialog.scss'

// 变更内容弹窗（FE-23）：纯前端渲染当前设备变更集——树形三列（属性/变更前/
// 变更后）+ 增/改/删图例计数（绿/黄/红），对齐 NCE 形态。语义自旧版逐行平移。
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

interface Row {
  id: string
  attr: string
  before?: string
  after?: string
  kind?: 'add' | 'modify' | 'remove'
  children?: Row[]
}

const show = (v: unknown) => (v === undefined || v === null ? '' : String(v))

// 条目 → 树行：update 逐字段对比基线（等值跳过、cleared 叶为删除行）；
// create 全字段为新值；delete 展示基线旧值（无基线仅键定位）。
function fieldRows(e: ChangesetEntry, idx: number): Row[] {
  const base = (e.baseline ?? {}) as Record<string, unknown>
  const payload = (e.payload ?? {}) as Record<string, unknown>
  const rows: Row[] = []
  if (e.op === 'delete') {
    const keys = Object.keys(base)
    if (keys.length === 0) {
      rows.push({ id: `e${idx}-key`, attr: e.keyValue ?? '', before: show(e.keyValue), kind: 'remove' })
    }
    for (const k of keys) {
      rows.push({ id: `e${idx}-${k}`, attr: k, before: show(base[k]), kind: 'remove' })
    }
    return rows
  }
  for (const [k, v] of Object.entries(payload)) {
    const was = base[k]
    if (e.op === 'create' || was === undefined || was === null || was === '') {
      if (show(v) !== '') rows.push({ id: `e${idx}-${k}`, attr: k, after: show(v), kind: 'add' })
      continue
    }
    if (show(v) !== show(was)) {
      rows.push({ id: `e${idx}-${k}`, attr: k, before: show(was), after: show(v), kind: 'modify' })
    }
  }
  for (const k of e.cleared ?? []) {
    const was = base[k]
    if (was !== undefined && was !== null && show(was) !== '') {
      rows.push({ id: `e${idx}-clr-${k}`, attr: k, before: show(was), kind: 'remove' })
    }
  }
  return rows
}

export default function ChangesContentDialog({
  open,
  device,
  onClose,
}: {
  open: boolean
  device: string
  onClose: () => void
}) {
  const changeset = useChangesetStore()
  const summary = changeset.summaryFor(device)
  const rows: Row[] = changeset.entriesFor(device).map((e, i) => ({
    id: `e${i}`,
    attr: e.label ?? `${e.path}${e.keyValue ? ` [${e.keyValue}]` : ''}`,
    children: fieldRows(e, i),
  }))

  return (
    <Modal open={open} title={t('console.batch.changes')} width="72%" footer={null} onCancel={onClose}>
      <div data-test="changes-legend" className="legend">
        <span className="legend-item added">{t('console.batch.legendAdd', { n: summary.creates })}</span>
        <span className="legend-item modified">{t('console.batch.legendModify', { n: summary.updates })}</span>
        <span className="legend-item removed">{t('console.batch.legendDelete', { n: summary.deletes })}</span>
      </div>
      {rows.length ? (
        <Table<Row>
          data-test="changes-table"
          size="small"
          rowKey="id"
          dataSource={rows}
          pagination={false}
          expandable={{ defaultExpandAllRows: true }}
          columns={[
            { title: t('console.batch.colAttr'), dataIndex: 'attr' },
            {
              title: t('console.batch.colBefore'),
              render: (_, row) => (
                <span className={row.kind === 'remove' ? 'cell-removed' : row.kind === 'modify' ? 'cell-modified' : ''}>
                  {row.before}
                </span>
              ),
            },
            {
              title: t('console.batch.colAfter'),
              render: (_, row) => (
                <span className={row.kind === 'add' ? 'cell-added' : row.kind === 'modify' ? 'cell-modified' : ''}>
                  {row.after}
                </span>
              ),
            },
          ]}
        />
      ) : (
        <Empty description={t('console.batch.empty')} />
      )}
    </Modal>
  )
}
