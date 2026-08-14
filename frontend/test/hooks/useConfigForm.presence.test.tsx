import { describe, it, expect } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useConfigForm } from '../../src/hooks/useConfigForm'
import type { Field } from '../../src/utils/crdSchemaParser'

// FE-27（spec ADDED）：表单键存在性即节点存在性。React 不可变更新最易踩的暗雷
// 是 `{...prev, [k]: undefined}` ——键仍存在（`in`/Object.keys 可见），presence
// 关闭/choice 切分支/dynamicDefault 留空的「节点不存在」语义被静默破坏，下发
// 多余字段且不报错（真机 unknown-element 拒绝整次配置）。本套件以**键枚举**
// 判定（非取值判定），红灯先行（T05/T07 同源精神）。

const fields: Field[] = [
  { path: '/m/name', type: 'string', label: 'name' },
  {
    path: '/m/tuning',
    type: 'group',
    label: 'tuning',
    presence: true,
    fields: [{ path: '/m/tuning/level', type: 'string', label: 'level' }],
  },
  {
    path: '/m/mode',
    type: 'choice',
    label: 'mode',
    cases: [
      { name: 'a', label: 'a', fields: [{ path: '/m/speed', type: 'number', label: 'speed' }] },
      { name: 'b', label: 'b', fields: [{ path: '/m/auto', type: 'boolean', label: 'auto' }] },
    ],
  },
  { path: '/m/vid', type: 'number', label: 'vid', dynamicDefault: true },
]

function mount() {
  return renderHook(() => useConfigForm(fields))
}

describe('FE-27 · 键存在性即节点存在性（键枚举判定）', () => {
  it('presence 容器关闭：键从 formData 真正消失，且不入 payload', () => {
    const { result } = mount()
    act(() => result.current.setField('tuning', { level: 'high' }))
    expect(Object.keys(result.current.formData)).toContain('tuning')

    // 关闭 presence（FieldRenderer 语义：emit undefined = 节点不存在）
    act(() => result.current.setField('tuning', undefined))
    expect(Object.keys(result.current.formData)).not.toContain('tuning')
    expect('tuning' in result.current.formData).toBe(false)
    expect(Object.keys(result.current.visiblePayload())).not.toContain('tuning')
  })

  it('choice 切分支：非激活 case 成员键整体移除，payload 只含激活分支', () => {
    const { result } = mount()
    act(() => result.current.onChoiceUpdate(fields[2], { speed: 100 }))
    expect('speed' in result.current.formData).toBe(true)

    // 切到 case b：a 的成员键必须消失（FieldRenderer switchCase 传入不含 speed 的 next）
    act(() => result.current.onChoiceUpdate(fields[2], { auto: true }))
    expect('speed' in result.current.formData).toBe(false)
    const payload = result.current.visiblePayload()
    expect('speed' in payload).toBe(false)
    expect(payload.auto).toBe(true)
  })

  it('dynamicDefault 叶清空：键不入 payload，不以空串/null 下发覆盖设备缺省（负路径）', () => {
    const { result } = mount()
    act(() => result.current.setField('vid', 100))
    expect(result.current.visiblePayload().vid).toBe(100)

    act(() => result.current.setField('vid', ''))
    expect(Object.keys(result.current.visiblePayload())).not.toContain('vid')
    act(() => result.current.setField('vid', null))
    expect(Object.keys(result.current.visiblePayload())).not.toContain('vid')
  })

  it('setField(undefined) 对任意字段等价删键（通用语义，非 presence 专属）', () => {
    const { result } = mount()
    act(() => result.current.setField('name', 'ge0'))
    act(() => result.current.setField('name', undefined))
    expect('name' in result.current.formData).toBe(false)
  })

  it('clearedKeys（FE-22 removals）：基线有值被清 → 键入 cleared 清单', () => {
    const { result } = renderHook(() => useConfigForm(fields, '', { removals: true }))
    act(() => result.current.resetForm({ name: 'ge0', vid: 5 }))
    act(() => result.current.setField('name', ''))
    expect(result.current.clearedKeys).toContain('name')
  })

  it('resetForm 种子后再删键：original 基线不受 formData 删键影响', () => {
    const { result } = mount()
    act(() => result.current.resetForm({ name: 'ge0', vid: 5 }))
    act(() => result.current.setField('name', undefined))
    expect('name' in result.current.formData).toBe(false)
    expect(result.current.original.name).toBe('ge0') // 基线独立快照
  })
})
