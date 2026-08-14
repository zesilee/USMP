import { i18n } from '../i18n'
import './Settings.scss'

// Settings：只读架构事实（非可编辑设置——无设置持久化后端，展示系统实际策略更
// 诚实）。数值与后端一致：runningCache=TTL 30s/LRU 4096（manager.go）；端口见
// CLAUDE.md §1/§3。
const t = (k: string) => i18n.global.t(k)

interface SetRow {
  k: string
  hint: string
  v: string
  muted: boolean
}

export default function Settings() {
  const cards: { title: string; meta: string; rows: SetRow[] }[] = [
    {
      title: t('settings.protocolCard'),
      meta: '',
      rows: [
        { k: t('settings.netconfPort'), hint: t('settings.netconfPortHint'), v: '830', muted: false },
        { k: t('settings.gnmiPort'), hint: t('settings.gnmiPortHint'), v: '9339 / 9340', muted: true },
        { k: t('settings.reconnect'), hint: t('settings.reconnectHint'), v: t('common.enabled'), muted: false },
        { k: t('settings.connTimeout'), hint: t('settings.connTimeoutHint'), v: '10s', muted: false },
      ],
    },
    {
      title: t('settings.cacheCard'),
      meta: t('settings.cacheMeta'),
      rows: [
        { k: t('settings.cacheTtl'), hint: t('settings.cacheTtlHint'), v: '30s', muted: false },
        { k: t('settings.lruCapacity'), hint: t('settings.lruCapacityHint'), v: t('settings.lruCapacityValue'), muted: false },
        { k: t('settings.invalidateOnPush'), hint: t('settings.invalidateOnPushHint'), v: t('common.enabled'), muted: false },
        { k: t('settings.persistence'), hint: t('settings.persistenceHint'), v: t('common.disabled'), muted: true },
      ],
    },
  ]

  return (
    <div className="settings" data-test="settings-page">
      <div className="page-header">
        <h1>{t('settings.title')}</h1>
        <div className="sub">{t('settings.subtitle')}</div>
      </div>

      <div className="set-grid">
        {cards.map((card) => (
          <div key={card.title} className="card">
            <div className="card-h">
              <h3>{card.title}</h3>
              {card.meta && <span className="meta">{card.meta}</span>}
            </div>
            <div className="card-b">
              {card.rows.map((row) => (
                <div key={row.k} className="set-row">
                  <div className="k">
                    <b>{row.k}</b>
                    <span>{row.hint}</span>
                  </div>
                  <div className={`v${row.muted ? ' muted' : ''}`}>{row.v}</div>
                </div>
              ))}
            </div>
          </div>
        ))}
      </div>

      <div className="footnote">{t('settings.footnote')}</div>
    </div>
  )
}
