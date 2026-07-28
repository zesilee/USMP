// leafref 目标解析（FE-19：rpc 输入 leafref → 目标值下拉）。
//
// YANG leafref path 如 /ifm:ifm/ifm:interfaces/ifm:interface/ifm:name 指向某个 list
// 的 key 叶。要把它渲染成下拉，需要：① 取哪个配置路径拉数据（list 的父容器）、
// ② list 在 payload 里的键、③ key 叶名。此模块只做纯路径解析（可单测）；实际拉取
// 由调用方经 getConfig + extractRows 完成。

export interface LeafrefTarget {
  /** 拉配置的路径（保留模块前缀，如 /ifm:ifm/ifm:interfaces）。 */
  fetchPath: string
  /** payload 里承载 list 的键（list 节点局部名，如 interface）。 */
  listKey: string
  /** key 叶局部名（如 name），下拉的 label/value 取此叶值。 */
  keyField: string
}

// 段局部名：去掉模块前缀（ifm:name → name）。
function local(seg: string): string {
  const i = seg.indexOf(':')
  return i >= 0 ? seg.slice(i + 1) : seg
}

// parseLeafref 把 leafref path 拆成拉取路径/list 键/key 叶。
// 末段=key 叶、次末段=list 节点、其余=list 的父容器路径（拉取点）。
// 段数不足或空路径 → null（调用方降级为文本输入）。
export function parseLeafref(leafRef: string | undefined): LeafrefTarget | null {
  if (!leafRef) return null
  const raw = leafRef.split('/').filter(Boolean)
  if (raw.length < 2) return null
  const keyField = local(raw[raw.length - 1])
  const listKey = local(raw[raw.length - 2])
  const fetchPath = '/' + raw.slice(0, -2).join('/')
  if (!keyField || !listKey || fetchPath === '/') return null
  return { fetchPath, listKey, keyField }
}
