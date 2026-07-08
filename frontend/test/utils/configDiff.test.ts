import { describe, it, expect } from 'vitest'
import { computeDiff, missingRequired } from '../../src/utils/configDiff'
import type { Field } from '../../src/utils/crdSchemaParser'

const fields: Field[] = [
  { path: '/vlan/vlans/vlan/id', type: 'number', label: 'VLAN ID', required: true },
  { path: '/vlan/vlans/vlan/name', type: 'string', label: '名称' },
  { path: '/vlan/vlans/vlan/admin-status', type: 'enum', label: '状态', options: [{ label: 'up', value: 'up' }] },
]

describe('computeDiff · 实时差异（表单值 ↔ 已回填 actual）', () => {
  it('仅列出发生变化且非空的字段', () => {
    const original = { id: 100, name: 'old', 'admin-status': 'up' }
    const form = { id: 100, name: 'new', 'admin-status': 'up' }
    const diff = computeDiff(form, original, fields)
    expect(diff.map((d) => d.key)).toEqual(['name'])
    expect(diff[0]).toMatchObject({ key: 'name', label: '名称', was: 'old', now: 'new', isNew: false })
  })

  it('原值为空 → 标记为新增（isNew）', () => {
    const original = { id: '', name: '', 'admin-status': '' }
    const form = { id: 200, name: 'guest', 'admin-status': 'up' }
    const diff = computeDiff(form, original, fields)
    expect(diff.map((d) => d.key).sort()).toEqual(['admin-status', 'id', 'name'])
    expect(diff.every((d) => d.isNew)).toBe(true)
  })

  it('新值为空串不算改动（不能用清空来“下发删除”，与原型一致）', () => {
    const original = { id: 100, name: 'keep' }
    const form = { id: 100, name: '' }
    expect(computeDiff(form, original, fields)).toEqual([])
  })

  it('数值与字符串比较按字符串归一（100 === "100" 视为无改动）', () => {
    const original = { id: 100 }
    const form = { id: '100' }
    expect(computeDiff(form, original, fields)).toEqual([])
  })

  it('保持 fields 声明顺序输出', () => {
    const original = {}
    const form = { name: 'n', id: 5, 'admin-status': 'up' }
    const diff = computeDiff(form, original, fields)
    expect(diff.map((d) => d.key)).toEqual(['id', 'name', 'admin-status'])
  })

  it('数值 0 是合法值：算作改动、不被当空（防 !v 判空回归）', () => {
    const original = { id: 100, name: 'x' }
    const numFields: Field[] = [{ path: '/x/mtu', type: 'number', label: 'MTU' }, ...fields]
    const diff = computeDiff({ mtu: 0 }, {}, numFields)
    expect(diff.find((d) => d.key === 'mtu')).toMatchObject({ now: 0, isNew: true })
  })

  it('空/异常输入安全降级（R08）', () => {
    expect(computeDiff(null as any, null as any, fields)).toEqual([])
    expect(computeDiff({}, {}, [])).toEqual([])
  })
})

describe('missingRequired · 数值 0 视为已填', () => {
  it('required 数值字段填 0 不算缺失', () => {
    const numFields: Field[] = [{ path: '/x/mtu', type: 'number', label: 'MTU', required: true }]
    expect(missingRequired(numFields, { mtu: 0 }, 'name')).toEqual([])
  })
})

describe('missingRequired · 必填未填校验（下发按钮启用前置）', () => {
  it('列出缺失的必填字段 label', () => {
    expect(missingRequired(fields, { id: '', name: 'x' }, 'id')).toEqual(['VLAN ID'])
  })

  it('keyField 视为必填（即使 schema 未标 required）', () => {
    const noReq: Field[] = [{ path: '/x/name', type: 'string', label: '名称' }]
    expect(missingRequired(noReq, { name: '' }, 'name')).toEqual(['名称'])
  })

  it('全部必填已填 → 空数组', () => {
    expect(missingRequired(fields, { id: 10, name: 'x' }, 'id')).toEqual([])
  })
})

describe('missingRequired · dynamicDefault 豁免（FE-15）', () => {
  const f = (name: string, extra: any = {}) => ({
    path: `/x/${name}`, type: 'string' as const, label: name, ...extra,
  })

  it('required + dynamicDefault 空值不算缺失（系统自动分配）', () => {
    const fields = [f('admin-status', { required: true, dynamicDefault: true })]
    expect(missingRequired(fields, {}, '')).toEqual([])
  })

  it('required 无 dynamicDefault 空值仍算缺失', () => {
    const fields = [f('name', { required: true })]
    expect(missingRequired(fields, {}, '')).toEqual(['name'])
  })

  it('keyField 即使 dynamicDefault 也恒必填（键不可缺）', () => {
    const fields = [f('id', { dynamicDefault: true })]
    expect(missingRequired(fields, {}, 'id')).toEqual(['id'])
  })
})
