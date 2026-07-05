import { describe, it, expect, vi, beforeEach } from 'vitest'
import { extractVlanItemFields, extractVlanRows, useVlanConfig } from '../../src/composables/useVlanConfig'
import { getYangSchema } from '../../src/api'

vi.mock('../../src/api')

// 模拟后端 ?form=nested 的 VLAN schema（container vlans → list vlan → 叶子 + member-ports）
const nestedSchema = {
  module: 'vlan',
  fields: [
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
            { path: '/vlan/vlans/vlan/id', type: 'number', label: 'id' },
            { path: '/vlan/vlans/vlan/admin-status', type: 'enum', label: 'admin-status', options: [{ label: 'up', value: 'up' }] },
            {
              path: '/vlan/vlans/vlan/member-ports',
              type: 'group',
              label: 'member-ports',
              fields: [
                {
                  path: '/vlan/vlans/vlan/member-ports/member-port',
                  type: 'list',
                  label: 'member-port',
                  fields: [{ path: '/vlan/vlans/vlan/member-ports/member-port/interface-name', type: 'string', label: 'interface-name' }],
                },
              ],
            },
          ],
        },
      ],
    },
  ],
}

describe('useVlanConfig helpers', () => {
  it('extractVlanItemFields 应从嵌套 schema 取出单个 VLAN 的字段集', () => {
    const fields = extractVlanItemFields(nestedSchema)
    const labels = fields.map((f) => f.label)
    expect(labels).toContain('id')
    expect(labels).toContain('admin-status')
    expect(labels).toContain('member-ports')
    // member-ports 是 group，内含 member-port(list)
    const mp = fields.find((f) => f.label === 'member-ports')
    expect(mp?.type).toBe('group')
    expect(mp?.fields?.[0]?.type).toBe('list')
  })

  it('extractVlanRows 兼容 {vlans:[...]}、数组、以 id 为键的 map', () => {
    expect(extractVlanRows({ data: { vlans: [{ id: 100 }] } })).toEqual([{ id: 100 }])
    expect(extractVlanRows([{ id: 200 }])).toEqual([{ id: 200 }])
    const fromMap = extractVlanRows({ '300': { name: 'Mgmt' } })
    expect(fromMap[0]).toMatchObject({ id: 300, name: 'Mgmt' })
  })

  it('extractVlanRows 对空/异常输入返回空数组（R08 降级）', () => {
    expect(extractVlanRows(null)).toEqual([])
    expect(extractVlanRows({})).toEqual([])
  })

  describe('loadSchema（走 api 客户端而非裸 fetch，staging 无 nginx /api 代理）', () => {
    beforeEach(() => {
      vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: nestedSchema } } as any)
    })

    it('应用 getYangSchema 拉取并填充单 VLAN 字段（新增表单不再空）', async () => {
      const vlan = useVlanConfig()
      await vlan.loadSchema()
      expect(getYangSchema).toHaveBeenCalledWith('vlan', 'nested')
      const labels = vlan.fields.value.map((f) => f.label)
      expect(labels).toContain('admin-status')
      expect(labels).toContain('member-ports')
    })
  })
})
