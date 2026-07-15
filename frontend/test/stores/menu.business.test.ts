import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useMenuStore } from '../../src/stores/menu'
import * as apiModule from '../../src/api'

// FE-17（F1）——业务模块分桶：task-name=business-network 的模块进 businessModules、
// 不进原生分组；无业务模块时业务组为空（Sidebar 整组隐藏）。

const modules = [
  { name: 'vlan', description: 'VLAN 配置', vendor: 'huawei', category: 'vlan' },
  { name: 'ifm', description: '接口管理', vendor: 'huawei', category: 'interface-mgr' },
  {
    name: 'business-vlan-service',
    description: '跨设备 VLAN 打通',
    vendor: 'usmp',
    category: 'business-network',
  },
]

describe('menu store — business modules split (FE-17)', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('business-network 模块进 businessModules 且不进 nativeGroups', async () => {
    vi.spyOn(apiModule, 'listYangModules').mockResolvedValue({
      data: { data: modules },
    } as any)
    const store = useMenuStore()
    await store.loadNativeModules()

    expect(store.businessModules.map((m) => m.name)).toEqual(['business-vlan-service'])
    expect(store.businessModules[0].title).toBe('跨设备 VLAN 打通')

    const grouped = store.nativeGroups.flatMap((g) => g.modules.map((m) => m.name))
    expect(grouped).toContain('vlan')
    expect(grouped).toContain('ifm')
    expect(grouped).not.toContain('business-vlan-service')
  })

  it('无业务模块 → businessModules 为空（组隐藏），原生分组不受影响', async () => {
    vi.spyOn(apiModule, 'listYangModules').mockResolvedValue({
      data: { data: modules.filter((m) => m.category !== 'business-network') },
    } as any)
    const store = useMenuStore()
    await store.loadNativeModules()

    expect(store.businessModules).toHaveLength(0)
    expect(store.nativeGroups.length).toBeGreaterThan(0)
  })

  it('加载失败回退内置项（无业务模块），不抛错（R08）', async () => {
    vi.spyOn(apiModule, 'listYangModules').mockRejectedValue(new Error('boom'))
    const store = useMenuStore()
    await store.loadNativeModules()

    expect(store.businessModules).toHaveLength(0)
    expect(store.nativeModules.length).toBeGreaterThan(0)
  })
})
