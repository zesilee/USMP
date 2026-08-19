// 路由 compat · inula-router(v5) 实现（波 C 翻转日，RT-03）。
// 仅在 USMP_RUNTIME=inula 构建时经 @app-router 别名启用（vite.config）；
// 外网测试与 antd 口径构建走 compat.ts（react-router 直通）。对外 API 与
// compat.ts 完全同形——调用点零改动：
// - createBrowserRouter/RouterProvider → BrowserRouter + 递归 Switch/Route
// - Outlet → Context 注入子路由树（v5 无嵌套渲染点）
// - useNavigate/useSearchParams → useHistory/useLocation 薄包装
// - useBlocker → Prompt + getUserConfirmation 桥（state/proceed/reset 契约
//   保持——FE-23 离开守卫 Scenario：弹确认、取消留原页、确认放行且变更集保留）
import * as IR from 'inula-router'
import type { ComponentType } from 'react'

// 类型断言桥：inula-router 的类型基于 openinula 节点类型，而 typecheck 恒跑
// React 类型面（外网无 inula 运行时）——组件/hooks cast 为 React 形态；运行时
// 行为由内网 E2E 验收（本文件仅 USMP_RUNTIME=inula 构建启用）。
/* eslint-disable @typescript-eslint/no-explicit-any */
const BrowserRouter = IR.BrowserRouter as unknown as ComponentType<{
  getUserConfirmation?: (message: string, cb: (ok: boolean) => void) => void
  children?: ReactNode
}>
const Switch = IR.Switch as unknown as ComponentType<{ children?: ReactNode }>
const Route = IR.Route as unknown as ComponentType<{
  path?: string
  exact?: boolean
  render?: () => ReactNode
}>
const Prompt = IR.Prompt as unknown as ComponentType<{
  when?: boolean
  message?: (location: unknown) => string | boolean
}>
const useHistory = IR.useHistory as unknown as () => {
  push: (to: unknown) => void
  replace: (to: unknown) => void
  go: (n: number) => void
}
const useLocation5 = IR.useLocation as unknown as () => { pathname: string; search: string; hash: string }
const useParams5 = IR.useParams as unknown as () => Record<string, string | undefined>
/* eslint-enable @typescript-eslint/no-explicit-any */
import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactElement,
  type ReactNode,
} from 'react'

export interface RouteDef {
  path?: string
  index?: boolean
  element?: ReactNode
  children?: RouteDef[]
}

export function createBrowserRouter(routes: RouteDef[]): { routes: RouteDef[] } {
  return { routes }
}

const OutletCtx = createContext<ReactNode>(null)

export function Outlet(): ReactElement {
  return <>{useContext(OutletCtx)}</>
}

function joinPath(base: string, seg?: string): string {
  if (!seg) return base || '/'
  return `${base.replace(/\/$/, '')}/${seg.replace(/^\//, '')}` || '/'
}

function renderRoutes(routes: RouteDef[], base = ''): ReactElement {
  return (
    <Switch>
      {routes.map((r, i) => {
        const full = r.index ? base || '/' : joinPath(base || '/', r.path)
        const hasKids = !!r.children?.length
        return (
          <Route
            key={r.path ?? `idx-${i}`}
            path={full}
            exact={!hasKids}
            render={() =>
              hasKids ? (
                <OutletCtx.Provider value={renderRoutes(r.children!, full)}>{r.element}</OutletCtx.Provider>
              ) : (
                <>{r.element}</>
              )
            }
          />
        )
      })}
    </Switch>
  )
}

// ===== useBlocker（Prompt + getUserConfirmation 桥）=====
type BlockerState = 'unblocked' | 'blocked'
const BLOCK_MSG = '__usmp_route_blocked__'
const blockers = new Set<{ shouldBlock: () => boolean }>()
const blockerListeners = new Set<() => void>()
let pendingCb: ((ok: boolean) => void) | null = null
let blockedFlag = false

function notifyBlockers(): void {
  blockerListeners.forEach((l) => l())
}

function confirmNavigation(message: string, cb: (ok: boolean) => void): void {
  if (message !== BLOCK_MSG) {
    cb(true)
    return
  }
  pendingCb = cb
  blockedFlag = true
  notifyBlockers()
}

export function useBlocker(shouldBlock: () => boolean): {
  state: BlockerState
  proceed: () => void
  reset: () => void
} {
  const entryRef = useRef({ shouldBlock })
  entryRef.current.shouldBlock = shouldBlock
  const [, force] = useState(0)
  useEffect(() => {
    const entry = entryRef.current
    blockers.add(entry)
    const l = () => force((n) => n + 1)
    blockerListeners.add(l)
    return () => {
      blockers.delete(entry)
      blockerListeners.delete(l)
    }
  }, [])
  const state: BlockerState = blockedFlag ? 'blocked' : 'unblocked'
  return useMemo(
    () => ({
      state,
      proceed() {
        const cb = pendingCb
        pendingCb = null
        blockedFlag = false
        notifyBlockers()
        cb?.(true)
      },
      reset() {
        const cb = pendingCb
        pendingCb = null
        blockedFlag = false
        notifyBlockers()
        cb?.(false)
      },
    }),
    [state],
  )
}

function GlobalPrompt(): ReactElement {
  // when 恒开；message 函数动态判定：任一 blocker 判阻→返回 BLOCK_MSG 触发
  // getUserConfirmation（异步确认）；否则 true 放行。
  return <Prompt when message={() => (Array.from(blockers).some((b) => b.shouldBlock()) ? BLOCK_MSG : true)} />
}

export function RouterProvider({ router }: { router: { routes: RouteDef[] } }): ReactElement {
  return (
    <BrowserRouter getUserConfirmation={confirmNavigation}>
      <GlobalPrompt />
      {renderRoutes(router.routes)}
    </BrowserRouter>
  )
}

// ===== hooks 薄包装 =====
export function useNavigate(): (to: string | number, opts?: { replace?: boolean }) => void {
  const history = useHistory()
  return (to, opts) => {
    if (typeof to === 'number') history.go(to)
    else if (opts?.replace) history.replace(to)
    else history.push(to)
  }
}

export function useLocation(): { pathname: string; search: string; hash: string } {
  return useLocation5()
}

export function useParams<T extends Record<string, string | undefined>>(): T {
  return useParams5() as T
}

export function useSearchParams(): [URLSearchParams, (p: URLSearchParams | Record<string, string>) => void] {
  const history = useHistory()
  const loc = useLocation5()
  const sp = useMemo(() => new URLSearchParams(loc.search), [loc.search])
  const set = (p: URLSearchParams | Record<string, string>) => {
    const s = p instanceof URLSearchParams ? p : new URLSearchParams(p)
    history.replace({ ...loc, search: s.toString() })
  }
  return [sp, set]
}
