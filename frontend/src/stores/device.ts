import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'

export interface Device {
  id: string
  ip: string
  name: string
  vendor: string
  model: string
  status: 'online' | 'offline' | 'unknown'
  lastSync: string
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
      const res = await axios.get('/api/devices')
      devices.value = res.data.devices || []
    } catch (e) {
      console.error('Failed to fetch devices:', e)
      devices.value = []
    } finally {
      isLoading.value = false
    }
  }

  async function testConnection(deviceId: string): Promise<{ success: boolean; message: string }> {
    try {
      const res = await axios.post(`/api/devices/${deviceId}/test-connection`)
      return res.data
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
