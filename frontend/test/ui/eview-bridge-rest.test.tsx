import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent, cleanup } from '@testing-library/react'

// 组 4.4 收尾组桥 F2（替身规格 = vendor d.ts + matrix + gate 实测）。
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

vi.mock('@nce/eview-react/Button', H.makeStub('Button', (p, h) =>
  h('button', { onClick: (e: unknown) => p.onClick?.(e, 'additional'), disabled: p.disabled }, 'go'),
))
vi.mock('@nce/eview-react/Loading', H.makeStub('Loading'))
vi.mock('@nce/eview-react/TipBox', H.makeStub('TipBox', (p, h) =>
  h('button', { onClick: () => p.onClose?.() }, 'close-tip'),
))
vi.mock('@nce/eview-react/DivMessage', H.makeStub('DivMessage', (p, h) =>
  h('button', { onClick: () => p.onClose?.() }, 'close-msg'),
))
vi.mock('@nce/eview-react/PageMessage', H.makeStub('PageMessage'))

import { Button, Spin, Tooltip, Popover, Alert } from '../../src/ui/eview/components/rest'

afterEach(() => {
  cleanup()
  recv.last = {}
})

describe('Button 桥', () => {
  it('type/danger→status 映射（danger 优先 risk）；link→text', () => {
    render(<Button type="primary">a</Button>)
    expect(recv.last.Button.status).toBe('primary')
    render(<Button type="primary" danger>b</Button>)
    expect(recv.last.Button.status).toBe('risk')
    render(<Button type="link">c</Button>)
    expect(recv.last.Button.status).toBe('text')
  })
  it('loading：禁点+自绘 spinner；ghost→样式类；onClick 吞双参对齐单参', () => {
    const onClick = vi.fn()
    const { rerender } = render(<Button loading onClick={onClick} ghost>x</Button>)
    expect(recv.last.Button.disabled).toBe(true)
    expect(recv.last.Button.className).toContain('ub-btn-ghost')
    fireEvent.click(screen.getByText('go'))
    expect(onClick).not.toHaveBeenCalled()
    rerender(<Button onClick={onClick}>x</Button>)
    fireEvent.click(screen.getByText('go'))
    expect(onClick).toHaveBeenCalledTimes(1)
    expect(onClick.mock.calls[0].length).toBe(1) // 单参
  })
})

describe('Spin/Tooltip/Popover 桥', () => {
  it('Spin→Loading(type=local, isOpen)', () => {
    render(<Spin tip="w" />)
    expect(recv.last.Loading).toMatchObject({ isOpen: true, type: 'local', desc: 'w' })
  })
  it('Tooltip：antd title→TipBox content（hover）', () => {
    render(<Tooltip title="提示"><span>t</span></Tooltip>)
    expect(recv.last.TipBox).toMatchObject({ content: '提示', trigger: 'hover' })
  })
  it('Popover：click 触发、open→display 尽力受控、关闭通知 onOpenChange(false)', () => {
    const onOpenChange = vi.fn()
    render(<Popover content={<b>面板</b>} trigger="click" open onOpenChange={onOpenChange}><span>t</span></Popover>)
    expect(recv.last.TipBox.trigger).toBe('click')
    expect(recv.last.TipBox.display).toBe(true)
    fireEvent.click(screen.getByText('close-tip'))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })
})

describe('Alert 桥（DivMessage/PageMessage 分派）', () => {
  it('closable→DivMessage 且强制关自动消失（gate 陷阱）', () => {
    const onClose = vi.fn()
    render(<Alert type="warning" message="注意" closable onClose={onClose} />)
    expect(recv.last.DivMessage).toMatchObject({ type: 'warn', enableDisposeTimeOut: false, text: '注意' })
    fireEvent.click(screen.getByText('close-msg'))
    expect(onClose).toHaveBeenCalled()
  })
  it('info 型：DivMessage 无 info 映 default、PageMessage 保 info', () => {
    render(<Alert type="info" message="i" closable />)
    expect(recv.last.DivMessage.type).toBe('default')
    render(<Alert type="info" message="j" />)
    expect(recv.last.PageMessage.type).toBe('info')
  })
})
