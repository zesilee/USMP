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

  // ===== 业务配置菜单（FE-13）：/yang/modules 驱动，指向通用模块控制台 =====
  const businessModules = ref<NativeModel[]>([])
  const businessLoaded = ref(false)

  async function loadBusinessModules() {
    if (businessLoaded.value) return
    try {
      const res = await fetch('/api/v1/yang/modules')
      const data = await res.json()
      const mods = (data.data || []).map((m: any) => ({
        name: m.name,
        title: m.description || m.title || m.name,
        vendor: m.vendor || '其他',
      }))
      if (!mods.length) throw new Error('empty modules')
      businessModules.value = mods
    } catch (e) {
      console.warn('加载 YANG 模块列表失败，回退内置菜单:', e)
      // 回退项（R08）：与后端注册的模块根名一致（GetSchema/{name} 可直接命中）。
      businessModules.value = [
        { name: 'ifm', title: '接口管理', vendor: 'huawei' },
        { name: 'vlan', title: 'VLAN 配置', vendor: 'huawei' },
      ]
    } finally {
      businessLoaded.value = true
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
    businessModules,
    businessLoaded,
    loadBusinessModules,
    groupedByVendor,
    toggleCollapse
  }
})
