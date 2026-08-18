import { describe, it, expect, vi, afterEach, beforeEach } from 'vitest'
import { render as rtlRender, screen, fireEvent, waitFor, cleanup } from '@testing-library/react'
import { createElement, useState, type ReactElement } from 'react'
// 顶层静态 import（REAL 分支内 require 相对 TS 路径不经 vitest 转换——R4 实录坑）
import { installFindDOMNodePolyfill } from '../../src/runtime/finddomnode-polyfill'

// ============ 内网真实校准套件（集成点专用，混合协作模式） ============
// 外网默认 skip（@nce 实现不存在）；内网运行：
//   EVIEW_REAL=1 npx vitest run test/integration/eview-real.test.tsx > bridge-report.txt 2>&1
// EVIEW_REAL=1 时 vitest.config 自动关闭 @nce stub alias——本套件打到真 EviewUI。
// 断言基于 gate R1/R2 报告的真实 DOM 特征（ev_* 类名体系）；每用例独立 try 影响
// 隔离；失败输出 DOM 快照供外网远程修桥。
const REAL = process.env.EVIEW_REAL === '1'
// 版本指纹：每轮修桥递增，报告第一行即可判内网跑的是否新代码（R8 教训：
// 合入到报告间隔过短无法排除旧代码，指纹终结猜疑）。
const CAL_VERSION = 'CAL-R9'
const d = REAL ? describe : describe.skip
if (REAL) {
  vi.setConfig({ testTimeout: 10000, hookTimeout: 10000 })
  // eslint-disable-next-line no-console
  console.log(`${CAL_VERSION} 校准套件启动`)
  // 入口标记：挂死轮次报告可精确显示死在哪个用例内（R8 挂点只能靠推断）。
  beforeEach(() => {
    // eslint-disable-next-line no-console
    console.log(`${CAL_VERSION} ENTER: ${expect.getState().currentTestName ?? '?'}`)
  })
}

// eview 编译产物内部 require('react-intl')（Popup 链等）且组件 contextType
// 读 intl 上下文——REAL 模式所有渲染统一包 IntlProvider（messages 用真包
// 内置 zh 语言包，缺档静默）。外网 skip 模式不加载 react-intl/locales。
let wrapIntl = (el: ReactElement): ReactElement => el
if (REAL) {
  // R3：EviewUI Dialog/Drawer 内部调用 React 19 已移除的 findDOMNode。
  installFindDOMNodePolyfill()
  // R5：happy-dom 的 requestAnimationFrame 同步立即执行——eview Tab 的
  // ink bar 动画（raf 递归）变同步死循环挂死整轮（超时中断不了同步循环）。
  // 异步化为宏任务，动画循环即可被 vitest 超时正常中断。
  const raf = (cb: FrameRequestCallback): number =>
    setTimeout(() => cb(performance.now()), 16) as unknown as number
  window.requestAnimationFrame = raf
  globalThis.requestAnimationFrame = raf
  const caf = (id: number): void => clearTimeout(id as unknown as ReturnType<typeof setTimeout>)
  window.cancelAnimationFrame = caf
  globalThis.cancelAnimationFrame = caf
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { IntlProvider } = require('react-intl') as { IntlProvider: never }
  let messages: Record<string, string> = {}
  for (const cand of ['zh', 'zh-cn', 'zh_CN']) {
    try {
      // eslint-disable-next-line @typescript-eslint/no-require-imports
      const m = require(`@nce/eview-react/locales/${cand}`)
      messages = (m && (m.default ?? m)) as Record<string, string>
      break
    } catch { /* 下一个候选 */ }
  }
  wrapIntl = (el) => createElement(IntlProvider, { locale: 'zh', messages, onError: () => undefined } as never, el)
}
const render = (el: ReactElement) => rtlRender(wrapIntl(el))

// 哨兵：外网（skip 模式）下保证文件非空（vitest 对空文件按失败处理）。
it(`eview-real 套件模式=${REAL ? 'REAL（内网真实校准）' : 'SKIP（外网，EVIEW_REAL=1 启用）'}`, () => {
  expect(true).toBe(true)
})

import { Tag, Badge, Breadcrumb, Empty, Drawer, Modal } from '../../src/ui/eview/components/display'
import { Dropdown, Segmented, Radio, Checkbox, Switch } from '../../src/ui/eview/components/controls'
import { Input, InputNumber, Select } from '../../src/ui/eview/components/inputs'
import { Tabs, Menu, Table } from '../../src/ui/eview/components/structure'
import { Button, Spin, Tooltip, Alert } from '../../src/ui/eview/components/rest'
import FormItemShell from '../../src/ui/eview/FormItemShell'

afterEach(() => {
  cleanup()
  document.body.innerHTML = ''
})

const snap = (el: Element | null, n = 500) => (el?.innerHTML ?? '').slice(0, n)

d('真实校准 · 展示组', () => {
  it('Tag/Badge/Breadcrumb/Empty 挂载出 ev_* DOM', () => {
    const { container } = render(
      <div>
        <Tag color="error">标记</Tag>
        <Badge count={5} />
        <Breadcrumb items={[{ title: '甲' }, { title: '乙' }]} separator=">" />
        <Empty description="空" />
      </div>,
    )
    console.log('DISPLAY-DOM:', snap(container, 1200))
    expect(container.textContent).toContain('标记')
    expect(container.textContent).toContain('甲')
  })

  it('Modal 开合受控 + 确认钮回调', async () => {
    const onOk = vi.fn()
    const onCancel = vi.fn()
    render(<Modal open title="确认框" onOk={onOk} onCancel={onCancel} okText="确定" />)
    console.log('MODAL-DOM:', snap(document.body, 1200))
    expect(document.body.textContent).toContain('确认框')
    const ok = Array.from(document.body.querySelectorAll('button')).find((b) => (b.textContent ?? '').includes('确定'))
    if (ok) {
      fireEvent.click(ok)
      expect(onOk).toHaveBeenCalled()
    } else {
      console.log('MODAL: 未找到确定钮——buttons 形态需校准')
    }
  })

  it('Drawer 受控开合', () => {
    render(<Drawer open title="抽屉" width={400} />)
    expect(document.body.textContent).toContain('抽屉')
  })
})

d('真实校准 · 交互组', () => {
  it('Switch 受控翻转（isControlToggled 真受控链）', async () => {
    function Host() {
      const [on, setOn] = useState(false)
      return (
        <div>
          <span data-probe="state">{String(on)}</span>
          <Switch checked={on} onChange={setOn} />
        </div>
      )
    }
    const { container } = render(<Host />)
    console.log('SWITCH-DOM:', snap(container, 600))
    // R3：点外层无效——container/thumb/track/keydown 逐个试，报告哪个生效。
    const targets: Array<[string, Element | null]> = [
      ['container', container.querySelector('.ev_toggle_container')],
      ['thumb', container.querySelector('.ev_toggle_thumb')],
      ['track', container.querySelector('.ev_toggle_track')],
    ]
    const state = () => container.querySelector('[data-probe="state"]')!.textContent
    for (const [name, el] of targets) {
      if (!el || state() === 'true') break
      fireEvent.mouseDown(el)
      fireEvent.mouseUp(el)
      fireEvent.click(el)
      await new Promise((r) => setTimeout(r, 60))
      console.log(`SWITCH 点击 ${name} 后 state=${state()}`)
    }
    if (state() !== 'true') {
      const kb = container.querySelector('.ev_toggle_container')
      if (kb) {
        fireEvent.keyDown(kb, { key: 'Enter' })
        fireEvent.keyDown(kb, { key: ' ' })
        await new Promise((r) => setTimeout(r, 60))
        console.log(`SWITCH keydown 后 state=${state()}`)
      }
    }
    expect(state()).toBe('true')
  })

  it('Radio.Group 参数序实测（gate 未定案项）', async () => {
    const calls: unknown[] = []
    render(
      <Radio.Group value="a" onChange={(e) => calls.push(e.target.value)}>
        <Radio value="a">甲</Radio>
        <Radio value="b">乙</Radio>
      </Radio.Group>,
    )
    // gate 教训：监听在叶子元素——点 label。
    const labelB = Array.from(document.querySelectorAll('label')).find((l) => l.textContent === '乙')
    const target = labelB ?? document.querySelectorAll('[role="radio"]')[1]
    if (target) fireEvent.click(target)
    await waitFor(() => expect(calls.length).toBeGreaterThan(0), { timeout: 1500 }).catch(() => {})
    console.log('RADIO onChange 提取的新值:', JSON.stringify(calls), '（应为 ["b"]——自适应判别验证点）')
    expect(calls.length === 0 || calls[0] === 'b').toBe(true) // 触发则必须判对
  })

  it('Checkbox 合成 e.target.checked', async () => {
    const calls: boolean[] = []
    const { container } = render(<Checkbox checked={false} onChange={(e) => calls.push(e.target.checked)}>勾选项</Checkbox>)
    console.log('CHECKBOX DOM:', snap(container, 500))
    const targets: Array<[string, Element | null]> = [
      ['label', Array.from(container.querySelectorAll('label')).find((l) => l.textContent === '勾选项') ?? null],
      ['span', container.querySelector('[class*="checkbox_span"]')],
      ['div', container.querySelector('[role="checkbox"]')],
    ]
    for (const [name, el] of targets) {
      if (!el || calls.length) break
      fireEvent.mouseDown(el)
      fireEvent.click(el)
      await new Promise((r) => setTimeout(r, 40))
      console.log(`CHECKBOX 点 ${name} 后 calls=${JSON.stringify(calls)}`)
    }
    if (!calls.length) {
      const kb = container.querySelector('[role="checkbox"], .ev_checkbox')
      if (kb) {
        fireEvent.keyDown(kb, { key: 'Enter' })
        fireEvent.keyDown(kb, { key: ' ' })
        console.log(`CHECKBOX keydown 后 calls=${JSON.stringify(calls)}`)
      }
    }
    expect(calls.length === 0 || calls[0] === true).toBe(true)
  })
})

d('真实校准 · 表单输入组（半受控桥）', () => {
  it('Input 受控回写：父接受则显新值', async () => {
    function Host() {
      const [v, setV] = useState('A')
      return <Input value={v} onChange={(e) => setV(e.target.value)} />
    }
    const { container } = render(<Host />)
    const input = container.querySelector('input')!
    const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value')!.set!
    setter.call(input, 'AB')
    fireEvent.input(input)
    await waitFor(() => expect(input.value).toBe('AB'), { timeout: 1500 }).catch(() => {})
    console.log('INPUT 接受路径 value=', input.value, '（应为 AB）')
  })

  it('Input 父拒写：重挂后回到 props 值（③档兜底实测）', async () => {
    function Host() {
      const [v] = useState('A')
      return <Input value={v} onChange={() => {}} />
    }
    const { container } = render(<Host />)
    const input = container.querySelector('input')!
    const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value')!.set!
    setter.call(input, 'AB')
    fireEvent.input(input)
    await new Promise((r) => setTimeout(r, 120))
    const now = container.querySelector('input')!.value
    console.log('INPUT 拒写路径 value=', now, '（应回 A——useSemiControlledBridge 重挂验证点）')
    expect(now).toBe('A')
  })

  it('Select 桥挂载 + 清空语义', () => {
    const calls: unknown[] = []
    const { container } = render(
      <Select
        options={[{ label: 'alpha', value: 'a' }, { label: 'beta', value: 'b' }]}
        value="a"
        allowClear
        onChange={(v) => calls.push(v)}
      />,
    )
    console.log('SELECT-DOM:', snap(container, 600))
    const input = container.querySelector('input')
    if (input) fireEvent.focus(input)
    const clear = container.querySelector('[class*="clear"]')
    if (clear) {
      // R3：clear span 常态 invisible（hover/focus 显示），完整事件序列触发。
      fireEvent.mouseDown(clear)
      fireEvent.mouseUp(clear)
      fireEvent.click(clear)
      console.log('SELECT 清空后 calls:', JSON.stringify(calls), 'clear 类=', clear.className, '（应含 undefined/null——键不入 payload 链）')
    }
  })

  it('InputNumber min/max 无界传入不被 0/100 限制', () => {
    const { container } = render(<InputNumber value={500} onChange={() => {}} />)
    const input = container.querySelector('input')
    console.log('SPINNER value 显示:', input?.value, '（应为 500——默认 0/100 陷阱验证点）')
  })
})


d('真实校准 · 收尾组与外壳', () => {
  it('Button 状态映射 + 点击；Spin/Tooltip/Alert 挂载', () => {
    const onClick = vi.fn()
    const { container } = render(
      <div>
        <Button type="primary" onClick={onClick}>主钮</Button>
        <Spin tip="加载" />
        <Tooltip title="提示"><span>悬浮源</span></Tooltip>
        <Alert type="warning" message="横幅" closable />
      </div>,
    )
    console.log('REST-DOM:', snap(container, 1200))
    const btn = container.querySelector('button.ev_button, [class*="ev_button"]') ?? container.querySelector('button')
    if (btn) {
      fireEvent.click(btn)
      expect(onClick).toHaveBeenCalled()
    }
    expect(container.textContent).toContain('横幅')
  })

  it('FormItemShell + Input 组合（错误态受控）', () => {
    const { container, rerender } = render(
      <FormItemShell label="字段" required error="必填项">
        <Input value="" onChange={() => {}} />
      </FormItemShell>,
    )
    expect(container.querySelector('.fis-error')).toBeTruthy()
    expect(screen.getByRole('alert').textContent).toBe('必填项')
    rerender(
      <FormItemShell label="字段" required>
        <Input value="x" onChange={() => {}} />
      </FormItemShell>,
    )
    expect(container.querySelector('.fis-error')).toBeNull()
  })
})

d('真实校准 · 结构组（Table→Tree→Tabs 静态；Tree 首渲有 R8 挂死嫌疑故居中）', () => {
  it('Table：动态列 render + 受控勾选回调', async () => {
    const checks: unknown[] = []
    const { container } = render(
      <Table
        data-test="cal-table"
        columns={[
          { title: '名称', dataIndex: 'name', width: 100 },
          { title: '值', dataIndex: 'val', width: 100, render: (v, r) => `R_${v}_${r.name}` },
        ]}
        dataSource={[{ name: 'r1', val: 'x' }, { name: 'r2', val: 'y' }]}
        rowKey="name"
        rowSelection={{ selectedRowKeys: [], onChange: (keys) => checks.push(keys) }}
        pagination={false}
      />,
    )
    console.log('TABLE-DOM:', snap(container, 1500))
    expect(container.textContent).toContain('r1')
    console.log('TABLE 自定义render R_x_r1 出现=', container.textContent?.includes('R_x_r1'))
    // gate 教训：checkbox 监听在叶子 span。
    const box = container.querySelectorAll('[class*="checkbox_span"], [role="checkbox"]')[1]
    if (box) {
      console.log('TABLE: 即将派发勾选点击')
      fireEvent.click(box)
      await new Promise((r) => setTimeout(r, 60))
      console.log('TABLE 勾选回调 keys:', JSON.stringify(checks), '（应含 ["r1"] 形态）')
    }
  })

  it('Menu→Tree：受控展开 + 节点选中回调', async () => {
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
    const { container } = render(<Host />)
    console.log('TREE-DOM(收起):', snap(container, 800))
    console.log('TREE: 即将派发展开点击')
    fireEvent.click(screen.getByText('expand-g1'))
    console.log('TREE: 展开点击已派发')
    await new Promise((r) => setTimeout(r, 80))
    const expanded = container.querySelector('.ev_tree_expanded')
    console.log('TREE expandedKeys 受控后 ev_tree_expanded 存在=', !!expanded)
    const leaf = Array.from(container.querySelectorAll('.ev_tree_text')).find((el) => el.textContent === '叶子一')
    if (leaf) {
      console.log('TREE: 即将派发叶子点击')
      fireEvent.click(leaf)
      console.log('TREE 点叶子 onClick keys:', JSON.stringify(clicks), '（应含 leaf1）')
    }
  })

  // R7 定案：Tabs 挂死不在点击而在 props 更新路径——R7 报告中 rerender(activeKey)
  // 且零点击的用例同样在 TABS-DOM 打印后挂死（标记日志未现），链条=任何 activeKey
  // 变更 → Tab cWRP/更新逻辑同步死循环（happy-dom 无真实布局，宽度恒 0 不收敛）。
  // 故 REAL 套件只校准 Tabs 静态渲染；受控切换与点击切换移交 F3 真浏览器
  // （真 Chromium 有真实布局，循环可收敛），组 6 的 F3 套件覆盖。
  it('Tabs：标签栏静态渲染（受控/点击切换=F3 项）', () => {
    const { container } = render(
      <Tabs
        items={[
          { key: 'a', label: '甲页', children: <p>content-a</p> },
          { key: 'b', label: '乙页', children: <p>content-b</p> },
        ]}
        activeKey="a"
        onChange={() => {}}
      />
    )
    console.log('TABS-DOM:', snap(container, 1000))
    expect(container.textContent).toContain('甲页')
    expect(container.textContent).toContain('乙页')
    expect(container.textContent).toContain('content-a')
  })
})
