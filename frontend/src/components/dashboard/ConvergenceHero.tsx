import { i18n } from '../../i18n'
import type { Overview } from '../../composables/useFleetOverview'
import ReconcileChip from './ReconcileChip'
import './ConvergenceHero.scss'

// 设备收敛率 hero：大号收敛率 + 分段条 + 四态图例。纯展示，全部输入经 overview prop。
const t = (k: string) => i18n.global.t(k)

export default function ConvergenceHero({ overview }: { overview: Overview }) {
  // 分段条只渲染 count>0 的段，避免零宽细缝。
  const visibleSegments = overview.segments.filter((s) => s.count > 0)
  return (
    <div className="card conv" data-test="convergence-hero">
      <div className="conv-top">
        <div className="conv-lead">
          <div className="conv-pct mono">
            {overview.convergenceRate}
            <small>%</small>
          </div>
          <div className="conv-cap">
            {t('dashboard.hero.fleetPrefix')}
            <b>{t('dashboard.hero.rateWord')}</b>
            <br />
            {t('dashboard.hero.rateDesc')}
          </div>
        </div>
        {overview.pendingCount > 0 ? (
          <ReconcileChip state="recon" />
        ) : overview.total > 0 ? (
          <ReconcileChip state="conv" />
        ) : null}
      </div>

      <div className="segbar" title={t('dashboard.hero.segbarTitle')}>
        {visibleSegments.map((seg) => (
          <span key={seg.key} className={`s-${seg.key}`} style={{ flexGrow: seg.grow }} />
        ))}
        {visibleSegments.length === 0 && <span className="s-empty" style={{ flexGrow: 1 }} />}
      </div>

      <div className="legend">
        {overview.segments.map((seg) => (
          <div key={seg.key} className="legend-row">
            <span className={`k k-${seg.key}`} />
            {seg.label}
            <span className="n mono">{seg.count}</span>
          </div>
        ))}
      </div>
      {overview.unknownCount > 0 && (
        <div className="legend-foot">
          {t('dashboard.hero.unknownPre')}
          <b className="mono">{overview.unknownCount}</b>
          {t('dashboard.hero.unknownPost')}
        </div>
      )}
    </div>
  )
}
