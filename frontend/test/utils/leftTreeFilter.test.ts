import { describe, it, expect } from 'vitest'
import type { LeftTreeNode } from '../../src/stores/menu'
import { filterLeftTree } from '../../src/utils/leftTreeFilter'

// NCE 左树形测试树：分组 → 模块叶 → container/rpc 子节点（LT-05）。
const tree: LeftTreeNode[] = [
  {
    zh: '接口管理',
    en: 'Interface Management',
    children: [
      {
        zh: '接口基础',
        en: 'Interface Basics',
        children: [
          {
            zh: '通用接口管理',
            en: 'Common Interface',
            sourceModule: 'huawei-ifm',
            available: true,
            module: 'ifm',
            children: [
              { zh: '通用接口', en: 'Common Interface', kind: 'container', name: 'ifm' },
              { zh: '重启接口', en: 'Restart Interface', kind: 'rpc', name: 'restart-if', highRisk: true },
            ],
          },
        ],
      },
    ],
  },
  {
    zh: '以太网交换',
    en: 'Ethernet Switching',
    children: [
      {
        zh: 'VLAN',
        en: 'VLAN',
        sourceModule: 'huawei-vlan',
        available: true,
        module: 'vlan',
        children: [{ zh: 'VLAN', en: 'VLAN', kind: 'container', name: 'vlan' }],
      },
      { zh: '端口聚合', en: 'Eth-Trunk', sourceModule: 'huawei-trunk', available: false },
    ],
  },
]

describe('filterLeftTree · 左树节点名过滤（LT-05）', () => {
  it('命中叶保留祖先链，未命中分支剪除', () => {
    const out = filterLeftTree(tree, '通用接口')
    expect(out).toHaveLength(1)
    expect(out[0].zh).toBe('接口管理')
    expect(out[0].children![0].zh).toBe('接口基础')
    expect(out[0].children![0].children![0].sourceModule).toBe('huawei-ifm')
  })

  it('zh/en/name 三口径命中且大小写不敏感', () => {
    expect(filterLeftTree(tree, 'ethernet')).toHaveLength(1)
    expect(filterLeftTree(tree, 'RESTART-IF')[0].zh).toBe('接口管理')
    expect(filterLeftTree(tree, 'vlan')[0].zh).toBe('以太网交换')
  })

  it('命中分组/模块叶保留整棵子树（可继续下钻）', () => {
    const out = filterLeftTree(tree, '以太网交换')
    expect(out).toHaveLength(1)
    expect(out[0].children).toHaveLength(2)
    const leaf = filterLeftTree(tree, '通用接口管理')
    expect(leaf[0].children![0].children![0].children).toHaveLength(2)
  })

  it('命中不可用叶保留且不改可用性语义（负路径）', () => {
    const out = filterLeftTree(tree, '端口聚合')
    const leaf = out[0].children![0]
    expect(leaf.sourceModule).toBe('huawei-trunk')
    expect(leaf.available).toBe(false)
  })

  it('部分命中仅保留命中子分支', () => {
    const out = filterLeftTree(tree, 'VLAN')
    expect(out[0].children).toHaveLength(1)
    expect(out[0].children![0].sourceModule).toBe('huawei-vlan')
  })

  it('无命中 → 空数组；空查询 → 原树原样', () => {
    expect(filterLeftTree(tree, '不存在的节点')).toEqual([])
    expect(filterLeftTree(tree, '')).toBe(tree)
    expect(filterLeftTree(tree, '   ')).toBe(tree)
  })

  it('输入树不被修改（纯函数）', () => {
    const snapshot = JSON.stringify(tree)
    filterLeftTree(tree, '通用接口')
    expect(JSON.stringify(tree)).toBe(snapshot)
  })
})
