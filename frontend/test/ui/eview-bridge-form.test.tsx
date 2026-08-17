import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, fireEvent, cleanup } from '@testing-library/react'
import { useState } from 'react'

// 组 4.2 表单输入桥 F2 + useSemiControlledBridge F1。
// 替身模拟 gate 实证的半受控行为：value 入内部 state、cWRP 同步、
// 输入先自改再回调——重挂（key 变化）时以 props.value 重新初始化。
const H = vi.hoisted(() => {
  const recv = { last: {} as Record<string, any> }
  const makeSemiStub =
    (name: string, emitBtn: (p: any, h: any, setV: (v: any) => void) => any) =>
    async () => {
      const React = await import('react')
      const h = React.createElement
      return {
        default: (p: any) => {
          recv.last[name] = p
          // 半受控内核：内部 state 初始=props.value；cWRP 同步（props 变即跟）。
          const [v, setV] = React.useState(p.value)
          const [prev, setPrev] = React.useState(p.value)
          if (!Object.is(p.value, prev)) {
            setPrev(p.value)
            setV(p.value)
          }
          return h('div', { 'data-stub': name, id: p.id }, h('output', null, String(v ?? '')), emitBtn(p, h, setV), p.suffix)
        },
      }
    }
  return { recv, makeSemiStub }
})
const recv = H.recv

vi.mock('@nce/eview-react/TextField', H.makeSemiStub('TextField', (p, h, setV) =>
  h('button', { onClick: () => { setV('AB'); p.onChange?.('AB', p.value) } }, 'type-AB'),
))
vi.mock('@nce/eview-react/Spinner', H.makeSemiStub('Spinner', (p, h, setV) =>
  h('div', null,
    h('button', { onClick: () => { setV(7); p.onChange?.(7) } }, 'num-7'),
    h('button', { onClick: () => p.onInputError?.('bad') }, 'num-bad'),
  ),
))
vi.mock('@nce/eview-react/InputSelect', H.makeSemiStub('InputSelect', (p, h, setV) =>
  h('div', null,
    h('button', { onClick: () => { setV('b'); p.onChange?.('b', p.value, 'select') } }, 'sel-b'),
    h('button', { onClick: () => p.onClear?.() }, 'sel-clear'),
  ),
))

import { Input, InputNumber, Select } from '../../src/ui/eview/components/inputs'
import { useSemiControlledBridge } from '../../src/ui/bridge'

afterEach(() => {
  cleanup()
  recv.last = {}
})

describe('useSemiControlledBridge（②回写+③重挂组合）', () => {
  function Probe({ accept }: { accept: boolean }) {
    const [v, setV] = useState('A')
    const { key, onEmit } = useSemiControlledBridge(v)
    return (
      <div>
        <span>{`key:${key} v:${v}`}</span>
        <button onClick={() => { onEmit('B'); if (accept) setV('B') }}>emit</button>
        <button onClick={() => setV('Z')}>parent-set</button>
      </div>
    )
  }
  it('父接受：key 不变（②档回写路径）', () => {
    render(<Probe accept />)
    fireEvent.click(screen.getByText('emit'))
    expect(screen.getByText('key:0 v:B')).toBeInTheDocument()
  })
  it('父拒写：key 递增强制重建（③档）', () => {
    render(<Probe accept={false} />)
    fireEvent.click(screen.getByText('emit'))
    expect(screen.getByText('key:1 v:A')).toBeInTheDocument()
  })
  it('父改写第三值：cWRP 路径不重挂', () => {
    render(<Probe accept={false} />)
    fireEvent.click(screen.getByText('parent-set'))
    expect(screen.getByText('key:0 v:Z')).toBeInTheDocument()
  })
})

describe('Input 桥', () => {
  it('onChange 合成 antd e.target.value；受控回显（父接受）', () => {
    function Host() {
      const [v, setV] = useState('A')
      return <Input value={v} onChange={(e) => setV(e.target.value)} data-test="fi" />
    }
    render(<Host />)
    fireEvent.click(screen.getByText('type-AB'))
    expect(screen.getByRole('status')).toHaveTextContent('AB') // output 元素
    expect(document.querySelector('#dt-fi')).toBeTruthy()
  })

  it('父拒写：重挂后回显 props 值（不停留用户输入）', () => {
    function Host() {
      const [v] = useState('A') // 拒写：onChange 不 setState
      return <Input value={v} onChange={() => {}} />
    }
    render(<Host />)
    fireEvent.click(screen.getByText('type-AB'))
    expect(screen.getByRole('status')).toHaveTextContent('A')
  })

  it('allowClear 有值出清除钮、点击上抛空串；password type 透传', () => {
    const onChange = vi.fn()
    render(<Input value="x" onChange={onChange} allowClear type="password" />)
    expect(recv.last.TextField.type).toBe('password')
    fireEvent.click(screen.getByLabelText('clear'))
    expect(onChange).toHaveBeenCalledWith({ target: { value: '' } })
  })

  it('validator 体系绝不下传（FA-06 守护）', () => {
    render(<Input value="x" onChange={() => {}} />)
    expect(recv.last.TextField.validator).toBeUndefined()
    expect(recv.last.TextField.required).toBeUndefined()
    expect(recv.last.TextField.rules).toBeUndefined()
  })
})

describe('InputNumber 桥', () => {
  it('min/max 缺省显式传无界（eview 默认 0/100 陷阱）', () => {
    render(<InputNumber value={1} onChange={() => {}} />)
    expect(recv.last.Spinner.min).toBe(Number.MIN_SAFE_INTEGER)
    expect(recv.last.Spinner.max).toBe(Number.MAX_SAFE_INTEGER)
  })
  it('有效值上抛数值、无效输入不上抛', () => {
    const onChange = vi.fn()
    render(<InputNumber value={1} onChange={onChange} min={0} max={10} />)
    expect(recv.last.Spinner.min).toBe(0)
    fireEvent.click(screen.getByText('num-7'))
    expect(onChange).toHaveBeenCalledWith(7)
    fireEvent.click(screen.getByText('num-bad'))
    expect(onChange).toHaveBeenCalledTimes(1)
  })
})

describe('Select 桥', () => {
  it('options label→text；onChange 取新值（gate 参数序）；showSearch=false→onlySelect', () => {
    const onChange = vi.fn()
    render(<Select options={[{ label: 'A', value: 'a' }, { label: 'B', value: 'b' }]} value="a" onChange={onChange} />)
    expect(recv.last.InputSelect.options[1]).toEqual({ text: 'B', value: 'b' })
    expect(recv.last.InputSelect.onlySelect).toBe(true)
    fireEvent.click(screen.getByText('sel-b'))
    expect(onChange).toHaveBeenCalledWith('b')
  })
  it('清空：onClear+onChange(undefined) 键不入 payload 语义', () => {
    const onChange = vi.fn()
    const onClear = vi.fn()
    render(<Select options={[]} value="a" onChange={onChange} allowClear onClear={onClear} showSearch />)
    expect(recv.last.InputSelect.onlySelect).toBe(false)
    fireEvent.click(screen.getByText('sel-clear'))
    expect(onClear).toHaveBeenCalled()
    expect(onChange).toHaveBeenCalledWith(undefined)
  })
})
