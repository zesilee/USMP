import { i18n } from '../../i18n'
import './Sparkline.scss'

// 负载迷你图（纯 SVG）：数值序列 → 折线+填充区；少于 2 点无法成线 → 空态「—」。
const t = (k: string) => i18n.global.t(k)

const W = 80
const H = 26

export default function Sparkline({ points }: { points: number[] | null | undefined }) {
  if (!points || points.length < 2) {
    return (
      <span className="spark-empty" title={t('devices.noLoadTelemetry')}>
        —
      </span>
    )
  }
  const max = Math.max(...points)
  const min = Math.min(...points)
  const nx = (i: number) => (i / (points.length - 1)) * W
  const ny = (v: number) => H - 2 - ((v - min) / (max - min || 1)) * (H - 6)
  const line = points.map((v, i) => `${nx(i).toFixed(1)},${ny(v).toFixed(1)}`).join(' ')
  const area = `0,${H} ${line} ${W},${H}`
  return (
    <svg className="spark" viewBox={`0 0 ${W} ${H}`} role="img" aria-label={t('devices.loadTrend')}>
      <polygon className="fillarea" points={area} />
      <polyline points={line} />
    </svg>
  )
}
