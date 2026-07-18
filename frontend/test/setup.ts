// 全局测试装配：@vue/test-utils 挂 i18n 插件（UI-02 后组件普遍依赖 useI18n）。
import { config } from '@vue/test-utils'
import { i18n } from '../src/i18n'

config.global.plugins.push(i18n)
