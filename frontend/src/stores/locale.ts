import { defineStore } from 'pinia'
import { ref } from 'vue'
import { i18n } from '../i18n'

// UI-01：语言偏好（zh-cn/en-us），localStorage 持久化；非法值回退 zh-cn（R08）。
export const LOCALE_STORAGE_KEY = 'usmp-locale'
export type AppLocale = 'zh-cn' | 'en-us'
const SUPPORTED: AppLocale[] = ['zh-cn', 'en-us']

function initialLocale(): AppLocale {
  const saved = localStorage.getItem(LOCALE_STORAGE_KEY)
  return SUPPORTED.includes(saved as AppLocale) ? (saved as AppLocale) : 'zh-cn'
}

export const useLocaleStore = defineStore('locale', () => {
  const locale = ref<AppLocale>(initialLocale())
  // 建 store 即同步 i18n（持久化恢复场景）
  i18n.global.locale.value = locale.value

  function setLocale(next: AppLocale) {
    if (!SUPPORTED.includes(next)) return
    locale.value = next
    i18n.global.locale.value = next
    localStorage.setItem(LOCALE_STORAGE_KEY, next)
  }

  return { locale, setLocale }
})
