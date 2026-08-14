import { useCallback, useEffect, useState } from 'react'
import { Alert, Modal, Spin, Table, Tabs } from '../../ui'
import { i18n } from '../../i18n'
import { previewChangeset, type ChangesetPreviewDataDTO } from '../../api'
import { useChangesetStore } from '../../stores/changeset'
import XmlViewer from './XmlViewer'
import './DryRunDialog.scss'

// 试运行弹窗（FE-23/CS-01）：打开即调 preview（纯计算不下发）。Tab① 待下发设备
// 数据 = 按条目正向/回滚报文双栏（无 XML 通道条目如实降级 CS-03，不伪造报文）；
// Tab② 网元数据差异对比 = diff 树 + 基线来源标注。失败如实报错且不影响变更集。
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

export default function DryRunDialog({
  open,
  device,
  onClose,
}: {
  open: boolean
  device: string
  onClose: () => void
}) {
  const changeset = useChangesetStore()
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [result, setResult] = useState<ChangesetPreviewDataDTO | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    setResult(null)
    try {
      const res = await previewChangeset(changeset.toRequest(device))
      const env = res.data as unknown as {
        success: boolean
        message?: string
        data?: ChangesetPreviewDataDTO
      }
      if (!env.success || !env.data) {
        setError(env.message || 'preview failed')
        return
      }
      setResult(env.data)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [device])

  useEffect(() => {
    if (open) void load()
  }, [open, load])

  const diffRows = result ? result.entries.flatMap((e) => e.diff ?? []) : []
  // 基线来源标注：多条目取首个非 none 来源（同请求内后端按锚点 memo）。
  const src = result?.entries.find((e) => e.baseline_source !== 'none')?.baseline_source ?? 'none'
  const baselineLabel = t(
    `console.batch.${{ desired: 'baselineDesired', cache: 'baselineCache', device: 'baselineDevice', none: 'baselineNone' }[src as string] ?? 'baselineNone'}`,
  )

  return (
    <Modal open={open} title={t('console.batch.dryRunTitle')} width="76%" footer={null} onCancel={onClose}>
      {loading && (
        <div className="dryrun-loading">
          <Spin />
        </div>
      )}
      {!loading && error && <Alert data-test="dryrun-error" type="error" showIcon message={error} />}
      {!loading && !error && result && (
        <Tabs
          items={[
            {
              key: 'payload',
              label: t('console.batch.tabPayload'),
              children: (
                <>
                  <Alert type="info" showIcon message={t('console.batch.payloadHint')} className="hint" />
                  <div className="device-name">
                    {t('console.batch.deviceName')}
                    {result.device}
                  </div>
                  {result.entries.map((e, i) => (
                    <div key={i} className="entry-block">
                      {e.unsupported ? (
                        <Alert
                          type="warning"
                          showIcon
                          message={`${e.path}：${e.unsupported_reason || t('console.batch.unsupported')}`}
                        />
                      ) : (
                        <div className="xml-panes">
                          <div className="xml-pane">
                            <div className="pane-title">{t('console.batch.forward')}</div>
                            <XmlViewer xml={e.forward_xml} />
                          </div>
                          <div className="xml-pane">
                            <div className="pane-title">{t('console.batch.rollback')}</div>
                            <XmlViewer xml={e.rollback_xml} />
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </>
              ),
            },
            {
              key: 'diff',
              label: t('console.batch.tabDiff'),
              children: (
                <>
                  <Alert type="info" showIcon message={t('console.batch.diffHint', { source: baselineLabel })} className="hint" />
                  <Table
                    data-test="dryrun-diff"
                    size="small"
                    rowKey={(r: any, i) => `${r.path}-${i}`}
                    dataSource={diffRows}
                    pagination={false}
                    columns={[
                      { title: t('console.batch.colAttr'), dataIndex: 'path' },
                      {
                        title: t('console.batch.colBefore'),
                        render: (_, r: any) => (
                          <span className={r.type === 'DELETE' ? 'cell-removed' : r.type === 'MODIFY' ? 'cell-modified' : ''}>
                            {r.old}
                          </span>
                        ),
                      },
                      {
                        title: t('console.batch.colAfter'),
                        render: (_, r: any) => (
                          <span className={r.type === 'ADD' ? 'cell-added' : r.type === 'MODIFY' ? 'cell-modified' : ''}>
                            {r.new}
                          </span>
                        ),
                      },
                    ]}
                  />
                </>
              ),
            },
          ]}
        />
      )}
    </Modal>
  )
}
