import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import FieldRenderer from '../../src/components/config/FieldRenderer.vue'
import ElementPlus from 'element-plus'

// FE-01（MODIFIED，nce-fidelity-polish）：enum 按选项数与必填性细分——
// 必填且 ≤3 选项 → el-segmented（NCE §2.2 分段控件）；可选或 >3 → el-select（可清空）。
const opts3 = [
  { label: 'Local', value: 'local' },
  { label: 'Third-party', value: 'third-party' },
  { label: 'Remote', value: 'remote' },
]
const opts4 = [...opts3, { label: 'Other', value: 'other' }]

function mountEnum(field: Record<string, unknown>, modelValue: unknown = undefined) {
  return mount(FieldRenderer, {
    props: {
      field: { path: '/x/mode', type: 'enum', label: 'mode', ...field },
      modelValue,
    },
    global: { plugins: [ElementPlus] },
  })
}

describe('FieldRenderer · enum 分段控件（FE-01 必填短枚举）', () => {
  it('必填且 ≤3 选项渲染 el-segmented，选项齐全', () => {
    const w = mountEnum({ required: true, options: opts3 })
    const seg = w.findComponent({ name: 'ElSegmented' })
    expect(seg.exists()).toBe(true)
    expect(w.findComponent({ name: 'ElSelect' }).exists()).toBe(false)
    expect(seg.props('options')).toHaveLength(3)
  })

  it('分段选中触发 update:modelValue', async () => {
    const w = mountEnum({ required: true, options: opts3 }, 'local')
    const seg = w.findComponent({ name: 'ElSegmented' })
    await seg.vm.$emit('update:modelValue', 'remote')
    expect(w.emitted('update:modelValue')?.at(-1)).toEqual(['remote'])
  })

  it('可选（required 缺省）≤3 选项仍渲染 el-select 且可清空', () => {
    const w = mountEnum({ options: opts3 })
    expect(w.findComponent({ name: 'ElSegmented' }).exists()).toBe(false)
    const sel = w.findComponent({ name: 'ElSelect' })
    expect(sel.exists()).toBe(true)
    expect(sel.props('clearable')).toBe(true)
  })

  it('必填但 >3 选项渲染 el-select（边界）', () => {
    const w = mountEnum({ required: true, options: opts4 })
    expect(w.findComponent({ name: 'ElSegmented' }).exists()).toBe(false)
    expect(w.findComponent({ name: 'ElSelect' }).exists()).toBe(true)
  })

  it('必填但零选项（异常 schema）降级 el-select，不崩（R08）', () => {
    const w = mountEnum({ required: true, options: [] })
    expect(w.findComponent({ name: 'ElSegmented' }).exists()).toBe(false)
    expect(w.findComponent({ name: 'ElSelect' }).exists()).toBe(true)
  })

  it('readonly / 外部 disabled 透传为 segmented 禁用', () => {
    const ro = mountEnum({ required: true, options: opts3, readonly: true })
    expect(ro.findComponent({ name: 'ElSegmented' }).props('disabled')).toBe(true)

    const dis = mount(FieldRenderer, {
      props: {
        field: { path: '/x/mode', type: 'enum', label: 'mode', required: true, options: opts3 },
        modelValue: 'local',
        disabled: true,
      },
      global: { plugins: [ElementPlus] },
    })
    expect(dis.findComponent({ name: 'ElSegmented' }).props('disabled')).toBe(true)
  })
})
