import { describe, it, expect, beforeEach } from 'vitest'
import { render, cleanup } from '@testing-library/react'
import { useState, type ReactElement, createElement } from 'react'
import { Tabs, Menu } from '@bridge/components/structure'
import { installFindDOMNodePolyfill } from '../../src/runtime/finddomnode-polyfill'
import { IntlProvider } from 'react-intl'
import zhMessages from '@nce/eview-react/locales/zh'

// ============ F3 真浏览器 · EviewUI 桥校准（组 6.1，内网专用） ============
// happy-dom 校准移交项（CAL-R7/R9 定案）：eview Tab 的 activeKey 更新路径与
// Tree 的首次渲染在 happy-dom 同步死循环（内部布局循环依赖真实元素宽度，
// happy-dom 恒 0 不收敛）。真 Chromium 有真实布局，循环可收敛——在此校准。
// 内网运行：EVIEW_REAL=1 npx vitest run --config vitest.browser.config.ts \
//   test/browser/eview-real-browser.test.tsx > bridge-report-f3.txt 2>&1
// 外网默认 skip（@ui-backend 指 antd 镜像，@bridge 真桥经 @nce 无从解析——
// 故本套件连 import 都以 REAL 门禁：见 vitest.browser.config 的 include 说明）。
// 真浏览器无 node process/require——REAL 开关由 vitest.browser.config 的
// define 注入；react-intl 与 locales 走顶层静态 import（外网 locales 经
// stub 别名解析为空字典，react-intl 真包两侧都有）。
declare const __EVIEW_REAL__: boolean
const REAL = __EVIEW_REAL__
const d = REAL ? describe : describe.skip

if (REAL) {
  installFindDOMNodePolyfill()
  // 报告减脂（F3-R2：回传文件过大）：真浏览器把 React/eview 的弃用与
  // 渲染警告全量转发进报告——同类消息（按前 60 字符归组）只放行前 2 条，
  // 其余吞掉计数；F3- 打点不受影响（走 console.log 原样）。
  const seen = new Map<string, number>()
  const throttle = (orig: (...a: unknown[]) => void) => (...args: unknown[]) => {
    const key = String(args[0]).slice(0, 60)
    const n = (seen.get(key) ?? 0) + 1
    seen.set(key, n)
    if (n <= 2) orig(...args)
    else if (n === 3) orig(`[节流] 同类消息继续出现，后续吞掉: ${key}…`)
  }
  console.warn = throttle(console.warn.bind(console))
  console.error = throttle(console.error.bind(console))
  // 入口打点（F3-R2 教训：报告只见 Tree 无 Tabs，执行序不明）——挂死轮次
  // 可精确显示进入了哪个用例。
  beforeEach(() => {
    // eslint-disable-next-line no-console
    console.log(`F3-ENTER: ${expect.getState().currentTestName ?? '?'}`)
  })
}

// eview 组件 contextType 读 intl——真浏览器同样需要 IntlProvider 包根。
let wrapIntl = (el: ReactElement): ReactElement => el
if (REAL) {
  const messages = ((zhMessages as { default?: Record<string, string> })?.default ??
    zhMessages ??
    {}) as Record<string, string>
  wrapIntl = (el) =>
    createElement(IntlProvider as never, { locale: 'zh', messages, onError: () => undefined } as never, el)
}
const renderReal = (el: ReactElement) => render(wrapIntl(el))

// 哨兵：外网 skip 模式下保证文件非空。
it(`eview-real-browser 模式=${REAL ? 'REAL（内网真浏览器校准）' : 'SKIP（外网，EVIEW_REAL=1 启用）'}`, () => {
  expect(true).toBe(true)
})

d('F3 真浏览器 · Tabs（happy-dom 移交项）', () => {
  it('受控切换：rerender 改 activeKey 不挂死且内容区切换', async () => {
    const items = [
      { key: 'a', label: '甲页', children: <p>content-a</p> },
      { key: 'b', label: '乙页', children: <p>content-b</p> },
    ]
    const { container, rerender } = renderReal(<Tabs items={items} activeKey="a" onChange={() => {}} />)
    expect(container.textContent).toContain('content-a')
    console.log('F3-TABS: 即将 rerender(activeKey=b)')
    rerender(wrapIntl(<Tabs items={items} activeKey="b" onChange={() => {}} />))
    console.log('F3-TABS: rerender 完成，含 content-b=', container.textContent?.includes('content-b'))
    expect(container.textContent).toContain('content-b')
    cleanup()
  })

  it('点击切换：真实点击驱动 onChange 回写', async () => {
    function Host() {
      const [k, setK] = useState('a')
      return (
        <Tabs
          items={[
            { key: 'a', label: '甲页', children: <p>content-a</p> },
            { key: 'b', label: '乙页', children: <p>content-b</p> },
          ]}
          activeKey={k}
          onChange={setK}
        />
      )
    }
    const { container } = renderReal(<Host />)
    const tabB = Array.from(container.querySelectorAll('*')).find(
      (el) => el.textContent === '乙页' && el.children.length === 0,
    ) as HTMLElement | undefined
    expect(tabB).toBeTruthy()
    console.log('F3-TABS: 即将真实点击乙页')
    tabB!.click()
    await new Promise((r) => setTimeout(r, 300))
    console.log('F3-TABS: 点击后含 content-b=', container.textContent?.includes('content-b'))
    expect(container.textContent).toContain('content-b')
    cleanup()
  })
})

d('F3 真浏览器 · Menu→Tree（happy-dom 移交项）', () => {
  it('首次渲染不挂死 + 受控展开 + 叶子选中回调', async () => {
    const clicks: string[] = []
    function Host() {
      const [open, setOpen] = useState<string[]>([])
      return (
        <div>
          <button data-probe="expand" onClick={() => setOpen(['g1'])}>expand-g1</button>
          <Menu
            items={[{ key: 'g1', label: '分组一', children: [{ key: 'leaf1', label: '叶子一' }] }]}
            openKeys={open}
            onOpenChange={setOpen}
            selectedKeys={[]}
            onClick={(i) => clicks.push(i.key)}
          />
        </div>
      )
    }
    console.log('F3-TREE: 即将首次渲染（happy-dom 曾在此同步死循环）')
    const { container, getByText } = renderReal(<Host />)
    console.log('F3-TREE: 首渲完成 DOM 长度=', container.innerHTML.length)
    expect(container.textContent).toContain('分组一')
    console.log('F3-TREE: 即将派发展开点击（R2 卡死点——桥已上 key 重挂绕行）')
    getByText('expand-g1').click()
    console.log('F3-TREE: 展开点击已派发')
    await new Promise((r) => setTimeout(r, 300))
    console.log('F3-TREE: 展开后含叶子一=', container.textContent?.includes('叶子一'))
    // R3 小尾巴：.ev_tree_text 点击无回调——监听点多目标逐试并报告命中
    // （Switch 校准同款方法论）。
    const leafText = Array.from(container.querySelectorAll('*')).find(
      (el) => el.textContent === '叶子一' && el.children.length === 0,
    ) as HTMLElement | undefined
    if (leafText) {
      const targets: Array<[string, HTMLElement | null]> = [
        ['text', leafText],
        ['parent', leafText.parentElement],
        ['grandparent', leafText.parentElement?.parentElement ?? null],
      ]
      for (const [name, el] of targets) {
        if (!el || clicks.length) break
        el.click()
        await new Promise((r) => setTimeout(r, 150))
        console.log(`F3-TREE: 点 ${name}(${el.className}) 后 keys=${JSON.stringify(clicks)}`)
      }
      console.log('F3-TREE: 叶子点击终态 keys=', JSON.stringify(clicks), '（应含 leaf1）')
    } else {
      console.log('F3-TREE: 未找到叶子文本节点——展开渲染形态需按本轮 DOM 校准')
    }
    cleanup()
  })
})
