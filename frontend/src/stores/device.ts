import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
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

export const useDeviceStore = defineStore('device', () => {
  const devices = ref<Device[]>([])
  // 全局设备上下文（FE-10）：设备作用域配置页共享的唯一选中态，IP 口径
  //（id 即 ip，与控制台下拉 value / 配置 API 设备标识一致）。
  const selectedDeviceIp = ref('')
  const isLoading = ref(false)

  const onlineCount = computed(() => devices.value.filter(d => d.status === 'online').length)
  const offlineCount = computed(() => devices.value.filter(d => d.status === 'offline').length)

  async function fetchDevices() {
    isLoading.value = true
    try {
      const res = await listDevices()
      // 真实契约信封(类型安全): { success, data: { devices: [...], stats } }
      // 旧二进制信封(兼容降级): { success, data: [...] }
      const env = res.data
      const raw = Array.isArray((env as any).data)
        ? (env as any).data // 旧后端扁平 data[]
        : (env.data?.devices ?? []) // res.data.data.devices —— 写成 res.data.devices 会编译报错
      devices.value = raw.map(normalizeDevice)
    } catch (e) {
      console.error('Failed to fetch devices:', e)
      devices.value = []
    } finally {
      isLoading.value = false
    }
  }

  async function testConnection(ip: string): Promise<{ success: boolean; message: string }> {
    try {
      // 后端无 test-connection 端点，用设备状态探活 GET /devices/:ip/status
      const res = await getDeviceStatus(ip)
      const connected = res.data.data?.connected === true
      return connected
        ? { success: true, message: i18n.global.t('devices.connOk') }
        : { success: false, message: i18n.global.t('devices.connNotConnected') }
    } catch (e) {
      return { success: false, message: i18n.global.t('devices.connTestFailed') }
    }
  }

  function selectDevice(ip: string) {
    selectedDeviceIp.value = ip
  }

  function clearSelection() {
    selectedDeviceIp.value = ''
  }

  return {
    devices,
    selectedDeviceIp,
    isLoading,
    onlineCount,
    offlineCount,
    fetchDevices,
    testConnection,
    selectDevice,
    clearSelection
  }
})
