import { describe, it, expect } from 'vitest'
import { extractRows } from '../../src/utils/extractRows'

// 自 useDeviceConfig 退役迁入（唯一存活消费方：RpcExecuteTab 的 leafref 下拉取数）。
describe('extractRows · 运行配置归一化为行数组', () => {
  it('兼容 {listKey:[...]}、数组、以主键为键的 map', () => {
    expect(extractRows({ data: { vlans: [{ id: 100 }] } }, 'vlans', 'id')).toEqual([{ id: 100 }])
    expect(extractRows([{ id: 200 }], 'vlans', 'id')).toEqual([{ id: 200 }])
    const fromMap = extractRows({ interface: { 'GE0/0/1': { mtu: 1500 } } }, 'interface', 'name')
    expect(fromMap[0]).toMatchObject({ name: 'GE0/0/1', mtu: 1500 })
  })

  it('数字键 map 的主键还原为 number', () => {
    const rows = extractRows({ vlan: { '100': { name: 'v100' } } }, 'vlan', 'id')
    expect(rows[0]).toMatchObject({ id: 100, name: 'v100' })
  })

  it('对空/异常输入返回空数组（R08 降级）', () => {
    expect(extractRows(null, 'vlans', 'id')).toEqual([])
    expect(extractRows({}, 'vlans', 'id')).toEqual([])
    expect(extractRows('garbage' as any, 'vlans', 'id')).toEqual([])
  })
})
