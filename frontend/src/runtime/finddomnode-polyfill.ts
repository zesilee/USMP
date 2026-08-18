// findDOMNode polyfill（内网校准 R3 实证需求）：React 19 移除了
// ReactDOM.findDOMNode，而 EviewUI Dialog/Drawer 编译产物内部调用
// `react_dom_1.default.findDOMNode`。本 polyfill 以 fiber 遍历实现同语义
// （类实例 → 首个宿主 fiber 的 stateNode），安装到 react-dom 导出对象
// （CJS interop 的 default 与命名空间同源）。窗口期方案：波 C 切 openinula
// 后其自带 findDOMNode（17 级 API），本 polyfill 随之退役。
import ReactDOMDefault from 'react-dom'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type AnyRec = Record<string, any>

function findFromFiber(inst: AnyRec | null | undefined): Element | Text | null {
  if (!inst) return null
  // 已是 DOM 节点
  if ((inst as AnyRec).nodeType === 1 || (inst as AnyRec).nodeType === 3) return inst as unknown as Element
  let fiber: AnyRec | null | undefined = (inst as AnyRec)._reactInternals ?? (inst as AnyRec)._reactInternalFiber
  if (!fiber) return null
  // 深度优先找宿主 fiber（tag 5=HostComponent / 6=HostText）。
  const stack: AnyRec[] = [fiber]
  while (stack.length) {
    const node = stack.pop()!
    if (node.tag === 5 || node.tag === 6) return node.stateNode ?? null
    if (node.sibling) stack.push(node.sibling)
    if (node.child) stack.push(node.child)
  }
  return null
}

/** 幂等安装（REAL 校准与组 5 接线后的窗口期运行时都需要）。 */
export function installFindDOMNodePolyfill(): void {
  // 静态 import 的 default（CJS interop = react-dom 的 exports 本体）为主路径
  // ——真浏览器 ESM 无 require（F3-R1 实录坑）；node/vitest 下再补 require
  // 视角兜底（typeof 守卫，浏览器不执行）。eview 编译产物读的是同一份
  // CJS exports 对象，两路径打到同一目标。
  const targets: AnyRec[] = [ReactDOMDefault as unknown as AnyRec]
  try {
    if (typeof require === 'function') {
      // eslint-disable-next-line @typescript-eslint/no-require-imports
      const rd = require('react-dom') as AnyRec
      targets.push(rd)
    }
  } catch {
    /* 浏览器无 require——静态路径已覆盖 */
  }
  for (const t of targets) {
    if (!t) continue
    if (typeof t.findDOMNode !== 'function') t.findDOMNode = findFromFiber
    if (t.default && typeof t.default.findDOMNode !== 'function') t.default.findDOMNode = findFromFiber
  }
}
