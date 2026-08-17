import { describe, it, expect, vi } from 'vitest'
import { useEffect, useState } from 'react'
import { render, screen, fireEvent, waitFor, cleanup } from './inula-testing'
import { afterEach } from 'vitest'

afterEach(cleanup)

// 闸门 1.2（红线项）：vitest + @testing-library/react + happy-dom 在
// openinula 运行时（react alias→openinula）下的基础能力探针——
// render / state 更新 / effect / 事件 / 卸载清理 / rerender 六件套。
// 本套件红 = 测试基建不兼容 = 闸门停。

function Counter({ onMount }: { onMount?: () => void }) {
  const [n, setN] = useState(0)
  useEffect(() => {
    onMount?.()
    return () => onMount?.()
  }, [onMount])
  return (
    <div>
      <span data-test="count">{`count:${n}`}</span>
      <button onClick={() => setN((v) => v + 1)}>inc</button>
    </div>
  )
}

describe('openinula 测试基建探针（闸门 1.2）', () => {
  it('render + 文本断言', () => {
    render(<Counter />)
    expect(screen.getByText('count:0')).toBeInTheDocument()
  })

  it('事件触发 state 更新并重渲', async () => {
    render(<Counter />)
    fireEvent.click(screen.getByRole('button', { name: 'inc' }))
    await waitFor(() => expect(screen.getByText('count:1')).toBeInTheDocument())
  })

  it('effect 挂载执行、卸载清理', () => {
    const onMount = vi.fn()
    const { unmount } = render(<Counter onMount={onMount} />)
    expect(onMount).toHaveBeenCalledTimes(1)
    unmount()
    expect(onMount).toHaveBeenCalledTimes(2)
  })

  it('rerender 以新 props 更新', () => {
    function Label({ text }: { text: string }) {
      return <span>{text}</span>
    }
    const { rerender } = render(<Label text="a" />)
    rerender(<Label text="b" />)
    expect(screen.getByText('b')).toBeInTheDocument()
  })

  it('data-test 属性落到真实 DOM（锚点前提）', () => {
    const { container } = render(<Counter />)
    expect(container.querySelector('[data-test="count"]')).toBeTruthy()
  })
})
