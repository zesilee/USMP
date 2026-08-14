import { useCallback, useEffect, useMemo, useState } from 'react'
import { Button, Input, Select, Table, icons, type TableColumnType } from '../ui'
import { i18n } from '../i18n'
import { getLogs } from '../api'
import { deriveLogRows, type LogRow } from '../utils/logRows'
import type { LogEntry } from '../types/api'
import { useMenuStore } from '../stores/menu'
import type { DisplayState } from '../composables/useFleetOverview'
import ReconcileChip from '../components/dashboard/ReconcileChip'
import './Logs.scss'

// Logs 页（FE-26）：审计台账——操作类型标签模型驱动派生（name→菜单标题，store
// 未加载回退段名 R08）；一次拉一批（≤500）后客户端筛选/分页。
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

function formatTime(iso: string): string {
  if (!iso) return '—'
  const d = new Date(iso)
  return isNaN(d.getTime()) ? iso : d.toLocaleString('zh-CN', { hour12: false })
}

export default function Logs() {
  const nativeModules = useMenuStore((s) => s.nativeModules)
  const loadNativeModules = useMenuStore((s) => s.loadNativeModules)
  useEffect(() => {
    void loadNativeModules()
  }, [loadNativeModules])

  // FE-26：menu store 晚于 getLogs 返回时标签自动从段名升级为菜单标题，无需重拉。
  const moduleTitles = useMemo(
    () => Object.fromEntries(nativeModules.map((m) => [m.name, m.title])),
    [nativeModules],
  )

  const [rawLogs, setRawLogs] = useState<LogEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [searchKeyword, setSearchKeyword] = useState('')
  const [statusFilter, setStatusFilter] = useState<DisplayState | ''>('')
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)

  const rows = useMemo<LogRow[]>(() => deriveLogRows(rawLogs, moduleTitles), [rawLogs, moduleTitles])

  const load = useCallback(async () => {
    setLoading(true)
    try {
      // 后端 /logs 单批上限 500（maxLogLimit）；审计量超限时最旧记录不可达（低频可接受）。
      const res = await getLogs({ limit: 500 })
      setRawLogs(res.data?.data?.logs ?? [])
    } catch {
      setRawLogs([]) // 拉取失败降级空表（R08）
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  useEffect(() => setPage(1), [searchKeyword, statusFilter])

  const filteredRows = useMemo(() => {
    let result = rows
    if (searchKeyword) {
      const kw = searchKeyword.toLowerCase()
      result = result.filter((r) => r.device.toLowerCase().includes(kw) || r.actor.toLowerCase().includes(kw))
    }
    if (statusFilter) result = result.filter((r) => r.reconcileState === statusFilter)
    return result
  }, [rows, searchKeyword, statusFilter])

  const statusOptions: { label: string; value: DisplayState }[] = [
    { label: t('common.state.conv'), value: 'conv' },
    { label: t('common.state.recon'), value: 'recon' },
    { label: t('common.state.drift'), value: 'drift' },
    { label: t('common.state.error'), value: 'error' },
    { label: t('common.state.unknown'), value: 'unknown' },
  ]

  const columns: TableColumnType<LogRow>[] = [
    { title: t('logs.colTime'), width: 180, render: (_, r) => <span className="mono dim">{formatTime(r.timestamp)}</span> },
    {
      title: t('logs.colOp'),
      width: 170,
      render: (_, r) => (
        <span className="log-op">
          <icons.DocumentIcon className="op-ico" />
          {r.opLabel}
        </span>
      ),
    },
    { title: t('logs.colDevice'), width: 150, render: (_, r) => <span className="strong">{r.device || '—'}</span> },
    { title: t('logs.colChange'), render: (_, r) => <span className="mono change">{r.summary || '—'}</span> },
    { title: t('logs.colActor'), width: 120, render: (_, r) => <span className="dim">{r.actor || '—'}</span> },
    { title: t('logs.colOutcome'), width: 130, render: (_, r) => <ReconcileChip state={r.reconcileState} /> },
  ]

  return (
    <div className="logs" data-test="logs-page">
      <div className="page-header">
        <h1>{t('logs.title')}</h1>
        <div className="header-actions">
          <Input
            allowClear
            style={{ width: 220 }}
            placeholder={t('logs.searchPlaceholder')}
            prefix={<icons.SearchIcon />}
            value={searchKeyword}
            onChange={(e) => setSearchKeyword(e.target.value)}
          />
          <Select
            allowClear
            style={{ width: 140 }}
            placeholder={t('logs.allOutcomes')}
            value={statusFilter || undefined}
            onChange={(v) => setStatusFilter((v as DisplayState) ?? '')}
            options={statusOptions}
          />
          <Button icon={<icons.RefreshIcon />} onClick={() => void load()}>
            {t('common.refresh')}
          </Button>
        </div>
      </div>

      <Table
        size="small"
        rowKey={(r) => `${r.timestamp}-${r.device}-${r.summary}`}
        columns={columns}
        dataSource={filteredRows}
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total: filteredRows.length,
          pageSizeOptions: [20, 50, 100],
          showSizeChanger: true,
          onChange: (p, ps) => {
            setPage(ps !== pageSize ? 1 : p)
            setPageSize(ps)
          },
        }}
        locale={{ emptyText: loading ? t('logs.loadingEllipsis') : t('logs.emptyNone') }}
      />
      <div className="footnote">{t('logs.footnote')}</div>
    </div>
  )
}
