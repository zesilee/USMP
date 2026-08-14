import { useSyncExternalStore } from 'react'
import { Button, Dropdown, icons } from '../../ui'
import { i18n, getLocale, subscribeLocale } from '../../i18n'
import { useLocaleStore, type AppLocale } from '../../stores/locale'
import { useMenuStore } from '../../stores/menu'
import './Header.scss'

// Header（UI-01 语言切换入口 + 折叠钮）：语言选择持久化并即时联动 i18n 薄层与
// 组件库 locale（UiProvider 订阅）。新鲜度环随 tasks 11.3 在此挂载。
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

// 语言自名走词表（两份同值——语言名不随界面语言翻译，UI-02 口径下仍归词表）。
const LOCALES: { key: AppLocale; labelKey: string; testId: string }[] = [
  { key: 'zh-cn', labelKey: 'locale.zhCn', testId: 'locale-zh' },
  { key: 'en-us', labelKey: 'locale.enUs', testId: 'locale-en' },
]

export default function Header() {
  const locale = useSyncExternalStore(subscribeLocale, getLocale)
  const setLocale = useLocaleStore((s) => s.setLocale)
  const toggleCollapse = useMenuStore((s) => s.toggleCollapse)
  const isCollapsed = useMenuStore((s) => s.isCollapsed)

  return (
    <header className="app-header" data-test="app-header">
      <Button
        type="text"
        icon={isCollapsed ? <icons.ExpandIcon /> : <icons.FoldIcon />}
        onClick={toggleCollapse}
        data-test="collapse-toggle"
        title={t('nav.toggleSidebar')}
      />
      <div className="header-spacer" />
      <Dropdown
        trigger={['click']}
        menu={{
          selectedKeys: [locale],
          items: LOCALES.map((l) => ({ key: l.key, label: <span data-test={l.testId}>{t(l.labelKey)}</span> })),
          onClick: ({ key }) => setLocale(key as AppLocale),
        }}
      >
        <Button type="text" data-test="locale-switch">
          {t(LOCALES.find((l) => l.key === locale)?.labelKey ?? 'locale.zhCn')}
        </Button>
      </Dropdown>
    </header>
  )
}
