import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

interface NativeModel {
  name: string
  title: string
  vendor: string
}

export const useMenuStore = defineStore('menu', () => {
  const nativeModels = ref<NativeModel[]>([])
  const nativeMenuLoaded = ref(false)
  const nativeMenuLoading = ref(false)
  const isCollapsed = ref(false)

  async function loadNativeModels() {
    if (nativeMenuLoaded.value) return

    nativeMenuLoading.value = true
    try {
      const res = await fetch('/api/v1/yang/modules')
      const data = await res.json()
      // Map response format to expected format
      nativeModels.value = (data.data || []).map((m: any) => ({
        name: m.name,
        title: m.description || m.name,
        vendor: 'huawei'
      }))
      nativeMenuLoaded.value = true
    } catch (e) {
      console.error('Failed to load native models:', e)
      // Fallback to example modules
      nativeModels.value = [
        { name: 'huawei-ifm', title: '华为接口管理', vendor: 'huawei' },
        { name: 'huawei-vlan', title: '华为 VLAN 配置', vendor: 'huawei' }
      ]
      nativeMenuLoaded.value = true
    } finally {
      nativeMenuLoading.value = false
    }
  }

  // Group modules by vendor
  const groupedByVendor = computed(() => {
    const groups = new Map<string, NativeModel[]>()
    for (const m of nativeModels.value) {
      const vendor = m.vendor || '其他'
      if (!groups.has(vendor)) {
        groups.set(vendor, [])
      }
      groups.get(vendor)!.push(m)
    }
    return groups
  })

  function toggleCollapse() {
    isCollapsed.value = !isCollapsed.value
  }

  return {
    nativeModels,
    nativeMenuLoaded,
    nativeMenuLoading,
    isCollapsed,
    loadNativeModels,
    groupedByVendor,
    toggleCollapse
  }
})
