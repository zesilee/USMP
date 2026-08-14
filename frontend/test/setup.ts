// 全局测试装配（React 栈）：@testing-library 断言扩展 + 用例后自动卸载。
// happy-dom 缺 matchMedia，antd 响应式组件（Modal/message）依赖之——最小 polyfill。
import '@testing-library/jest-dom/vitest'
import { cleanup, configure } from '@testing-library/react'
import { afterEach } from 'vitest'

// CI 慢跑道下 antd 渲染+effect 链可超 waitFor 默认 1s（#327 后置红实例：
// ModuleFormTab 回填断言本地绿 CI 超时）。全局放宽 async 断言上限而非逐用例补。
configure({ asyncUtilTimeout: 4000 })

if (typeof window !== 'undefined' && !window.matchMedia) {
  window.matchMedia = ((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: () => {},
    removeListener: () => {},
    addEventListener: () => {},
    removeEventListener: () => {},
    dispatchEvent: () => false,
  })) as unknown as typeof window.matchMedia
}

afterEach(() => {
  cleanup()
  // antd 弹层（Modal.confirm/message）挂在 body 下且不随 React 树卸载，
  // 不清理会让跨用例的按钮查询串台。
  document.body.innerHTML = ''
})
