import { describe, it, expect } from 'vitest'
import { deriveSchemaTree, type SchemaTreeNode } from '../../src/utils/schemaTree'
import type { Field } from '../../src/utils/crdSchemaParser'

// 模拟后端 /yang/schema?form=nested 的完整树（container=group / list / leaf）。
const vlanSchema: Field[] = [
  { path: '/vlan/default-instance', type: 'group', label: 'default-instance', fields: [] },
  {
    path: '/vlan/vlans',
    type: 'group',
    label: 'vlans',
    fields: [
      {
        path: '/vlan/vlans/vlan',
        type: 'list',
        label: 'vlan',
        fields: [
          { path: '/vlan/vlans/vlan/id', type: 'number', label: 'id', required: true },
          { path: '/vlan/vlans/vlan/name', type: 'string', label: 'name' },
          { path: '/vlan/vlans/vlan/admin-status', type: 'enum', label: 'admin-status', options: [{ label: 'up', value: 'up' }] },
          {
            path: '/vlan/vlans/vlan/member-ports',
            type: 'group',
            label: 'member-ports',
            fields: [{ path: '/vlan/vlans/vlan/member-ports/member-port', type: 'list', label: 'member-port', fields: [] }],
          },
        ],
      },
    ],
  },
]

function byPath(nodes: SchemaTreeNode[], path: string): SchemaTreeNode {
  const n = nodes.find((x) => x.path === path)
  if (!n) throw new Error(`node not found: ${path}`)
  return n
}

describe('deriveSchemaTree', () => {
  it('把 group/list/leaf 映射为 container/list/leaf 三种 kind', () => {
    const nodes = deriveSchemaTree(vlanSchema)
    expect(byPath(nodes, '/vlan/vlans').kind).toBe('container')
    expect(byPath(nodes, '/vlan/vlans/vlan').kind).toBe('list')
    expect(byPath(nodes, '/vlan/vlans/vlan/id').kind).toBe('leaf')
  })

  it('DFS 前序展平并标注 depth（根=0，逐层 +1）', () => {
    const nodes = deriveSchemaTree(vlanSchema)
    expect(byPath(nodes, '/vlan/vlans').depth).toBe(0)
    expect(byPath(nodes, '/vlan/vlans/vlan').depth).toBe(1)
    expect(byPath(nodes, '/vlan/vlans/vlan/id').depth).toBe(2)
    expect(byPath(nodes, '/vlan/vlans/vlan/member-ports').depth).toBe(2)
    expect(byPath(nodes, '/vlan/vlans/vlan/member-ports/member-port').depth).toBe(3)
    // 前序：父在子之前
    const paths = nodes.map((n) => n.path)
    expect(paths.indexOf('/vlan/vlans')).toBeLessThan(paths.indexOf('/vlan/vlans/vlan'))
    expect(paths.indexOf('/vlan/vlans/vlan')).toBeLessThan(paths.indexOf('/vlan/vlans/vlan/id'))
  })

  it('叶子带出 YANG 数据类型；容器/列表无 dataType', () => {
    const nodes = deriveSchemaTree(vlanSchema)
    expect(byPath(nodes, '/vlan/vlans/vlan/id').dataType).toBe('number')
    expect(byPath(nodes, '/vlan/vlans/vlan/admin-status').dataType).toBe('enum')
    expect(byPath(nodes, '/vlan/vlans/vlan').dataType).toBeUndefined()
    expect(byPath(nodes, '/vlan/vlans').dataType).toBeUndefined()
  })

  it('keyField 仅标注 list 直接子叶子中同名者为 key', () => {
    const nodes = deriveSchemaTree(vlanSchema, { keyField: 'id' })
    expect(byPath(nodes, '/vlan/vlans/vlan/id').isKey).toBe(true)
    expect(byPath(nodes, '/vlan/vlans/vlan/name').isKey).toBe(false)
  })

  it('list 之外的同名叶子不会被误标 key', () => {
    const schema: Field[] = [
      { path: '/sys/id', type: 'string', label: 'id' }, // 顶层叶子，非 list 子节点
      {
        path: '/sys/things',
        type: 'list',
        label: 'things',
        fields: [{ path: '/sys/things/thing/id', type: 'string', label: 'id' }],
      },
    ]
    const nodes = deriveSchemaTree(schema, { keyField: 'id' })
    expect(byPath(nodes, '/sys/id').isKey).toBe(false)
    expect(byPath(nodes, '/sys/things/thing/id').isKey).toBe(true)
  })

  it('key 按 path 末段匹配，label 被本地化也不失效', () => {
    const schema: Field[] = [
      {
        path: '/vlan/vlans/vlan',
        type: 'list',
        label: 'vlan',
        fields: [{ path: '/vlan/vlans/vlan/id', type: 'number', label: '标识' }], // label 已本地化
      },
    ]
    const nodes = deriveSchemaTree(schema, { keyField: 'id' })
    const idNode = byPath(nodes, '/vlan/vlans/vlan/id')
    expect(idNode.name).toBe('标识') // 展示仍用 label
    expect(idNode.isKey).toBe(true) // 但 key 匹配走 path 末段，稳
  })

  it('未提供 keyField 时不标注任何 key', () => {
    const nodes = deriveSchemaTree(vlanSchema)
    expect(nodes.every((n) => !n.isKey)).toBe(true)
  })

  it('可配置叶子 isConfig=true；readonly 叶子 isConfig=false 且 isReadonly=true', () => {
    const schema: Field[] = [
      { path: '/x/a', type: 'string', label: 'a' },
      { path: '/x/b', type: 'string', label: 'b', readonly: true },
    ]
    const nodes = deriveSchemaTree(schema)
    expect(byPath(nodes, '/x/a').isConfig).toBe(true)
    expect(byPath(nodes, '/x/a').isReadonly).toBe(false)
    expect(byPath(nodes, '/x/b').isConfig).toBe(false)
    expect(byPath(nodes, '/x/b').isReadonly).toBe(true)
  })

  it('容器/列表不是可配置字段（isConfig=false）', () => {
    const nodes = deriveSchemaTree(vlanSchema)
    expect(byPath(nodes, '/vlan/vlans').isConfig).toBe(false)
    expect(byPath(nodes, '/vlan/vlans/vlan').isConfig).toBe(false)
  })

  it('required 透传', () => {
    const nodes = deriveSchemaTree(vlanSchema)
    expect(byPath(nodes, '/vlan/vlans/vlan/id').required).toBe(true)
    expect(byPath(nodes, '/vlan/vlans/vlan/name').required).toBe(false)
  })

  it('name 优先取 label，回退到 path 末段', () => {
    const schema: Field[] = [{ path: '/a/b/c', type: 'string', label: '' }]
    const nodes = deriveSchemaTree(schema)
    expect(nodes[0].name).toBe('c')
  })

  it('空/异常输入安全降级为空数组（R08）', () => {
    expect(deriveSchemaTree([])).toEqual([])
    expect(deriveSchemaTree(null as any)).toEqual([])
    expect(deriveSchemaTree(undefined as any)).toEqual([])
  })
})
