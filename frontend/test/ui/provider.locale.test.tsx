import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen, act } from '@testing-library/react'
import { Pagination } from 'antd'
// 组 5 接线：本套件锚定 antd 后端行为（外网测试镜像），直引 antd-backend
// 版 UiProvider；生产 eview 版 provider 的联动由校准/E2E 兜底。
import { UiProvider } from '../../src/ui/antd-backend/provider'
import { useLocaleStore } from '../../src/stores/locale'

// UI-01 联动（F2）：locale store → i18n 薄层 → UiProvider（useSyncExternalStore）
// → 组件库 locale。切语言即时重渲染，无需刷新。
// 断言载体用 Pagination 的每页条数下拉文案（zh「条/页」/ en「/ page」，稳定且无弹层）。
describe('UiProvider locale 联动（UI-01）', () => {
  beforeEach(() => {
    localStorage.clear()
    useLocaleStore.getState().__resetForTest()
  })

  it('默认 zh-cn 渲染中文文案；切 en-us 即时切换；切回收尾', async () => {
    render(
      <UiProvider>
        <Pagination total={100} showSizeChanger pageSizeOptions={[10]} />
      </UiProvider>,
    )
    expect(await screen.findByTitle('10 条/页')).toBeInTheDocument()

    act(() => useLocaleStore.getState().setLocale('en-us'))
    expect(await screen.findByTitle('10 / page')).toBeInTheDocument()

    act(() => useLocaleStore.getState().setLocale('zh-cn'))
    expect(await screen.findByTitle('10 条/页')).toBeInTheDocument()
  })
})
