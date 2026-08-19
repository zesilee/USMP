import { useEffect } from 'react'
import { useNavigate } from '@app-router'
import { Alert, Button } from '../ui'
import { i18n } from '../i18n'
import { useFleetOverview } from '../composables/useFleetOverview'
import ConvergenceHero from '../components/dashboard/ConvergenceHero'
import ReconcileChip from '../components/dashboard/ReconcileChip'
import './Dashboard.scss'

// Dashboard（设备总览）：收敛率 hero + 统计栈 + 待对账台账 + 最近对账。
// 数据全部经 useFleetOverview（deriveOverview 纯函数，数据边界见其注释）。
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

function formatTime(iso: string | null): string {
  if (!iso) return '—'
  const ts = Date.parse(iso)
  if (Number.isNaN(ts)) return '—'
  return new Date(ts).toLocaleString('zh-CN', { hour12: false })
}

export default function Dashboard() {
  const navigate = useNavigate()
  const { overview: o, loading, error, load } = useFleetOverview()
  const recentTop = o.recent.slice(0, 6)

  useEffect(() => {
    void load()
  }, [load])

  return (
    <div className="dashboard" data-test="dashboard">
      <div className="ph">
        <div>
          <h1>{t('dashboard.title')}</h1>
          <div className="sub">{t('dashboard.subtitle', { total: o.total })}</div>
        </div>
        <div className="ph-actions">
          <Button loading={loading} onClick={() => void load()}>
            {t('common.refresh')}
          </Button>
          <Button type="primary" onClick={() => navigate('/module/vlan')}>
            {t('dashboard.pushConfig')}
          </Button>
        </div>
      </div>

      {error && <Alert type="error" showIcon message={t('dashboard.loadError', { error })} />}

      <div className="grid-hero">
        <ConvergenceHero overview={o} />
        <div className="stat-stack">
          <div className="card stat">
            <div className="stat-k">{t('dashboard.onlineDevices')}</div>
            <div className="stat-v mono">
              {o.online}
              <span className="u">/ {o.total}</span>
            </div>
          </div>
          <div className="card stat">
            <div className="stat-k">{t('dashboard.pendingChanges')}</div>
            <div className="stat-v mono">
              {o.pendingCount}
              {o.pendingCount > 0 ? (
                <span className="trend warn">{t('common.state.attention')}</span>
              ) : (
                <span className="trend up">{t('dashboard.allConverged')}</span>
              )}
            </div>
          </div>
          <div className="card stat">
            <div className="stat-k">{t('dashboard.unknownDevices')}</div>
            <div className="stat-v mono">
              {o.unknownCount}
              <span className="u">{t('dashboard.unknownUnit')}</span>
            </div>
          </div>
        </div>
      </div>

      <div className="grid-2">
        <div className="card">
          <div className="card-h">
            <h3>{t('dashboard.ledgerTitle')}</h3>
            <span className="meta">{t('dashboard.ledgerMeta')}</span>
          </div>
          <div className="wrap-tbl">
            <table className="tbl">
              <thead>
                <tr>
                  <th>{t('common.device')}</th>
                  <th>{t('dashboard.colOutcome')}</th>
                  <th>{t('dashboard.colLastReconcile')}</th>
                </tr>
              </thead>
              <tbody>
                {o.ledger.map((row) => (
                  <tr key={row.ip}>
                    <td>
                      <div className="strong mono">{row.ip}</div>
                    </td>
                    <td>
                      <ReconcileChip state={row.state} />
                    </td>
                    <td className="mono muted">{formatTime(row.lastRun)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {o.ledger.length === 0 && (
              <div className="empty">
                {o.total === 0 ? t('dashboard.noDevices') : t('dashboard.allConvergedEmpty')}
              </div>
            )}
          </div>
        </div>

        <div className="card">
          <div className="card-h">
            <h3>{t('dashboard.recentTitle')}</h3>
            <button className="link" onClick={() => navigate('/logs')}>
              {t('dashboard.viewAll')}
            </button>
          </div>
          <div className="wrap-tbl">
            <table className="tbl">
              <thead>
                <tr>
                  <th>{t('common.device')}</th>
                  <th>{t('dashboard.colOutcome')}</th>
                  <th>{t('dashboard.colTime')}</th>
                </tr>
              </thead>
              <tbody>
                {recentTop.map((row) => (
                  <tr key={row.ip}>
                    <td>
                      <div className="strong mono">{row.ip}</div>
                    </td>
                    <td>
                      <ReconcileChip state={row.state} />
                    </td>
                    <td className="mono muted">{formatTime(row.lastRun)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {recentTop.length === 0 && <div className="empty">{t('dashboard.noRecords')}</div>}
          </div>
        </div>
      </div>
    </div>
  )
}
