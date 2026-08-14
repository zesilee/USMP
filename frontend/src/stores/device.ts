// 清场期类型占位：Pinia 实现已随 Vue 栈退役，zustand 版 store（PR-4）在同路径
// 重建并保留本类型导出。类型先行驻留此处，使 utils 的 type-only import 零改动
// 沿用（D4 逐字节）。

export interface Device {
  id: string
  ip: string
  name: string
  vendor: string
  model: string
  role: string
  status: 'online' | 'offline' | 'unknown'
  lastSync: string
}
