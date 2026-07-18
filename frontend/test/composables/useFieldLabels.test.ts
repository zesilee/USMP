import { describe, it, expect } from 'vitest'
import { localizeFields, sourceModuleFor, resKeyFor } from '../../src/composables/useFieldLabels'

// UI-03 F1：res 查表（真实 vlan res 副本）、路径换算、缺键/缺文件回退、双语。
describe('useFieldLabels（UI-03）', () => {
  const fields = [
    {
      path: '/vlan/vlans',
      type: 'list',
      label: 'vlans',
      fields: [
        { path: '/vlan/vlans/vlan/id', type: 'number', label: 'id' },
        { path: '/vlan/vlans/vlan/no-such-leaf', type: 'string', label: 'no-such-leaf' },
      ],
    },
  ] as any[]

  it('resKeyFor：扁平路径 → 源模块前缀键', () => {
    expect(resKeyFor('huawei-vlan', '/vlan/vlans/vlan/id')).toBe('/huawei-vlan:vlan/vlans/vlan/id')
  })

  it('sourceModuleFor：左树命中优先，否则 huawei-<root> 约定回退', () => {
    const leftTree = [
      { zh: 'g', en: 'g', children: [{ zh: 'l', en: 'l', sourceModule: 'huawei-vlan', module: 'vlan' }] },
    ] as any[]
    expect(sourceModuleFor('vlan', leftTree)).toBe('huawei-vlan')
    expect(sourceModuleFor('ifm', [])).toBe('huawei-ifm')
  })

  it('zh：命中 res 换标签，缺键回退原 label（R08）', async () => {
    const out = await localizeFields(fields, 'vlan', 'zh-cn', [])
    expect(out[0].label).toBe('VLAN列表')
    expect(out[0].fields![0].label).toBe('VLAN标识')
    expect(out[0].fields![1].label).toBe('no-such-leaf')
  })

  it('en：同键取英文名', async () => {
    const out = await localizeFields(fields, 'vlan', 'en-us', [])
    expect(out[0].fields![0].label.toLowerCase()).toContain('vlan')
    expect(out[0].fields![0].label).not.toBe('VLAN标识')
  })

  it('res 文件缺失：整树原样回退不抛错', async () => {
    const out = await localizeFields(fields, 'no-such-module', 'zh-cn', [])
    expect(out[0].fields![0].label).toBe('id')
  })
})
