import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { listYangModules, getLeftTree } from '../api'
import { i18n } from '../i18n'

// SND 左树节点（LT-03）：分组（children）或叶子（sourceModule；available/module 标注）。
export interface LeftTreeNode {
  zh: string
  en: string
  sourceModule?: string
  available?: boolean
  module?: string
  supported?: boolean
  children?: LeftTreeNode[]
}

interface NativeModel {
  name: string
  title: string
  vendor: string
  // 任务域（后端 /yang/modules category，源自模块级 task-name 扩展，FE-13）。
  category?: string
}

export const useMenuStore = defineStore('menu', () => {
  const isCollapsed = ref(false)

  // ===== 原生配置菜单（FE-13）：/yang/modules 驱动，指向通用模块控制台 =====
  // 原生配置 = 直接基于 YANG 模型的设备配置管理（Stack B 直连主链路）。
  // 「业务网络配置」为未来扩展层（业务侧 YANG 模型定义自动化能力，USMP 编排为
  // 原生配置下发），方向见 openspec/tasks/business-network-config.md。
  const nativeModules = ref<NativeModel[]>([])
  const nativeLoaded = ref(false)

  // ===== SND 左树（LT-03）：/yang/left-tree 驱动 14 组/3 层导航；失败回退
  // category 分组（leftTree 为空即回退态，R08 导航不消失）。 =====
  const leftTree = ref<LeftTreeNode[]>([])
  const leftTreeLoaded = ref(false)

  async function loadLeftTree() {
    if (leftTreeLoaded.value) return
    try {
      const res = await getLeftTree()
      const tree = res.data.data || []
      if (!tree.length) throw new Error('empty left tree')
      leftTree.value = tree
    } catch (e) {
      console.warn('加载 SND 左树失败，回退任务域分组导航:', e)
      leftTree.value = []
    } finally {
      leftTreeLoaded.value = true
    }
  }

  async function loadNativeModules() {
    if (nativeLoaded.value) return
    try {
      // 必须走 api 客户端（绝对 baseURL）：staging nginx 不代理 /api，裸相对
      // fetch('/api/...') 会命中 SPA fallback 返回 index.html → JSON 解析报错。
      const res = await listYangModules()
      const data = res.data
      const mods = (data.data || []).map((m: any) => ({
        name: m.name,
        title: m.description || m.title || m.name,
        vendor: m.vendor || i18n.global.t('nav.otherGroup'),
        category: m.category,
      }))
      if (!mods.length) throw new Error('empty modules')
      nativeModules.value = mods
    } catch (e) {
      console.warn('加载 YANG 模块列表失败，回退内置菜单:', e)
      // 回退项（R08）：与后端注册的模块根名一致（GetSchema/{name} 可直接命中）。
      nativeModules.value = [
        { name: 'ifm', title: i18n.global.t('nav.fallbackInterface'), vendor: 'huawei' },
        { name: 'vlan', title: i18n.global.t('nav.fallbackVlan'), vendor: 'huawei' },
      ]
    } finally {
      nativeLoaded.value = true
    }
  }

  // 业务网络配置模块（FE-17）：task-name=business-network 的模块归业务菜单组
  // （意图层，平台作用域控制台 /business/:module），不进「原生配置」分组。
  const BUSINESS_CATEGORY = 'business-network'
  const businessModules = computed(() =>
    nativeModules.value.filter((m) => m.category === BUSINESS_CATEGORY),
  )

  // 原生模块按任务域聚合（FE-13）：category 首现序，未标注归默认组('')排最后；
  // 全部未标注 → 单一默认组，菜单退化为平铺（R08 渲染不失败）。
  const nativeGroups = computed(() => {
    const order: string[] = []
    const byCat = new Map<string, NativeModel[]>()
    for (const m of nativeModules.value) {
      if (m.category === BUSINESS_CATEGORY) continue
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

  function toggleCollapse() {
    isCollapsed.value = !isCollapsed.value
  }

  return {
    isCollapsed,
    leftTree,
    leftTreeLoaded,
    loadLeftTree,
    nativeModules,
    nativeGroups,
    businessModules,
    nativeLoaded,
    loadNativeModules,
    toggleCollapse
  }
})
