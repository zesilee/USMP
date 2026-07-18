import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useMenuStore } from '../../src/stores/menu'
import * as apiModule from '../../src/api'

// FE-13：原生配置菜单由 /yang/modules 驱动；API 失败回退内置项（R08）。
// 走 api 客户端（绝对 baseURL）——staging nginx 不代理 /api，裸相对 fetch 会
// 命中 SPA fallback 返回 index.html（回归见 bugfix「Unexpected token '<'」）。
function mockModules(data: any[]) {
  return vi.spyOn(apiModule, 'listYangModules').mockResolvedValue({ data: { data } } as any)
}

describe('menu store · loadNativeModules', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('成功：模块列表映射为 {name,title,vendor}', async () => {
    mockModules([
      { name: 'ifm', title: 'ifm', vendor: 'huawei', description: '接口管理' },
      { name: 'vlan', title: 'vlan', vendor: 'huawei', description: '' },
    ])
    const store = useMenuStore()
    await store.loadNativeModules()
    expect(store.nativeModules).toEqual([
      { name: 'ifm', title: '接口管理', vendor: 'huawei' },
      { name: 'vlan', title: 'vlan', vendor: 'huawei' },
    ])
    expect(store.nativeLoaded).toBe(true)
  })

  it('失败：回退内置项（模块根名，可直接命中 GetSchema）', async () => {
    vi.spyOn(apiModule, 'listYangModules').mockRejectedValue(new Error('network down'))
    const store = useMenuStore()
    await store.loadNativeModules()
    expect(store.nativeModules.map((m) => m.name)).toEqual(['ifm', 'vlan'])
    expect(store.nativeLoaded).toBe(true)
  })

  // 回归（T07）：走 api 客户端 = axios，非 JSON 响应（如 staging nginx SPA fallback
  // 返回 index.html）由 axios 抛错，被 catch 稳妥回退，绝不冒泡到控制台崩菜单。
  it('回归：非 JSON 响应（HTML）稳妥回退，不抛出', async () => {
    vi.spyOn(apiModule, 'listYangModules').mockRejectedValue(
      new SyntaxError("Unexpected token '<', \"<!DOCTYPE \"... is not valid JSON")
    )
    const store = useMenuStore()
    await expect(store.loadNativeModules()).resolves.toBeUndefined()
    expect(store.nativeModules.map((m) => m.name)).toEqual(['ifm', 'vlan'])
    expect(store.nativeLoaded).toBe(true)
  })

  it('空列表也走回退（避免空菜单）', async () => {
    mockModules([])
    const store = useMenuStore()
    await store.loadNativeModules()
    expect(store.nativeModules.length).toBeGreaterThan(0)
  })

  it('幂等：已加载不再发请求', async () => {
    const f = mockModules([{ name: 'ifm' }])
    const store = useMenuStore()
    await store.loadNativeModules()
    await store.loadNativeModules()
    expect(f).toHaveBeenCalledTimes(1)
  })
})

// FE-13 扩展：模块携带 category（任务域）时按组聚合；无 category 归默认组，渲染不失败（R08）。
describe('menu store · nativeGroups 任务域分组', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('带/不带 category 混合：按任务域聚合，未标注归「其他」且排最后', async () => {
    mockModules([
      { name: 'ifm', title: 'ifm', vendor: 'huawei', description: '接口管理', category: 'interface-mgr' },
      { name: 'vlan', title: 'vlan', vendor: 'huawei', description: 'VLAN', category: 'vlan' },
      { name: 'bgp', title: 'bgp', vendor: 'huawei', description: 'BGP' },
    ])
    const store = useMenuStore()
    await store.loadNativeModules()
    const groups = store.nativeGroups
    expect(groups.map((g) => g.category)).toEqual(['interface-mgr', 'vlan', ''])
    expect(groups[0].modules.map((m) => m.name)).toEqual(['ifm'])
    expect(groups[2].modules.map((m) => m.name)).toEqual(['bgp'])
  })

  it('全部无 category（含回退项）：单一默认组，等价平铺', async () => {
    vi.spyOn(apiModule, 'listYangModules').mockRejectedValue(new Error('down'))
    const store = useMenuStore()
    await store.loadNativeModules()
    expect(store.nativeGroups).toHaveLength(1)
    expect(store.nativeGroups[0].category).toBe('')
    expect(store.nativeGroups[0].modules.length).toBeGreaterThan(0)
  })
})
