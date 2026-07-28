import { describe, it, expect } from 'vitest'
import { parseLeafref } from '../../src/utils/leafref'

describe('parseLeafref（FE-19：leafref → 拉取路径/list键/key叶）', () => {
  it('标准 leafref：拉父容器、list键=次末段、key叶=末段', () => {
    const t = parseLeafref('/ifm:ifm/ifm:interfaces/ifm:interface/ifm:name')
    expect(t).toEqual({
      fetchPath: '/ifm:ifm/ifm:interfaces',
      listKey: 'interface',
      keyField: 'name',
    })
  })

  it('保留模块前缀于 fetchPath，局部名于 listKey/keyField', () => {
    const t = parseLeafref('/vlan:vlan/vlan:vlans/vlan:vlan/vlan:id')
    expect(t?.fetchPath).toBe('/vlan:vlan/vlan:vlans')
    expect(t?.listKey).toBe('vlan')
    expect(t?.keyField).toBe('id')
  })

  it('空/过短/无效 → null（降级文本输入）', () => {
    expect(parseLeafref(undefined)).toBeNull()
    expect(parseLeafref('')).toBeNull()
    expect(parseLeafref('/name')).toBeNull()
    expect(parseLeafref('/a/b')).toBeNull() // fetchPath 会是 '/'，无效
  })
})
