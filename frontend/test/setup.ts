// 全局测试装配：@vue/test-utils 挂 i18n 插件（UI-02 后组件普遍依赖 useI18n）。
import { config, enableAutoUnmount } from '@vue/test-utils'
import { afterEach } from 'vitest'
import { i18n } from '../src/i18n'

config.global.plugins.push(i18n)

// 用例后自动卸载挂载体：重组件（el-table 排序/筛选/详情区）不卸载会让活跃
// watcher 跨用例累积，单文件后段用例被拖到超时（NCE 改版期实测递增 2s/例）。
// 仅 happy-dom 环境启用——browser 模式用例自带 wrapper.unmount()，全局再卸一次
// 会 removeChild 双重卸载报错。
if (!(globalThis as any).__vitest_browser__) {
  enableAutoUnmount(afterEach)
}
