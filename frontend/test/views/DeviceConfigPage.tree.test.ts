import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import router from '../../src/router'
import App from '../../src/App.vue'
import { getYangSchema } from '../../src/api'

vi.mock('../../src/api')

// 嵌套 schema：container vlans → list vlan → 叶子（含主键 id）
const vlanNested = {
  fields: [
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
          ],
        },
      ],
    },
  ],
}

// PR-2a：设备配置页应从 /yang/schema 真数据渲染左侧 YANG 架构树（模型驱动，R05）。
describe('DeviceConfigPage 渲染 YANG 架构树（真 schema）', () => {
  beforeEach(() => {
    vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: vlanNested } } as any)
  })

  it('/config/vlan 展示架构树的 container/list/leaf 节点与 key 标记', async () => {
    const wrapper = mount(App, { global: { plugins: [createPinia(), ElementPlus, router] } })
    await router.push('/config/vlan')
    await flushPromises()

    const tree = wrapper.find('.schema-tree')
    expect(tree.exists()).toBe(true)
    const names = tree.findAll('.ynode .nm').map((n) => n.text())
    expect(names).toEqual(expect.arrayContaining(['vlans', 'vlan', 'id', 'name']))

    // keyField='id'（路由 options）→ id 叶子带 key 标记
    const idNode = tree.findAll('.ynode').find((n) => n.find('.nm').text() === 'id')!
    expect(idNode.find('.keyt').exists()).toBe(true)

    wrapper.unmount()
  })
})
