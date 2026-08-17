// openinula 的 React 18 API 垫片（change frontend-eviewui-inula-switch 翻转波）。
// openinula 提供 React 17 级 API；antd 6 及部分生态依赖 18 级四件：
// useSyncExternalStore / useId / useTransition / useDeferredValue。
// 本模块 = openinula 全量再导出 + 四个用户态垫片，构建期把 'react' 别名到此。
// 垫片语义（无 SSR、无并发调度的教科书退化实现）：
// - useSyncExternalStore：官方 use-sync-external-store/shim 同算法（订阅+快照比对）。
// - useId：模块级计数器（仅保证运行时唯一性，无 SSR 一致性需求）。
// - useTransition：同步执行（[isPending=false, run]）。
// - useDeferredValue：恒等透传。
export * from 'openinula'
import Inula, { useState, useEffect, useLayoutEffect, useRef } from 'openinula'
export default Inula

export function useSyncExternalStore<T>(
  subscribe: (onStoreChange: () => void) => () => void,
  getSnapshot: () => T,
  _getServerSnapshot?: () => T,
): T {
  const value = getSnapshot()
  const [{ inst }, forceUpdate] = useState({ inst: { value, getSnapshot } })
  useLayoutEffect(() => {
    inst.value = value
    inst.getSnapshot = getSnapshot
    if (checkIfChanged(inst)) forceUpdate({ inst })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [subscribe, value, getSnapshot])
  useEffect(() => {
    if (checkIfChanged(inst)) forceUpdate({ inst })
    return subscribe(() => {
      if (checkIfChanged(inst)) forceUpdate({ inst })
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [subscribe])
  return value
}

function checkIfChanged(inst: { value: unknown; getSnapshot: () => unknown }): boolean {
  try {
    return !Object.is(inst.value, inst.getSnapshot())
  } catch {
    return true
  }
}

let idCounter = 0
export function useId(): string {
  const ref = useRef<string | null>(null)
  if (ref.current === null) ref.current = `:inula-id-${++idCounter}:`
  return ref.current
}

export function useTransition(): [boolean, (cb: () => void) => void] {
  const run = useRef((cb: () => void) => cb()).current
  return [false, run]
}

export function useDeferredValue<T>(value: T): T {
  return value
}

// 18 级次常用面：antd/rc-* 偶用，退化到 layout effect / 空实现。
export const useInsertionEffect = useLayoutEffect
