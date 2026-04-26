// VLAN 状态类型
export type AdminStatus = 'UP' | 'DOWN'
export type OperStatus = 'ACTIVE' | 'INACTIVE' | 'SUSPENDED'

// VLAN 列表项
export interface VlanItem {
  id: number
  name: string
  adminStatus: AdminStatus
  operStatus: OperStatus
  taggedPorts: string[]
  untaggedPorts: string[]
}

// VLAN 表单数据
export interface VlanFormData {
  id: number | null
  name: string
  adminStatus: AdminStatus
  taggedPorts: string[]
  untaggedPorts: string[]
}

// 端口信息
export interface PortInfo {
  name: string
  type: 'GE' | '10GE' | 'Eth' | 'XGE'
  status: 'UP' | 'DOWN'
  currentVlan?: number
}

// VLAN 配置响应
export interface VlanConfigResponse {
  vlans: VlanItem[]
  fromCache: boolean
  lastSync: string
}
