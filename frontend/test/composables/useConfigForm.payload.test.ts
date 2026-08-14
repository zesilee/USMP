import { describe, it, expect } from 'vitest'
import { visiblePayload, changedPayload, snapshotBaseline } from '../../src/form/configForm'
import type { Field } from '../../src/utils/crdSchemaParser'

// FE-14 深层排除（NS-08/BR-01 回归）：读路径带回 config=false 状态后，
// 可写 group/嵌套 list 内的 readonly 子叶不得随组对象进下发 payload——
// Encode 是 populated-means-pushed，state 叶下发真机会被拒绝。

const fields: Field[] = [
  { path: '/x/name', type: 'string', label: 'name' },
  {
    path: '/x/tuning',
    type: 'group',
    label: 'tuning',
    fields: [
      { path: '/x/tuning/level', type: 'string', label: 'level' },
      { path: '/x/tuning/oper-state', type: 'string', label: 'oper-state', readonly: true },
      {
        path: '/x/tuning/inner',
        type: 'group',
        label: 'inner',
        fields: [
          { path: '/x/tuning/inner/knob', type: 'string', label: 'knob' },
          { path: '/x/tuning/inner/counter', type: 'string', label: 'counter', readonly: true },
        ],
      },
    ],
  },
  {
    path: '/x/members',
    type: 'list',
    label: 'members',
    fields: [
      { path: '/x/members/member/id', type: 'string', label: 'id', isKey: true },
      { path: '/x/members/member/state', type: 'string', label: 'state', readonly: true },
    ],
  },
  {
    path: '/x/dynamic',
    type: 'group',
    label: 'dynamic',
    readonly: true,
    fields: [{ path: '/x/dynamic/mac', type: 'string', label: 'mac', readonly: true }],
  },
] as Field[]

describe('useConfigForm · payload 深层排除 readonly 状态叶（FE-14）', () => {
  it('可写 group 内的 readonly 子叶（含嵌套）不入 payload，可写叶保留', () => {
    const p = visiblePayload(fields, {
      name: 'a',
      tuning: {
        level: 'high',
        'oper-state': 'up',
        inner: { knob: 'k1', counter: '42' },
      },
    })
    expect(p.name).toBe('a')
    expect(p.tuning.level).toBe('high')
    expect(p.tuning['oper-state'], 'group 内 readonly 叶不得下发').toBeUndefined()
    expect(p.tuning.inner.knob).toBe('k1')
    expect(p.tuning.inner.counter, '嵌套 group 内 readonly 叶不得下发').toBeUndefined()
  })

  it('嵌套 list 行内的 readonly 叶不入 payload', () => {
    const p = visiblePayload(fields, {
      name: 'a',
      members: [
        { id: 'm1', state: 'active' },
        { id: 'm2', state: 'down' },
      ],
    })
    expect(p.members).toHaveLength(2)
    expect(p.members[0].id).toBe('m1')
    expect(p.members[0].state, 'list 行内 readonly 叶不得下发').toBeUndefined()
    expect(p.members[1].state).toBeUndefined()
  })

  it('整组 readonly（config false 容器）整体不入 payload（既有 FE-14 行为不回退）', () => {
    const p = visiblePayload(fields, { name: 'a', dynamic: { mac: '00:11' } })
    expect(p.dynamic).toBeUndefined()
    expect(p.name).toBe('a')
  })
})

// 真机回归（T07，unknown-element 拒绝）：编辑态表单被设备回读整行填满后，
// 「确定」的下发载荷只能带「主键 + 用户改过的字段」。设备按接口类型裁剪
// 叶能力（statistic-mode 等），把回读值原样回推会被 rpc-error unknown-element
// 拒绝（2026-08-04 真机创建接口即此因）。
describe('useConfigForm · changedPayload 只含主键与改动字段', () => {
  const kf = 'name'
  const editFields: Field[] = [
    { path: '/ifm/interfaces/interface/name', type: 'string', label: 'name', isKey: true },
    { path: '/ifm/interfaces/interface/description', type: 'string', label: 'description' },
    { path: '/ifm/interfaces/interface/statistic-mode', type: 'string', label: 'statistic-mode' },
    { path: '/ifm/interfaces/interface/admin-status', type: 'string', label: 'admin-status' },
    { path: '/ifm/interfaces/interface/auto-name', type: 'string', label: 'auto-name', dynamicDefault: true },
    {
      path: '/ifm/interfaces/interface/statistics-cfg',
      type: 'group',
      label: 'statistics-cfg',
      fields: [
        { path: '/ifm/interfaces/interface/statistics-cfg/interval', type: 'number', label: 'interval' },
        { path: '/ifm/interfaces/interface/statistics-cfg/oper', type: 'string', label: 'oper', readonly: true },
      ],
    },
  ] as Field[]
  const seed = () => ({
    name: 'GE0/0/1',
    description: 'uplink',
    'statistic-mode': 'interface-based',
    'admin-status': 'up',
    'statistics-cfg': { interval: 30, oper: 'on' },
  })

  it('编辑态：未改字段（statistic-mode 等回读值）不入载荷，主键+改动字段保留', () => {
    const original = snapshotBaseline(seed())
    const formData: Record<string, any> = { ...seed(), description: 'core-link' }
    const p = changedPayload(editFields, formData, original, kf)
    expect(p['name'], '主键恒入载荷').toBe('GE0/0/1')
    expect(p['description']).toBe('core-link')
    expect('statistic-mode' in p, '未改动的回读字段不得回推设备').toBe(false)
    expect('admin-status' in p).toBe(false)
    expect('statistics-cfg' in p).toBe(false)
  })

  it('嵌套 group 内改动可被识别（原位修改，深快照基线）', () => {
    const original = snapshotBaseline(seed())
    const formData: Record<string, any> = seed()
    formData['statistics-cfg'].interval = 60 // 原位改嵌套：深快照基线仍能识别差异
    const p = changedPayload(editFields, formData, original, kf)
    expect(p['statistics-cfg']?.interval, '嵌套改动须入载荷').toBe(60)
    expect(p['statistics-cfg']?.oper, 'readonly 子叶仍被剥除').toBeUndefined()
    expect('statistic-mode' in p).toBe(false)
  })

  it('创建态（空基线）：全部已填字段入载荷（与既有创建行为一致）', () => {
    const p = changedPayload(editFields, { name: 'GE9/9/9', description: 'new' }, {}, kf)
    expect(p['name']).toBe('GE9/9/9')
    expect(p['description']).toBe('new')
  })

  it('dynamicDefault 叶空值仍不入载荷（FE-15 语义保持）', () => {
    const original = snapshotBaseline(seed())
    const formData: Record<string, any> = { ...seed(), description: 'x', 'auto-name': '' }
    const p = changedPayload(editFields, formData, original, kf)
    expect('auto-name' in p).toBe(false)
  })
})
