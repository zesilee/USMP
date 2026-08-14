import zhCn from '../locales/zh-cn.json'
import enUs from '../locales/en-us.json'

// UI-01/02 框架无关实现（React 重建期铺路）：对纯逻辑层暴露与旧 vue-i18n 完全
// 同形的 `i18n.global.t(key, params?)` API，使 utils/composables 沿用零改动（D4）。
// 词表 = 同一批 locales JSON；插值语法沿用 `{name}`；缺 key 回退 zh-cn 再回退
// key 本身（R08 界面不空白）。React 层的语言切换（UI-01）后续在此模块上扩展，
// 不引第三方 i18n 运行时——词表查表 + 插值本就只有十几行。

type Messages = Record<string, unknown>

const messages: Record<string, Messages> = { 'zh-cn': zhCn, 'en-us': enUs }

let locale = 'zh-cn'

// 语言变更订阅（external-store 形态）：React 侧（UiProvider/useT）经
// useSyncExternalStore 接入，语言切换即时重渲染（UI-01）；非 React 消费方
// （utils 的 t()）每次调用取当前值，无需订阅。
const listeners = new Set<() => void>()

/** 设置当前语言（默认 zh-cn；未知语言忽略，R08）并通知订阅方。 */
export function setLocale(next: string): void {
  if (!messages[next] || next === locale) return
  locale = next
  listeners.forEach((l) => l())
}

export function getLocale(): string {
  return locale
}

/** 订阅语言变更（返回退订函数），供 useSyncExternalStore 使用。 */
export function subscribeLocale(listener: () => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

// 点分 key 逐层取值：'console.basicTab' → messages[locale].console.basicTab。
function lookup(msgs: Messages | undefined, key: string): unknown {
  return key
    .split('.')
    .reduce<unknown>((node, seg) => (node && typeof node === 'object' ? (node as Messages)[seg] : undefined), msgs)
}

// `{name}` 命名插值（与旧 vue-i18n 词表语法一致）；缺参保留原样便于排查；
// null/undefined 参数值渲染空串（对齐旧 vue-i18n toDisplayString 语义——否则
// `t('devices.addFailed', { reason: err.message })` 在 message 缺失时界面出现
// 字面 "undefined"）。
function interpolate(tpl: string, params?: Record<string, unknown>): string {
  if (!params) return tpl
  return tpl.replace(/\{(\w+)\}/g, (raw, name: string) =>
    name in params ? (params[name] == null ? '' : String(params[name])) : raw,
  )
}

function t(key: string, params?: Record<string, unknown>): string {
  const hit = lookup(messages[locale], key) ?? lookup(messages['zh-cn'], key)
  return typeof hit === 'string' ? interpolate(hit, params) : key
}

export const i18n = { global: { t } }
