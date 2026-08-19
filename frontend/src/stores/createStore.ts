// 状态层内核（波 C 拍板 2026-08-19：自研 React 17 级薄层替换 zustand——
// zustand v5 内部依赖 useSyncExternalStore（React 18+ API），openinula（17 级）
// 无法运行；inula-X 则反向锁死 openinula 使外网 React 测试体系失效。本薄层
// 只用 useState/useEffect/useRef，React 与 openinula 双运行时可跑——外网
// 600+ 测试体系保留、内网/交付照常。API 与 zustand create 同形（调用点仅换
// import）：create(init) → useStore(selector?) + getState/setState/subscribe。
// selector 语义对齐 zustand v5：Object.is 比较，不变不重渲。
import { useEffect, useRef, useState } from 'react'

type SetState<T> = (partial: Partial<T> | ((s: T) => Partial<T>)) => void
type GetState<T> = () => T

export interface UseStore<T> {
  (): T
  <U>(selector: (s: T) => U): U
  getState: GetState<T>
  setState: SetState<T>
  subscribe: (listener: () => void) => () => void
}

export function create<T extends object>(init: (set: SetState<T>, get: GetState<T>) => T): UseStore<T> {
  let state: T
  const listeners = new Set<() => void>()
  const get: GetState<T> = () => state
  const set: SetState<T> = (partial) => {
    const next = typeof partial === 'function' ? (partial as (s: T) => Partial<T>)(state) : partial
    state = { ...state, ...next }
    listeners.forEach((l) => l())
  }
  state = init(set, get)

  function useStore<U>(selector?: (s: T) => U): U {
    const sel = selector ?? ((s: T) => s as unknown as U)
    const [, force] = useState(0)
    const selRef = useRef(sel)
    selRef.current = sel
    const valRef = useRef<U>(sel(state))
    valRef.current = sel(state)
    useEffect(() => {
      const check = () => {
        const next = selRef.current(state)
        if (!Object.is(next, valRef.current)) {
          valRef.current = next
          force((n) => n + 1)
        }
      }
      // 订阅生效前的空窗补偿（render→effect 间的变更）。
      check()
      listeners.add(check)
      return () => {
        listeners.delete(check)
      }
    }, [])
    return valRef.current
  }
  const store = useStore as UseStore<T>
  store.getState = get
  store.setState = set
  store.subscribe = (listener: () => void) => {
    listeners.add(listener)
    return () => {
      listeners.delete(listener)
    }
  }
  return store
}
