import { Alert } from './ui'
import { i18n } from './i18n'

// React 重建期壳（tasks 3 组）：恢复可构建入口；路由/布局/页面随 tasks 8-11 组
// 于此接管。占位提示走 i18n 词表（UI-02 零硬编码中文）。
export default function App() {
  const t = i18n.global.t
  return (
    <div style={{ maxWidth: 640, margin: '96px auto', padding: '0 16px' }}>
      <Alert type="info" showIcon message={t('app.rebuildingTitle')} description={t('app.rebuildingDesc')} />
    </div>
  )
}
