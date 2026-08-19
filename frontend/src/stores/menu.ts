import { create } from './createStore'
import { listYangModules, getLeftTree } from '../api'
import { i18n } from '../i18n'

// SND 左树节点（LT-03）：分组（children）、叶子（sourceModule；available/module
// 标注），或叶子下的模块级子节点（kind=container/rpc，与 YANG 模块顶层同级平铺，
// LT-02）——container 路由 /module/<name>、rpc 路由 /module/<叶module>/rpc/<name>。
export interface LeftTreeNode {
  zh: string
  en: string
  sourceModule?: string
  available?: boolean
  module?: string
  supported?: boolean
  kind?: 'container' | 'rpc'
  name?: string
  highRisk?: boolean
  children?: LeftTreeNode[]
}

interface NativeModel {
  name: string
  title: string
  vendor: string
  // 任务域（后端 /yang/modules category，源自模块级 task-name 扩展，FE-13）。
  category?: string
}

export interface NativeGroup {
  category: string
  modules: NativeModel[]
}

// 业务网络配置模块（FE-17）：task-name=business-network 的模块归业务菜单组
// （意图层，平台作用域控制台 /business/:module），不进「原生配置」分组。
const BUSINESS_CATEGORY = 'business-network'

interface MenuState {
  isCollapsed: boolean
  // ===== 原生配置菜单（FE-13）：/yang/modules 驱动，指向通用模块控制台 =====
  nativeModules: NativeModel[]
  nativeLoaded: boolean
  // ===== SND 左树（LT-03）：/yang/left-tree 驱动 14 组/3 层导航；失败回退
  // category 分组（leftTree 为空即回退态，R08 导航不消失）。 =====
  leftTree: LeftTreeNode[]
  leftTreeLoaded: boolean
  loadLeftTree: () => Promise<void>
  loadNativeModules: () => Promise<void>
  /** 业务菜单组（旧 Pinia computed → 方法形态）。
   *  ⚠️ 每次调用返回新数组：勿直接放进 zustand selector（getSnapshot 不稳定会
   *  无限重渲染）——组件里取 nativeModules 后自行 useMemo。 */
  businessModules: () => NativeModel[]
  /** 原生模块按任务域聚合（FE-13）：category 首现序，未标注归默认组('')排最后；
   *  全部未标注 → 单一默认组，菜单退化为平铺（R08 渲染不失败）。
   *  ⚠️ selector 警示同 businessModules。 */
  nativeGroups: () => NativeGroup[]
  toggleCollapse: () => void
}

export const useMenuStore = create<MenuState>((set, get) => ({
  isCollapsed: false,
  nativeModules: [],
  nativeLoaded: false,
  leftTree: [],
  leftTreeLoaded: false,

  loadLeftTree: async () => {
    if (get().leftTreeLoaded) return
    try {
      const res = await getLeftTree()
      const tree = res.data.data || []
      if (!tree.length) throw new Error('empty left tree')
      set({ leftTree: tree })
    } catch (e) {
      console.warn('left-tree load failed, fallback to category groups:', e)
      set({ leftTree: [] })
    } finally {
      set({ leftTreeLoaded: true })
    }
  },

  loadNativeModules: async () => {
    if (get().nativeLoaded) return
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
      set({ nativeModules: mods })
    } catch (e) {
      console.warn('yang modules load failed, fallback to built-in menu:', e)
      // 回退项（R08）：与后端注册的模块根名一致（GetSchema/{name} 可直接命中）。
      set({
        nativeModules: [
          { name: 'ifm', title: i18n.global.t('nav.fallbackInterface'), vendor: 'huawei' },
          { name: 'vlan', title: i18n.global.t('nav.fallbackVlan'), vendor: 'huawei' },
        ],
      })
    } finally {
      set({ nativeLoaded: true })
    }
  },

  businessModules: () => get().nativeModules.filter((m) => m.category === BUSINESS_CATEGORY),

  nativeGroups: () => {
    const order: string[] = []
    const byCat = new Map<string, NativeModel[]>()
    for (const m of get().nativeModules) {
      if (m.category === BUSINESS_CATEGORY) continue
      const c = m.category || ''
      if (!byCat.has(c)) {
        byCat.set(c, [])
        if (c) order.push(c)
      }
      byCat.get(c)!.push(m)
    }
    const out: NativeGroup[] = order.map((c) => ({ category: c, modules: byCat.get(c)! }))
    if (byCat.has('')) out.push({ category: '', modules: byCat.get('')! })
    return out
  },

  toggleCollapse: () => set((s) => ({ isCollapsed: !s.isCollapsed })),
}))
