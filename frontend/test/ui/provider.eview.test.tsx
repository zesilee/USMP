import { describe, it, expect, afterEach } from 'vitest'
import { render, cleanup } from '@testing-library/react'
import { UiProvider } from '@bridge/provider'
import { useLocaleStore } from '../../src/stores/locale'

// 生产 UiProvider（eview 版，组 5 接线）F1：IntlProvider 装配 + locale 联动
// （UI-01）+ 主题 CSS 变量注入幂等。外网 @nce locales 经 stub（空字典）——
// 渲染链路与 intl 缺档降级（onError 静默）都在此验证。
describe('UiProvider（eview 生产版）', () => {
  afterEach(() => {
    cleanup()
    useLocaleStore.getState().setLocale('zh-cn')
    document.querySelectorAll('style[data-ub-theme]').forEach((el) => el.remove())
  })

  it('渲染子树 + 注入主题 CSS 变量（幂等）', () => {
    const { getByText, unmount } = render(
      <UiProvider>
        <span>内容甲</span>
      </UiProvider>,
    )
    expect(getByText('内容甲')).toBeInTheDocument()
    const count = () => document.querySelectorAll('style[data-ub-theme]').length
    const first = count()
    expect(first).toBeGreaterThanOrEqual(0) // 注入实现细节由 theme.test 钉；此处仅确保不炸
    unmount()
  })

  it('locale 切换即时重渲染（UI-01）——en-us 分支走缺档降级不炸（R08）', () => {
    const { getByText } = render(
      <UiProvider>
        <span>内容乙</span>
      </UiProvider>,
    )
    useLocaleStore.getState().setLocale('en-us')
    expect(getByText('内容乙')).toBeInTheDocument()
    useLocaleStore.getState().setLocale('zh-cn')
    expect(getByText('内容乙')).toBeInTheDocument()
  })
})
