import { Tag } from '../../ui'
import { i18n } from '../../i18n'
import type { DiffEntry } from '../../utils/configDiff'
import './DiffPreview.scss'

// DiffPreview（FE-21/FE-03）：条目相对基线的改动清单——add/modify/remove 三色。
// 空 diff 不渲染（无改动无噪音）。
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

const OP_COLOR: Record<DiffEntry['op'], string> = { add: 'green', modify: 'orange', remove: 'red' }

export default function DiffPreview({ diff }: { diff: DiffEntry[] }) {
  if (!diff.length) return null
  return (
    <div className="diff-preview" data-test="diff-preview">
      <div className="diff-title">{t('console.diffTitle')}</div>
      <ul>
        {diff.map((d) => (
          <li key={d.key}>
            <Tag color={OP_COLOR[d.op]}>{t(`console.diffOp.${d.op}`)}</Tag>
            <code>{d.key}</code>
            {d.op !== 'add' && <span className="was">{String(d.was ?? '')}</span>}
            {d.op !== 'remove' && (
              <>
                <span className="arrow" aria-hidden="true">
                  →
                </span>
                <span className="now">{String(d.now ?? '')}</span>
              </>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}
