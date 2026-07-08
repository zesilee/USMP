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
