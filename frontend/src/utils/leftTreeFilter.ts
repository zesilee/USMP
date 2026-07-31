import type { LeftTreeNode } from '../stores/menu'

// 左树搜索过滤（LT-05）：zh/en/name 三口径大小写不敏感子串匹配。
// 命中节点保留整棵子树（可继续下钻）；未命中节点若有命中后代则保留祖先壳
// （children 收窄为命中分支）；均未命中则剪除。空查询原样返回（零开销）。
// 纯函数：不修改输入树。

function matches(node: LeftTreeNode, q: string): boolean {
  return [node.zh, node.en, node.name].some((s) => !!s && s.toLowerCase().includes(q))
}

function filterNodes(nodes: LeftTreeNode[], q: string): LeftTreeNode[] {
  const out: LeftTreeNode[] = []
  for (const node of nodes) {
    if (matches(node, q)) {
      out.push(node)
      continue
    }
    const kids = node.children?.length ? filterNodes(node.children, q) : []
    if (kids.length) out.push({ ...node, children: kids })
  }
  return out
}

export function filterLeftTree(nodes: LeftTreeNode[], query: string): LeftTreeNode[] {
  const q = query.trim().toLowerCase()
  if (!q) return nodes
  return filterNodes(nodes, q)
}
