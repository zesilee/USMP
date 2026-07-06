import { describe, it, expect, vi, beforeEach } from 'vitest'
import { extractItemFields, extractRows, findItemListPath, useDeviceConfig } from '../../src/composables/useDeviceConfig'
import { getYangSchema } from '../../src/api'

vi.mock('../../src/api')

// 模拟嵌套 schema（container vlans → list vlan → 叶子 + member-ports）
const nestedSchema = {
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
              fields: [{ path: '/vlan/vlans/vlan/member-ports/member-port', type: 'list', label: 'member-port', fields: [] }],
            },
          ],
        },
      ],
    },
  ],
}

describe('useDeviceConfig helpers', () => {
  it('extractItemFields 用 DFS 按 path 后缀取出 list 的字段集', () => {
    const fields = extractItemFields(nestedSchema, '/vlan')
    const labels = fields.map((f) => f.label)
    expect(labels).toContain('id')
    expect(labels).toContain('admin-status')
    expect(labels).toContain('member-ports')
  })

  it('extractItemFields 支持不同模块的 list 后缀（如 /interface）', () => {
    const ifmSchema = {
      fields: [
        {
          path: '/ifm/interfaces',
          type: 'group',
          label: 'interfaces',
          fields: [
            { path: '/ifm/interfaces/interface', type: 'list', label: 'interface', fields: [{ path: '/ifm/interfaces/interface/name', type: 'string', label: 'name' }] },
          ],
        },
      ],
    }
    const fields = extractItemFields(ifmSchema, '/interface')
    expect(fields.map((f) => f.label)).toContain('name')
  })

  it('extractRows 兼容 {listKey:[...]}、数组、以主键为键的 map', () => {
    expect(extractRows({ data: { vlans: [{ id: 100 }] } }, 'vlans', 'id')).toEqual([{ id: 100 }])
    expect(extractRows([{ id: 200 }], 'vlans', 'id')).toEqual([{ id: 200 }])
    const fromMap = extractRows({ interface: { 'GE0/0/1': { mtu: 1500 } } }, 'interface', 'name')
    expect(fromMap[0]).toMatchObject({ name: 'GE0/0/1', mtu: 1500 })
  })

  it('extractRows 对空/异常输入返回空数组（R08 降级）', () => {
    expect(extractRows(null, 'vlans', 'id')).toEqual([])
    expect(extractRows({}, 'vlans', 'id')).toEqual([])
  })

  it('findItemListPath 返回目标 list 的完整 path，未命中返回空串', () => {
    expect(findItemListPath(nestedSchema, '/vlan')).toBe('/vlan/vlans/vlan')
    expect(findItemListPath(nestedSchema, '/nope')).toBe('')
    expect(findItemListPath(null, '/vlan')).toBe('')
  })

  describe('loadSchema（走 api 客户端而非裸 fetch，staging 无 nginx /api 代理）', () => {
    beforeEach(() => {
      vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: nestedSchema } } as any)
    })

    it('应用 getYangSchema 拉取并按 module/suffix 填充字段', async () => {
      const cfg = useDeviceConfig({ module: 'vlan', configPath: 'huawei-vlan:vlan/vlans', itemListSuffix: '/vlan', listKey: 'vlans', keyField: 'id' })
      await cfg.loadSchema()
      expect(getYangSchema).toHaveBeenCalledWith('vlan', 'nested')
      expect(cfg.fields.value.map((f) => f.label)).toContain('admin-status')
    })

    it('同时暴露完整 schema 树与目标 list path（供架构树渲染/数量 pill）', async () => {
      const cfg = useDeviceConfig({ module: 'vlan', configPath: 'huawei-vlan:vlan/vlans', itemListSuffix: '/vlan', listKey: 'vlans', keyField: 'id' })
      await cfg.loadSchema()
      // schemaFields = 顶层容器（default-instance / vlans），非仅单条记录字段
      expect(cfg.schemaFields.value.map((f) => f.label)).toEqual(['default-instance', 'vlans'])
      expect(cfg.itemListPath.value).toBe('/vlan/vlans/vlan')
    })
  })
})
