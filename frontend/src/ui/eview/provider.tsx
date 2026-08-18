// UI 适配层 · 根 Provider（FA-01/FA-03/FA-04，组 5 接线 eview 版）：
// 1) intl 上下文——EviewUI 编译产物内部 require('react-intl') 且组件 contextType
//    读 intl（校准 R3 定案），根上必须包 IntlProvider；zh 语言包静态引入真包
//    locales（外网测试经 vitest 别名到空 stub）。en 缺档走组件 defaultMessage。
// 2) findDOMNode polyfill——React 19 移除该 API 而 eview Dialog/Drawer 内部调用
//    （校准 R3 实证），生产链路在此安装（波 C 切 openinula 后随之退役）。
// 3) 主题注入——eview design-token 的 CSS 变量覆盖（theme.ts 十档色阶），幂等。
// locale 联动（UI-01）：订阅 i18n 薄层——语言切换即时重渲染整树。
// 测试口径：外网 vitest 经 @ui-backend 解析到 antd-backend/provider（含 App
// 实例 feedback 绑定——静态 message 残留定时器曾致 teardown 后炸，见其 README）。
import { useEffect, type ReactNode } from 'react'
import { IntlProvider } from 'react-intl'
import zhMessages from '@nce/eview-react/locales/zh'
import { useLocale } from '../../i18n'
import { injectTokenOverride } from './theme'
import { installFindDOMNodePolyfill } from '../../runtime/finddomnode-polyfill'

installFindDOMNodePolyfill()

const messagesByLocale: Record<string, Record<string, string>> = {
  'zh-cn': ((zhMessages as { default?: Record<string, string> })?.default ??
    (zhMessages as Record<string, string>) ??
    {}) as Record<string, string>,
  // en 语言包真包内是否存在未证实——缺档时 intl 回退 defaultMessage，不炸（R08）。
  'en-us': {},
}

export function UiProvider({ children }: { children: ReactNode }) {
  const current = useLocale()
  useEffect(() => {
    injectTokenOverride()
  }, [])
  return (
    <IntlProvider
      locale={current === 'en-us' ? 'en' : 'zh'}
      messages={messagesByLocale[current] ?? messagesByLocale['zh-cn']}
      onError={() => undefined}
    >
      {children}
    </IntlProvider>
  )
}
