// 接口类型枚举
export type InterfaceType = 'PHYSICAL' | 'LOGICAL' | 'LAG' | 'LOOPBACK' | 'VLAN'

// 管理状态
export type AdminStatus = 'UP' | 'DOWN' | 'TESTING'

// 运行状态
export type OperStatus = 'UP' | 'DOWN' | 'TESTING' | 'UNKNOWN' | 'DORMANT' | 'NOT_PRESENT' | 'LOWER_LAYER_DOWN'

// 接口配置数据
export interface InterfaceConfig {
  /** 接口名称 */
  name: string
  /** 接口类型 */
  type: InterfaceType
  /** MTU (最大传输单元) */
  mtu: number
  /** 是否启用 */
  enabled: boolean
  /** 接口描述 */
  description?: string
}

// 接口运行状态数据
export interface InterfaceState {
  /** 接口索引 */
  ifindex?: number
  /** 管理状态 */
  'admin-status'?: AdminStatus
  /** 运行状态 */
  'oper-status'?: OperStatus
}

// 接口列表项
export interface InterfaceItem {
  /** 接口名称 */
  name: string
  /** 配置数据 */
  config?: InterfaceConfig
  /** 运行状态数据 */
  state?: InterfaceState
}

// 接口配置响应
export interface InterfaceConfigResponse {
  /** 接口列表 */
  interface: Record<string, InterfaceItem>
  /** 是否来自缓存 */
  fromCache?: boolean
  /** 最后同步时间 */
  lastSync?: string
}

// 接口表单数据
export interface InterfaceFormData {
  name: string
  type: InterfaceType
  mtu: number
  enabled: boolean
  description: string
}
