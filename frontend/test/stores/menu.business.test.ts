import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useMenuStore } from '../../src/stores/menu'

// FE-13：业务配置菜单由 /yang/modules 驱动；API 失败回退内置项（R08）。
describe('menu store · loadBusinessModules', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('成功：模块列表映射为 {name,title,vendor}', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      json: async () => ({
        data: [
          { name: 'ifm', title: 'ifm', vendor: 'huawei', description: '接口管理' },
          { name: 'vlan', title: 'vlan', vendor: 'huawei', description: '' },
        ],
      }),
    }))
    const store = useMenuStore()
    await store.loadBusinessModules()
    expect(store.businessModules).toEqual([
      { name: 'ifm', title: '接口管理', vendor: 'huawei' },
      { name: 'vlan', title: 'vlan', vendor: 'huawei' },
    ])
    expect(store.businessLoaded).toBe(true)
  })

  it('失败：回退内置项（模块根名，可直接命中 GetSchema）', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('network down')))
    const store = useMenuStore()
    await store.loadBusinessModules()
    expect(store.businessModules.map((m) => m.name)).toEqual(['ifm', 'vlan'])
    expect(store.businessLoaded).toBe(true)
  })

  it('空列表也走回退（避免空菜单）', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ json: async () => ({ data: [] }) }))
    const store = useMenuStore()
    await store.loadBusinessModules()
    expect(store.businessModules.length).toBeGreaterThan(0)
  })

  it('幂等：已加载不再发请求', async () => {
    const f = vi.fn().mockResolvedValue({ json: async () => ({ data: [{ name: 'ifm' }] }) })
    vi.stubGlobal('fetch', f)
    const store = useMenuStore()
    await store.loadBusinessModules()
    await store.loadBusinessModules()
    expect(f).toHaveBeenCalledTimes(1)
  })
})

// FE-13 扩展：模块携带 category（任务域）时按组聚合；无 category 归默认组，渲染不失败（R08）。
describe('menu store · businessGroups 任务域分组', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('带/不带 category 混合：按任务域聚合，未标注归「其他」且排最后', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      json: async () => ({
        data: [
          { name: 'ifm', title: 'ifm', vendor: 'huawei', description: '接口管理', category: 'interface-mgr' },
          { name: 'vlan', title: 'vlan', vendor: 'huawei', description: 'VLAN', category: 'vlan' },
          { name: 'interfaces', title: 'interfaces', vendor: 'openconfig', description: 'oc 接口' },
        ],
      }),
    }))
    const store = useMenuStore()
    await store.loadBusinessModules()
    const groups = store.businessGroups
    expect(groups.map((g) => g.category)).toEqual(['interface-mgr', 'vlan', ''])
    expect(groups[0].modules.map((m) => m.name)).toEqual(['ifm'])
    expect(groups[2].modules.map((m) => m.name)).toEqual(['interfaces'])
  })

  it('全部无 category（含回退项）：单一默认组，等价平铺', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('down')))
    const store = useMenuStore()
    await store.loadBusinessModules()
    expect(store.businessGroups).toHaveLength(1)
    expect(store.businessGroups[0].category).toBe('')
    expect(store.businessGroups[0].modules.length).toBeGreaterThan(0)
  })
})
