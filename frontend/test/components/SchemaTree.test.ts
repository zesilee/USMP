import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import SchemaTree from '../../src/components/config/SchemaTree.vue'
import type { Field } from '../../src/utils/crdSchemaParser'

const fields: Field[] = [
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
          { path: '/vlan/vlans/vlan/state', type: 'string', label: 'state', readonly: true },
        ],
      },
    ],
  },
]

describe('SchemaTree · YANG 架构树', () => {
  it('渲染 container/list/leaf 三种 kind 标签', () => {
    const w = mount(SchemaTree, { props: { fields } })
    const kinds = w.findAll('.ynode .kind').map((n) => n.text())
    expect(kinds).toContain('容器')
    expect(kinds).toContain('列表')
    expect(kinds).toContain('叶子')
  })

  it('叶子右侧展示 YANG 数据类型', () => {
    const w = mount(SchemaTree, { props: { fields } })
    const idNode = w.findAll('.ynode').find((n) => n.find('.nm').text() === 'id')!
    expect(idNode.find('.ty').text()).toBe('number')
  })

  it('提供 keyField 时主键叶子显示 key 标记', () => {
    const w = mount(SchemaTree, { props: { fields, keyField: 'id' } })
    const idNode = w.findAll('.ynode').find((n) => n.find('.nm').text() === 'id')!
    expect(idNode.find('.keyt').exists()).toBe(true)
    const nameNode = w.findAll('.ynode').find((n) => n.find('.nm').text() === 'name')!
    expect(nameNode.find('.keyt').exists()).toBe(false)
  })

  it('可配置叶子带 cfg 类，只读叶子带 ro 类', () => {
    const w = mount(SchemaTree, { props: { fields } })
    const nameNode = w.findAll('.ynode').find((n) => n.find('.nm').text() === 'name')!
    expect(nameNode.classes()).toContain('cfg')
    const stateNode = w.findAll('.ynode').find((n) => n.find('.nm').text() === 'state')!
    expect(stateNode.classes()).toContain('ro')
    expect(stateNode.classes()).not.toContain('cfg')
  })

  it('depth 决定缩进（padding-left = 10 + depth*14）', () => {
    const w = mount(SchemaTree, { props: { fields } })
    const vlansNode = w.findAll('.ynode').find((n) => n.find('.nm').text() === 'vlans')!
    expect(vlansNode.attributes('style')).toContain('padding-left: 10px')
    const idNode = w.findAll('.ynode').find((n) => n.find('.nm').text() === 'id')!
    expect(idNode.attributes('style')).toContain('padding-left: 38px') // depth 2 → 10+28
  })

  it('itemCounts 命中的 list 节点渲染数量 pill', () => {
    const w = mount(SchemaTree, { props: { fields, itemCounts: { '/vlan/vlans/vlan': 7 } } })
    const listNode = w.findAll('.ynode').find((n) => n.find('.nm').text() === 'vlan')!
    expect(listNode.find('.count-pill').text()).toBe('7')
  })

  it('显示模块标签与图例脚注', () => {
    const w = mount(SchemaTree, { props: { fields, moduleLabel: 'huawei-vlan' } })
    expect(w.find('.tree-h').text()).toContain('huawei-vlan')
    expect(w.find('.tree-foot').text()).toContain('可配置字段')
  })

  it('空 fields 显示占位、不渲染脚注（R08 降级）', () => {
    const w = mount(SchemaTree, { props: { fields: [] } })
    expect(w.find('.tree-empty').exists()).toBe(true)
    expect(w.find('.tree-foot').exists()).toBe(false)
  })

  it('点击节点派发 node-click', async () => {
    const w = mount(SchemaTree, { props: { fields } })
    await w.find('.ynode').trigger('click')
    expect(w.emitted('node-click')).toBeTruthy()
  })
})
