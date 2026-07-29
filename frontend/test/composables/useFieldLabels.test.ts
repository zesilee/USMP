import { describe, it, expect } from 'vitest'
import { localizeFields, localizeRpcs, sourceModuleFor, resKeyFor } from '../../src/composables/useFieldLabels'

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

// UI-03 rpc 扩展 F1：rpc 标签（模块顶层键，无根容器段）+ input 叶（/input/ 段）查表、
// 缺键回退原名、缺 res 整树回退、双语。真实 huawei-ifm res 副本。
describe('localizeRpcs（UI-03 扩展 rpc）', () => {
  const rpcs = [
    {
      name: 'restart-if',
      label: 'restart-if',
      highRisk: true,
      input: [{ path: 'if-name', type: 'string', label: 'if-name' }],
    },
    {
      // res 无此 rpc 键 → 整条回退原名
      name: 'no-such-rpc',
      label: 'no-such-rpc',
      input: [{ path: 'no-such-leaf', type: 'string', label: 'no-such-leaf' }],
    },
  ] as any[]

  it('zh：rpc 标签与 input 叶命中中文', async () => {
    const out = await localizeRpcs(rpcs, 'ifm', 'zh-cn', [])
    expect(out[0].label).toBe('重启接口')
    expect(out[0].input[0].label).toBe('重启接口名')
  })

  it('zh：缺键 rpc 与缺键 input 叶回退原名（R08）', async () => {
    const out = await localizeRpcs(rpcs, 'ifm', 'zh-cn', [])
    expect(out[1].label).toBe('no-such-rpc')
    expect(out[1].input[0].label).toBe('no-such-leaf')
  })

  it('en：同键取英文名', async () => {
    const out = await localizeRpcs(rpcs, 'ifm', 'en-us', [])
    expect(out[0].label.toLowerCase()).toContain('restart')
    expect(out[0].label).not.toBe('重启接口')
  })

  it('res 文件缺失：整树原样回退不抛错', async () => {
    const out = await localizeRpcs(rpcs, 'no-such-module', 'zh-cn', [])
    expect(out[0].label).toBe('restart-if')
    expect(out[0].input[0].label).toBe('if-name')
  })

  it('不改入参（返回新对象树）', async () => {
    const src = JSON.parse(JSON.stringify(rpcs))
    await localizeRpcs(src, 'ifm', 'zh-cn', [])
    expect(src[0].label).toBe('restart-if')
    expect(src[0].input[0].label).toBe('if-name')
  })
})
