import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useMenuStore } from '../../src/stores/menu'
import * as apiModule from '../../src/api'

// LT-03 F1：loadLeftTree 装配与失败降级（leftTree 空即回退态，导航不消失 R08）。
function mockLeftTree(data: any[]) {
  return vi.spyOn(apiModule, 'getLeftTree').mockResolvedValue({ data: { data } } as any)
}

const sampleTree = [
  {
    zh: '以太网交换',
    en: 'Ethernet Switching',
    children: [
      {
        zh: 'VLAN',
        en: 'VLAN',
        children: [
          { zh: 'huawei-vlan', en: 'huawei-vlan', sourceModule: 'huawei-vlan', available: true, module: 'vlan' },
        ],
      },
    ],
  },
  {
    zh: '安全',
    en: 'Security',
    children: [{ zh: 'huawei-dsa', en: 'huawei-dsa', sourceModule: 'huawei-dsa', available: false }],
  },
]

describe('menu store · loadLeftTree（LT-03）', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.restoreAllMocks()
  })

  it('成功：树原样入 store，leftTreeLoaded 置位', async () => {
    mockLeftTree(sampleTree)
    const store = useMenuStore()
    await store.loadLeftTree()
    expect(store.leftTree).toHaveLength(2)
    expect(store.leftTree[0].zh).toBe('以太网交换')
    expect(store.leftTree[0].children![0].children![0].module).toBe('vlan')
    expect(store.leftTreeLoaded).toBe(true)
  })

  it('失败：leftTree 为空（回退态），不抛错', async () => {
    vi.spyOn(apiModule, 'getLeftTree').mockRejectedValue(new Error('down'))
    const store = useMenuStore()
    await store.loadLeftTree()
    expect(store.leftTree).toEqual([])
    expect(store.leftTreeLoaded).toBe(true)
  })

  it('空树视同失败（回退态）', async () => {
    mockLeftTree([])
    const store = useMenuStore()
    await store.loadLeftTree()
    expect(store.leftTree).toEqual([])
  })

  it('幂等：已加载不重复请求', async () => {
    const spy = mockLeftTree(sampleTree)
    const store = useMenuStore()
    await store.loadLeftTree()
    await store.loadLeftTree()
    expect(spy).toHaveBeenCalledTimes(1)
  })
})
