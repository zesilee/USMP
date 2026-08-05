import { describe, it, expect } from 'vitest'
import { mergeReadonlyState } from '../../src/utils/configState'
import type { Field } from '../../src/utils/crdSchemaParser'

// 读通道拆分（真机回归）：列表读改走 config-only 快通道后，行数据不再携带
// config=false 状态；详情打开时按需单行读状态，只把「只读字段」的值合并进
// 表单展示——绝不覆盖用户可编辑字段（未保存的草稿必须原样保留）。
const fields: Field[] = [
  { path: '/i/name', type: 'string', label: 'name', isKey: true },
  { path: '/i/description', type: 'string', label: 'description' },
  { path: '/i/dynamic', type: 'group', label: 'dynamic', readonly: true, fields: [
    { path: '/i/dynamic/mac-address', type: 'string', label: 'mac-address', readonly: true },
  ] },
  { path: '/i/statistics-cfg', type: 'group', label: 'statistics-cfg', fields: [
    { path: '/i/statistics-cfg/interval', type: 'number', label: 'interval' },
    { path: '/i/statistics-cfg/oper', type: 'string', label: 'oper', readonly: true },
  ] },
] as Field[]

describe('mergeReadonlyState', () => {
  it('只读组整体合入；可编辑字段（含用户草稿）不被覆盖', () => {
    const formData: Record<string, any> = { name: 'GE0/0/1', description: 'draft-edit' }
    mergeReadonlyState(fields, formData, {
      name: 'GE0/0/1',
      description: 'device-value',
      dynamic: { 'mac-address': '00:aa:bb:cc:dd:ee' },
    })
    expect(formData['dynamic']).toEqual({ 'mac-address': '00:aa:bb:cc:dd:ee' })
    expect(formData['description'], '用户草稿不得被状态合并覆盖').toBe('draft-edit')
    expect(formData['name']).toBe('GE0/0/1')
  })

  it('可写组内的只读子叶深合并，同组可写子叶保留', () => {
    const formData: Record<string, any> = { 'statistics-cfg': { interval: 60 } }
    mergeReadonlyState(fields, formData, { 'statistics-cfg': { interval: 30, oper: 'on' } })
    expect(formData['statistics-cfg']).toEqual({ interval: 60, oper: 'on' })
  })

  it('状态数据缺失/为空 → 表单原样（不构造空占位）', () => {
    const formData: Record<string, any> = { name: 'x' }
    mergeReadonlyState(fields, formData, undefined)
    mergeReadonlyState(fields, formData, {})
    expect(formData).toEqual({ name: 'x' })
  })
})
