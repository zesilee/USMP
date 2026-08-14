import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useMenuStore } from '../../src/stores/menu'
import * as apiModule from '../../src/api'
import { resetStores } from './reset'

// LT-03 F1：loadLeftTree 装配与失败降级（leftTree 空即回退态，导航不消失 R08）。
function mockLeftTree(data: any[]) {
  return vi.spyOn(apiModule, 'getLeftTree').mockResolvedValue({ data: { data } } as any)
}

const S = () => useMenuStore.getState()

const sampleTree = [
  {
    zh: '以太网交换',
    en: 'Ethernet Switching',
    children: [
      {
        zh: 'VLAN',
        en: 'VLAN',
        children: [
          {
            zh: 'huawei-vlan', en: 'huawei-vlan', sourceModule: 'huawei-vlan', available: true, module: 'vlan',
            // LT-02 模块级子节点：container/rpc 平级（kind/name/highRisk 原样入 store）。
            children: [
              { zh: 'VLAN配置', en: 'VLAN Config', kind: 'container', name: 'vlan' },
              { zh: '重启VLAN', en: 'Restart VLAN', kind: 'rpc', name: 'restart-vlan', highRisk: true },
            ],
          },
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
    resetStores()
    vi.restoreAllMocks()
  })

  it('成功：树原样入 store，leftTreeLoaded 置位', async () => {
    mockLeftTree(sampleTree)
    await S().loadLeftTree()
    expect(S().leftTree).toHaveLength(2)
    expect(S().leftTree[0].zh).toBe('以太网交换')
    expect(S().leftTree[0].children![0].children![0].module).toBe('vlan')
    // LT-02 children：kind/name/highRisk 原样透传（类型收编进 LeftTreeNode）。
    const moduleChildren = S().leftTree[0].children![0].children![0].children!
    expect(moduleChildren[0].kind).toBe('container')
    expect(moduleChildren[0].name).toBe('vlan')
    expect(moduleChildren[1].kind).toBe('rpc')
    expect(moduleChildren[1].highRisk).toBe(true)
    expect(S().leftTreeLoaded).toBe(true)
  })

  it('失败：leftTree 为空（回退态），不抛错', async () => {
    vi.spyOn(apiModule, 'getLeftTree').mockRejectedValue(new Error('down'))
    await S().loadLeftTree()
    expect(S().leftTree).toEqual([])
    expect(S().leftTreeLoaded).toBe(true)
  })

  it('空树视同失败（回退态）', async () => {
    mockLeftTree([])
    await S().loadLeftTree()
    expect(S().leftTree).toEqual([])
  })

  it('幂等：已加载不重复请求', async () => {
    const spy = mockLeftTree(sampleTree)
    await S().loadLeftTree()
    await S().loadLeftTree()
    expect(spy).toHaveBeenCalledTimes(1)
  })
})
