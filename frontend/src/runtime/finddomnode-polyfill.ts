// findDOMNode polyfill（内网校准 R3 实证需求）：React 19 移除了
// ReactDOM.findDOMNode，而 EviewUI Dialog/Drawer 编译产物内部调用
// `react_dom_1.default.findDOMNode`。本 polyfill 以 fiber 遍历实现同语义
// （类实例 → 首个宿主 fiber 的 stateNode），安装到 react-dom 导出对象
// （CJS interop 的 default 与命名空间同源）。窗口期方案：波 C 切 openinula
// 后其自带 findDOMNode（17 级 API），本 polyfill 随之退役。
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
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const rd = require('react-dom') as AnyRec
  if (typeof rd.findDOMNode !== 'function') rd.findDOMNode = findFromFiber
  if (rd.default && typeof rd.default.findDOMNode !== 'function') rd.default.findDOMNode = findFromFiber
}
