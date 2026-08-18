// icon-plus 图标组件在 ref 回调里 attachShadow 渲染 SVG，且无「已挂载」防御；
// React 19 的生命周期重放会对同一宿主二次 attachShadow 抛 NotSupportedError
// 且走 React 不捕获的路径（E2E 实录）。icon-plus 为 closed 模式（宿主上查
// 不到 shadowRoot）——WeakMap 记账缓存 root，二次调用返回缓存实现幂等。
// 注意：生产 bundle 模块序下本模块须由入口第一行 import（install-guards）；
// index.html 另有内联兜底（双保险）。波 C 切 openinula 后复评。
const attachedRoots = new WeakMap<Element, ShadowRoot>()

export function installAttachShadowGuard(): void {
  if (typeof Element === 'undefined') return
  const proto = Element.prototype as { __ubShadowGuard?: boolean } & typeof Element.prototype
  if (proto.__ubShadowGuard) return
  proto.__ubShadowGuard = true
  const orig = proto.attachShadow
  proto.attachShadow = function (init: ShadowRootInit) {
    const prev = attachedRoots.get(this) ?? this.shadowRoot
    if (prev) return prev
    const root = orig.call(this, init)
    attachedRoots.set(this, root)
    return root
  }
}
