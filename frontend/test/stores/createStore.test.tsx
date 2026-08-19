import { describe, it, expect, vi } from 'vitest'
import { render, act } from '@testing-library/react'
import { create } from '../../src/stores/createStore'

// 状态层内核 F1（波 C 2.3，TDD 先行）：zustand 同形语义——selector bail
// （Object.is 不变不重渲）、函数式 setState、get 闭包、订阅/退订、
// 组件外 getState/setState、多组件独立订阅。
interface S {
  n: number
  s: string
  inc: () => void
}

function makeStore() {
  return create<S>((set, get) => ({
    n: 0,
    s: 'a',
    inc: () => set({ n: get().n + 1 }),
  }))
}

describe('createStore（React 17 级状态内核）', () => {
  it('组件订阅：变更重渲、selector 不变不重渲（bail）', () => {
    const useStore = makeStore()
    let renders = 0
    function Probe() {
      renders++
      const n = useStore((st) => st.n)
      return <i data-n={n} />
    }
    const { container } = render(<Probe />)
    expect(container.querySelector('i')?.getAttribute('data-n')).toBe('0')
    const before = renders
    act(() => useStore.getState().inc())
    expect(container.querySelector('i')?.getAttribute('data-n')).toBe('1')
    expect(renders).toBeGreaterThan(before)
    const mid = renders
    act(() => useStore.setState({ s: 'b' })) // 无关键变更——selector bail
    expect(renders).toBe(mid)
  })

  it('函数式 setState 与 getState（组件外）', () => {
    const useStore = makeStore()
    useStore.setState((st) => ({ n: st.n + 10 }))
    expect(useStore.getState().n).toBe(10)
  })

  it('subscribe/退订', () => {
    const useStore = makeStore()
    const fn = vi.fn()
    const off = useStore.subscribe(fn)
    useStore.setState({ n: 1 })
    expect(fn).toHaveBeenCalledTimes(1)
    off()
    useStore.setState({ n: 2 })
    expect(fn).toHaveBeenCalledTimes(1)
  })

  it('卸载后变更不再触发该组件（无泄漏告警）', () => {
    const useStore = makeStore()
    function Probe() {
      const n = useStore((st) => st.n)
      return <i data-n={n} />
    }
    const { unmount } = render(<Probe />)
    unmount()
    expect(() => act(() => useStore.getState().inc())).not.toThrow()
  })

  it('无 selector 时返回整体 state（引用每次变更后更新）', () => {
    const useStore = makeStore()
    function Probe() {
      const st = useStore()
      return <i data-n={st.n} />
    }
    const { container } = render(<Probe />)
    act(() => useStore.getState().inc())
    expect(container.querySelector('i')?.getAttribute('data-n')).toBe('1')
  })
})
