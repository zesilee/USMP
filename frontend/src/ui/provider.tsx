// UI 适配层 · 根 Provider（FA-01/FA-03/FA-04）：主题令牌 + 组件库 locale +
// 命令式反馈实例挂载，三件收口于此。应用入口与测试装配都用它包根。
// locale 联动（UI-01）：跟随 i18n 薄层 getLocale()；语言切换的响应式外壳
// 随 tasks 5.4 在此扩展。
import { useEffect, type ReactNode } from 'react'
import { App as AntApp, ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import enUS from 'antd/locale/en_US'
import { getLocale } from '../i18n'
import { antdTheme } from './tokens'
import { __bindFeedback } from './feedback'

// App.useApp() 只能在 <AntApp> 内调用——内层小组件负责把带上下文的
// message/modal 实例绑给模块级 feedback API。绑定放 useEffect：render 期写
// 模块变量在并发模式下可随丢弃的 render 泄漏（评审 #4）；卸载时解绑回退
// 静态降级，避免 stale 实例。
function FeedbackBinder({ children }: { children: ReactNode }) {
  const { message, modal } = AntApp.useApp()
  useEffect(() => {
    __bindFeedback(message, modal)
    return () => __bindFeedback(null, null)
  }, [message, modal])
  return <>{children}</>
}

export function UiProvider({ children }: { children: ReactNode }) {
  const locale = getLocale() === 'en-us' ? enUS : zhCN
  return (
    <ConfigProvider locale={locale} theme={antdTheme}>
      <AntApp>
        <FeedbackBinder>{children}</FeedbackBinder>
      </AntApp>
    </ConfigProvider>
  )
}
