import { i18n } from '../../i18n'
import type { DisplayState } from '../../composables/useFleetOverview'
import './ReconcileChip.scss'

// 对账态 chip（纯展示）：state → 色点 + 词表文案（common.state.*）。
const t = (k: string) => i18n.global.t(k)

export default function ReconcileChip({ state }: { state: DisplayState }) {
  return (
    <span className={`reconcile-chip chip-${state}`} data-test="reconcile-chip">
      <span className="dot" aria-hidden="true" />
      {t(`common.state.${state}`)}
    </span>
  )
}
