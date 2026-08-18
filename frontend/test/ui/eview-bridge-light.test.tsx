import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent, cleanup } from '@testing-library/react'

// 组 4.1 轻组件桥 F2（替身规格 = vendor d.ts + component-matrix + gate 实测）。
// 替身职责：暴露收到的 eview props 供断言（data-recv）+ 最小交互行为。
// vi.mock 工厂被提升到文件顶部——辅助必须经 vi.hoisted 且不得引用模块作用域
// 的 JSX 运行时，故工厂内动态 import react 用 createElement 构造替身。
const H = vi.hoisted(() => {
  const recv = { last: {} as Record<string, any> }
  const makeStub =
    (name: string, extra?: (p: any, h: any) => any) =>
    async () => {
      const { createElement: h } = await import('react')
      return {
        default: (p: any) => {
          recv.last[name] = p
          return h('div', { 'data-stub': name, id: p.id }, extra?.(p, h), p.children)
        },
      }
    }
  return { recv, makeStub }
})
const recv = H.recv

vi.mock('@nce/eview-react/Tag', H.makeStub('Tag'))
vi.mock('@nce/eview-react/Badge', H.makeStub('Badge'))
vi.mock('@nce/eview-react/Crumbs', H.makeStub('Crumbs'))
vi.mock('@nce/eview-react/Empty', H.makeStub('Empty'))
vi.mock('@nce/eview-react/Drawer', H.makeStub('Drawer'))
vi.mock('@nce/eview-react/Dialog', H.makeStub('Dialog', (p, h) =>
  (p.buttons ?? []).map((b: any, i: number) => h('button', { key: i, onClick: b.onClick }, b.text ?? `btn${i}`)),
))
vi.mock('@nce/eview-react/DropDown', H.makeStub('DropDown', (p, h) =>
  h('button', { onClick: () => p.onItemClick?.(p.data?.[1]) }, 'pick2'),
))
vi.mock('@nce/eview-react/Segmented', H.makeStub('Segmented', (p, h) =>
  h('button', { onClick: () => p.onChange?.(p.data?.[1]?.value) }, 'seg2'),
))
vi.mock('@nce/eview-react/RadioGroup', H.makeStub('RadioGroup', (p, h) =>
  h('div', null,
    h('button', { onClick: () => p.onChange?.(p.value, p.data?.[1]?.value) }, 'old-first'),
    h('button', { onClick: () => p.onChange?.(p.data?.[1]?.value, p.value) }, 'new-first'),
  ),
))
vi.mock('@nce/eview-react/Checkbox', H.makeStub('Checkbox', (p, h) =>
  h('button', { onClick: () => p.onChange?.('v', !p.checked) }, 'flip'),
))
vi.mock('@nce/eview-react/Switch', H.makeStub('Switch', (p, h) =>
  h('button', { onClick: () => p.onToggle?.(!p.toggled) }, 'toggle'),
))

import { Tag, Badge, Breadcrumb, Empty, Drawer, Modal, toPx } from '@bridge/components/display'
import { Dropdown, Segmented, Radio, Checkbox, Switch } from '@bridge/components/controls'
import { ANCHOR_SELECTOR } from '../../src/ui/bridge'

afterEach(() => {
  cleanup()
  recv.last = {}
})

describe('展示组桥', () => {
  it('Tag：error→danger、processing→primary、round 跟默认、锚点可命中', () => {
    render(<Tag color="error" data-test="row-mark">x</Tag>)
    expect(recv.last.Tag.color).toBe('danger')
    // R3 实测 round 语义与 d.ts 推断相反——桥跟默认走（不传），目视验收定。
    expect(recv.last.Tag.round).toBeUndefined()
    expect(document.querySelector(ANCHOR_SELECTOR('row-mark'))).toBeTruthy()
    render(<Tag color="processing">y</Tag>)
    expect(recv.last.Tag.color).toBe('primary')
  })

  it('Badge：count→content、small 固化样式', () => {
    render(<Badge count={5} size="small" />)
    expect(recv.last.Badge.content).toBe(5)
    expect(recv.last.Badge.badgeStyle).toBeTruthy()
    expect(recv.last.Badge.max).toBe(99)
  })

  it('Breadcrumb：items→data、separator→seprater 拼写映射', () => {
    render(<Breadcrumb items={[{ title: 'a' }, { title: 'b' }]} separator=">" />)
    expect(recv.last.Crumbs.data).toEqual([{ title: 'a' }, { title: 'b' }])
    expect(recv.last.Crumbs.seprator).toBe('>')
  })

  it('Drawer：open→visible、% 宽度折算 px、maskClosable→isClickMask', () => {
    render(<Drawer open width="50%" maskClosable={false} title="t" />)
    expect(recv.last.Drawer.visible).toBe(true)
    expect(typeof recv.last.Drawer.width).toBe('number')
    expect(recv.last.Drawer.isClickMask).toBe(false)
    expect(toPx('50%', 1000)).toBe(500)
    expect(toPx(320, 1000)).toBe(320)
  })

  it('Modal：open→isOpen、onOk→buttons、footer=null 无底栏、confirmLoading 吞 onOk', () => {
    const onOk = vi.fn()
    const { rerender } = render(<Modal open onOk={onOk} okText="确定" title="t" width={600} />)
    expect(recv.last.Dialog.isOpen).toBe(true)
    expect(recv.last.Dialog.size).toEqual([600, null])
    expect(recv.last.Dialog.movable).toBe(false)
    fireEvent.click(screen.getByText('确定'))
    expect(onOk).toHaveBeenCalledTimes(1)
    rerender(<Modal open onOk={onOk} okText="确定" title="t" confirmLoading />)
    fireEvent.click(screen.getByText('确定'))
    expect(onOk).toHaveBeenCalledTimes(1) // loading 期间吞掉
    rerender(<Modal open footer={null} title="t" />)
    expect(recv.last.Dialog.buttons).toBeUndefined()
  })
})

describe('交互组桥', () => {
  it('Dropdown：items→data(key↔value)、onItemClick 还原 antd onClick({key})', () => {
    const onClick = vi.fn()
    render(
      <Dropdown menu={{ items: [{ key: 'a', label: 'A' }, { key: 'b', label: 'B' }], onClick }} trigger={['click']}>
        <span>trig</span>
      </Dropdown>,
    )
    expect(recv.last.DropDown.data[1]).toMatchObject({ text: 'B', value: 'b' })
    fireEvent.click(screen.getByText('pick2'))
    expect(onClick).toHaveBeenCalledWith({ key: 'b' })
  })

  it('Segmented：options→data(label→text/disable 拼写)、onChange 透传', () => {
    const onChange = vi.fn()
    render(<Segmented options={[{ label: 'X', value: 'x' }, { label: 'Y', value: 'y', disabled: true }]} value="x" onChange={onChange} />)
    expect(recv.last.Segmented.data[1]).toMatchObject({ text: 'Y', value: 'y', disable: true })
    fireEvent.click(screen.getByText('seg2'))
    expect(onChange).toHaveBeenCalledWith('y')
  })

  it('Radio.Group：children 形态→data、isControlled、参数序自适应（两种顺序同判新值）', () => {
    const onChange = vi.fn()
    render(
      <Radio.Group value="a" onChange={onChange}>
        <Radio value="a">甲</Radio>
        <Radio value="b" disabled>乙</Radio>
      </Radio.Group>,
    )
    expect(recv.last.RadioGroup.isControlled).toBe(true)
    expect(recv.last.RadioGroup.data).toEqual([
      { value: 'a', text: '甲', disabled: undefined },
      { value: 'b', text: '乙', disabled: true },
    ])
    fireEvent.click(screen.getByText('old-first'))
    expect(onChange).toHaveBeenLastCalledWith({ target: { value: 'b' } })
    fireEvent.click(screen.getByText('new-first'))
    expect(onChange).toHaveBeenLastCalledWith({ target: { value: 'b' } })
  })

  it('Checkbox：eview (值,checked) → antd e.target.checked', () => {
    const onChange = vi.fn()
    render(<Checkbox checked={false} onChange={onChange} />)
    fireEvent.click(screen.getByText('flip'))
    expect(onChange).toHaveBeenCalledWith({ target: { checked: true } })
  })

  it('Switch：checked→toggled+isControlToggled+data 两态、onToggle→onChange(bool)', () => {
    const onChange = vi.fn()
    render(<Switch checked={false} onChange={onChange} />)
    expect(recv.last.Switch.isControlToggled).toBe(true)
    expect(recv.last.Switch.data).toEqual([false, true])
    fireEvent.click(screen.getByText('toggle'))
    expect(onChange).toHaveBeenCalledWith(true)
  })
})
