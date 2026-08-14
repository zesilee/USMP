import { Empty } from '../ui'
import { i18n } from '../i18n'

// React 重建期占位页（tasks 10 组逐页做实：Dashboard/Devices/Logs/Settings/Business）。
export default function PlaceholderPage({ nameKey }: { nameKey: string }) {
  const t = i18n.global.t
  return (
    <Empty
      data-test="page-placeholder"
      description={t('app.pageRebuilding', { page: t(nameKey) })}
    />
  )
}
