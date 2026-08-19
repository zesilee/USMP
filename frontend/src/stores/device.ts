import { create } from './createStore'
import { listDevices, getDeviceStatus } from '../api'
import { i18n } from '../i18n'

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

// 后端 DeviceStatus 字段 → 前端 Device 归一化。兼容两种后端返回：
//   新契约 (#47+): { ip, port, online: bool, ... }
//   旧二进制:       { ip, port, status: 'online', ... }
// 缺失的 name/vendor/model/lastSync 后端未提供，用 ip 兜底避免空行（R08 降级）。
function normalizeDevice(d: any): Device {
  const online = typeof d.online === 'boolean' ? d.online : d.status === 'online'
  return {
    id: d.ip,
    ip: d.ip,
    name: d.name || d.ip,
    vendor: d.vendor || '',
    model: d.model || '',
    role: d.role || '',
    status: online ? 'online' : 'offline',
    lastSync: d.lastSync || '',
  }
}

interface DeviceState {
  devices: Device[]
  /** 全局设备上下文（FE-10）：设备作用域配置页共享的唯一选中态，IP 口径。 */
  selectedDeviceIp: string
  isLoading: boolean
  /** 在线/离线计数（旧 Pinia computed → 方法形态，zustand 惯用）。 */
  onlineCount: () => number
  offlineCount: () => number
  fetchDevices: () => Promise<void>
  testConnection: (ip: string) => Promise<{ success: boolean; message: string }>
  selectDevice: (ip: string) => void
  clearSelection: () => void
}

export const useDeviceStore = create<DeviceState>((set, get) => ({
  devices: [],
  selectedDeviceIp: '',
  isLoading: false,

  onlineCount: () => get().devices.filter((d) => d.status === 'online').length,
  offlineCount: () => get().devices.filter((d) => d.status === 'offline').length,

  fetchDevices: async () => {
    set({ isLoading: true })
    try {
      const res = await listDevices()
      // 真实契约信封(类型安全): { success, data: { devices: [...], stats } }
      // 旧二进制信封(兼容降级): { success, data: [...] }
      const env = res.data
      const raw = Array.isArray((env as any).data)
        ? (env as any).data // 旧后端扁平 data[]
        : (env.data?.devices ?? []) // res.data.data.devices —— 写成 res.data.devices 会编译报错
      set({ devices: raw.map(normalizeDevice) })
    } catch (e) {
      console.error('Failed to fetch devices:', e)
      set({ devices: [] })
    } finally {
      set({ isLoading: false })
    }
  },

  testConnection: async (ip) => {
    try {
      // 后端无 test-connection 端点，用设备状态探活 GET /devices/:ip/status
      const res = await getDeviceStatus(ip)
      const connected = res.data.data?.connected === true
      return connected
        ? { success: true, message: i18n.global.t('devices.connOk') }
        : { success: false, message: i18n.global.t('devices.connNotConnected') }
    } catch {
      return { success: false, message: i18n.global.t('devices.connTestFailed') }
    }
  },

  selectDevice: (ip) => set({ selectedDeviceIp: ip }),
  clearSelection: () => set({ selectedDeviceIp: '' }),
}))
