import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { listDevices, getDeviceStatus } from '../api'

export interface Device {
  id: string
  ip: string
  name: string
  vendor: string
  model: string
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
    status: online ? 'online' : 'offline',
    lastSync: d.lastSync || '',
  }
}

export const useDeviceStore = defineStore('device', () => {
  const devices = ref<Device[]>([])
  const selectedDevice = ref<Device | null>(null)
  const isLoading = ref(false)

  const onlineCount = computed(() => devices.value.filter(d => d.status === 'online').length)
  const offlineCount = computed(() => devices.value.filter(d => d.status === 'offline').length)

  async function fetchDevices() {
    isLoading.value = true
    try {
      const res = await listDevices()
      // 真实后端信封: { success, data: { devices: [...], stats } }
      // 旧二进制信封: { success, data: [...] } —— 两者都兼容。
      const payload: any = res.data?.data
      const raw = Array.isArray(payload) ? payload : (payload?.devices ?? [])
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
      const connected = (res.data?.data as any)?.connected === true
      return connected
        ? { success: true, message: '连接正常' }
        : { success: false, message: '设备未连接' }
    } catch (e) {
      return { success: false, message: '连接测试失败' }
    }
  }

  function selectDevice(device: Device) {
    selectedDevice.value = device
  }

  function clearSelection() {
    selectedDevice.value = null
  }

  return {
    devices,
    selectedDevice,
    isLoading,
    onlineCount,
    offlineCount,
    fetchDevices,
    testConnection,
    selectDevice,
    clearSelection
  }
})
