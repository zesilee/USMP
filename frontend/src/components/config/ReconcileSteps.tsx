import { i18n } from '../../i18n'
import { icons } from '../../ui'
import type { ReconcileProgress } from '../../utils/reconcileProgress'
import './ReconcileSteps.scss'

// 对账进度步骤条（FE-03）：pushing→reading→终局的状态机呈现；超时（未拿到终态）
// 诚实标注「仍在对账」而非成功。纯展示。
const t = (k: string) => i18n.global.t(k)

export default function ReconcileSteps({
  progress,
  timedOut,
}: {
  progress: ReconcileProgress
  timedOut?: boolean
}) {
  const resultChip = timedOut
    ? { cls: 'recon', label: t('console.steps.timeout') }
    : progress.outcome === 'converged'
      ? { cls: 'conv', label: t('console.steps.conv') }
      : progress.outcome === 'drifted'
        ? { cls: 'drift', label: t('console.steps.drift') }
        : progress.outcome === 'error'
          ? { cls: 'error', label: t('console.steps.error') }
          : null

  return (
    <div className="reconcile-steps" data-test="reconcile-steps">
      <div className="section-lbl">{t('console.steps.title')}</div>
      <div className="recon-steps">
        {progress.steps.map((s, i) => (
          <span key={s.key} className="rstep-wrap">
            <span className={`rstep ${s.state}`}>
              <span className="ico">
                {s.state === 'done' && <icons.CheckIcon />}
                {s.state === 'error' && <icons.CloseIcon />}
              </span>
              <span className="rstep-txt">
                <b>{s.title}</b>
                <span>{s.sub}</span>
              </span>
            </span>
            {i < progress.steps.length - 1 && (
              <span className={`rline${s.state === 'done' ? ' done' : ''}`} />
            )}
          </span>
        ))}
      </div>
      {resultChip && (
        <div className={`recon-result ${resultChip.cls}`} data-test="recon-result">
          <span className="glyph" />
          {resultChip.label}
        </div>
      )}
    </div>
  )
}
