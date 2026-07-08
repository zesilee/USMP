import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

interface NativeModel {
  name: string
  title: string
  vendor: string
  // 任务域（后端 /yang/modules category，源自模块级 task-name 扩展，FE-13）。
  category?: string
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
        category: m.category,
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

  // 业务模块按任务域聚合（FE-13）：category 首现序，未标注归默认组('')排最后；
  // 全部未标注 → 单一默认组，菜单退化为平铺（R08 渲染不失败）。
  const businessGroups = computed(() => {
    const order: string[] = []
    const byCat = new Map<string, NativeModel[]>()
    for (const m of businessModules.value) {
      const c = m.category || ''
      if (!byCat.has(c)) {
        byCat.set(c, [])
        if (c) order.push(c)
      }
      byCat.get(c)!.push(m)
    }
    const out = order.map((c) => ({ category: c, modules: byCat.get(c)! }))
    if (byCat.has('')) out.push({ category: '', modules: byCat.get('')! })
    return out
  })

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
    businessGroups,
    businessLoaded,
    loadBusinessModules,
    groupedByVendor,
    toggleCollapse
  }
})
