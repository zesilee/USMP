import { createI18n } from 'vue-i18n'
import zhCn from '../locales/zh-cn.json'
import enUs from '../locales/en-us.json'

// UI-01/02：composition 模式（legacy:false）；chrome 文案按域组织（nav/devices/…）。
// 缺 key 回退 zh-cn 再回退 key 本身（R08 界面不空白）。
export const i18n = createI18n({
  legacy: false,
  locale: 'zh-cn',
  fallbackLocale: 'zh-cn',
  messages: { 'zh-cn': zhCn, 'en-us': enUs },
  missingWarn: false,
  fallbackWarn: false,
})
