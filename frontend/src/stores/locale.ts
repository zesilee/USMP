import { create } from 'zustand'
import { setLocale as i18nSetLocale, getLocale } from '../i18n'

// UI-01：语言偏好（zh-cn/en-us），localStorage 持久化；非法值回退 zh-cn（R08）。
// zustand 版语义与旧 Pinia store 一致：建 store 即从存量恢复并同步 i18n 薄层；
// React 重渲染由 i18n 薄层的 subscribeLocale 驱动（UiProvider useSyncExternalStore）。
export const LOCALE_STORAGE_KEY = 'usmp-locale'
export type AppLocale = 'zh-cn' | 'en-us'
const SUPPORTED: AppLocale[] = ['zh-cn', 'en-us']

function initialLocale(): AppLocale {
  const saved = localStorage.getItem(LOCALE_STORAGE_KEY)
  return SUPPORTED.includes(saved as AppLocale) ? (saved as AppLocale) : 'zh-cn'
}

interface LocaleState {
  locale: AppLocale
  setLocale: (next: AppLocale) => void
  /** 测试隔离：按当前 localStorage 重算初始态（beforeEach 用）。 */
  __resetForTest: () => void
}

function boot(): AppLocale {
  const l = initialLocale()
  i18nSetLocale(l) // 持久化恢复场景：建 store 即同步 i18n
  return l
}

export const useLocaleStore = create<LocaleState>((set) => ({
  locale: boot(),
  setLocale: (next) => {
    if (!SUPPORTED.includes(next)) return
    set({ locale: next })
    i18nSetLocale(next)
    localStorage.setItem(LOCALE_STORAGE_KEY, next)
  },
  __resetForTest: () => set({ locale: boot() }),
}))

// 便于非组件上下文读取（与 Pinia 时代 store.locale 直读对齐）。
export function currentLocale(): AppLocale {
  return (getLocale() as AppLocale) ?? useLocaleStore.getState().locale
}
