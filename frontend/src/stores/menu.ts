// 清场期类型占位：Pinia 实现已随 Vue 栈退役，zustand 版 store（PR-4）在同路径
// 重建并保留本类型导出。类型先行驻留此处，使 utils/composables 的 type-only
// import 零改动沿用（D4 逐字节）。

// SND 左树节点（LT-03）：分组（children）、叶子（sourceModule；available/module
// 标注），或叶子下的模块级子节点（kind=container/rpc，与 YANG 模块顶层同级平铺，
// LT-02）——container 路由 /module/<name>、rpc 路由 /module/<叶module>/rpc/<name>。
export interface LeftTreeNode {
  zh: string
  en: string
  sourceModule?: string
  available?: boolean
  module?: string
  supported?: boolean
  kind?: 'container' | 'rpc'
  name?: string
  highRisk?: boolean
  children?: LeftTreeNode[]
}
