import { describe, it, expect } from 'vitest'
import { ref } from 'vue'
import { useConfigForm } from '../../src/composables/useConfigForm'
import type { Field } from '../../src/utils/crdSchemaParser'

// FE-14 深层排除（NS-08/BR-01 回归）：读路径带回 config=false 状态后，
// 可写 group/嵌套 list 内的 readonly 子叶不得随组对象进下发 payload——
// Encode 是 populated-means-pushed，state 叶下发真机会被拒绝。

const fields = ref<Field[]>([
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
] as Field[])

describe('useConfigForm · payload 深层排除 readonly 状态叶（FE-14）', () => {
  it('可写 group 内的 readonly 子叶（含嵌套）不入 payload，可写叶保留', () => {
    const form = useConfigForm(fields)
    form.resetForm({
      name: 'a',
      tuning: {
        level: 'high',
        'oper-state': 'up',
        inner: { knob: 'k1', counter: '42' },
      },
    })
    const p = form.visiblePayload()
    expect(p.name).toBe('a')
    expect(p.tuning.level).toBe('high')
    expect(p.tuning['oper-state'], 'group 内 readonly 叶不得下发').toBeUndefined()
    expect(p.tuning.inner.knob).toBe('k1')
    expect(p.tuning.inner.counter, '嵌套 group 内 readonly 叶不得下发').toBeUndefined()
  })

  it('嵌套 list 行内的 readonly 叶不入 payload', () => {
    const form = useConfigForm(fields)
    form.resetForm({
      name: 'a',
      members: [
        { id: 'm1', state: 'active' },
        { id: 'm2', state: 'down' },
      ],
    })
    const p = form.visiblePayload()
    expect(p.members).toHaveLength(2)
    expect(p.members[0].id).toBe('m1')
    expect(p.members[0].state, 'list 行内 readonly 叶不得下发').toBeUndefined()
    expect(p.members[1].state).toBeUndefined()
  })

  it('整组 readonly（config false 容器）整体不入 payload（既有 FE-14 行为不回退）', () => {
    const form = useConfigForm(fields)
    form.resetForm({ name: 'a', dynamic: { mac: '00:11' } })
    const p = form.visiblePayload()
    expect(p.dynamic).toBeUndefined()
    expect(p.name).toBe('a')
  })
})
