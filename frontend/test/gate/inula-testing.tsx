// 闸门 1.2 方案 B：@testing-library/react 的 CJS 子模块（pure/act-compat）在
// vitest inline 转换外走了 Node 原生 require，拿到真 react-dom 与 openinula
// 元素互不相认——不与打包器搏斗，改为自写 render 薄层包 openinula 的
// createRoot+act，查询/事件/等待直接用 @testing-library/dom（纯 DOM 无 react
// 依赖）。API 与 @testing-library/react 同形（render/cleanup/screen/fireEvent/
// waitFor），全面迁移时 alias '@testing-library/react' → 本模块可让存量测试
// 零改动（闸门验证目标）。
import { createRoot, act } from 'react'
import type { ReactElement } from 'react'

export { screen, fireEvent, waitFor, within, getByText } from '@testing-library/dom'

type Root = { render: (el: ReactElement) => void; unmount: () => void }

const mounted: Array<{ root: Root; container: HTMLElement }> = []

export function render(el: ReactElement) {
  const container = document.createElement('div')
  document.body.appendChild(container)
  const root = (createRoot as (c: HTMLElement) => Root)(container)
  act(() => root.render(el))
  mounted.push({ root, container })
  return {
    container,
    rerender(next: ReactElement) {
      act(() => root.render(next))
    },
    unmount() {
      act(() => root.unmount())
      container.remove()
    },
  }
}

export function cleanup() {
  for (const { root, container } of mounted.splice(0)) {
    try {
      act(() => root.unmount())
    } catch {
      /* 已卸载则忽略 */
    }
    container.remove()
  }
}
