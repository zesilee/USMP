import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import FieldRenderer from '../../src/components/config/FieldRenderer.vue'
import type { Field } from '../../src/utils/crdSchemaParser'

// bandwidth-type: 两个单叶 case（mbps / kbps）→ 互斥 RadioGroup。成员 path 扁平。
const bwChoice: Field = {
  path: '/ifm/interfaces/interface/bandwidth-type',
  type: 'choice',
  label: 'bandwidth-type',
  cases: [
    {
      name: 'bandwidth-mbps',
      label: 'bandwidth-mbps',
      fields: [{ path: '/ifm/interfaces/interface/bandwidth', type: 'number', label: 'bandwidth' }],
    },
    {
      name: 'bandwidth-kbps',
      label: 'bandwidth-kbps',
      fields: [{ path: '/ifm/interfaces/interface/bandwidth-kbps', type: 'number', label: 'bandwidth-kbps' }],
    },
  ],
}

// 一个含多字段 case 的 choice → 应渲染为 Tabs（而非 Radio）。
const tabsChoice: Field = {
  path: '/x/mode',
  type: 'choice',
  label: 'mode',
  cases: [
    {
      name: 'manual',
      label: 'manual',
      fields: [
        { path: '/x/a', type: 'number', label: 'a' },
        { path: '/x/b', type: 'number', label: 'b' },
      ],
    },
    { name: 'auto', label: 'auto', fields: [{ path: '/x/c', type: 'number', label: 'c' }] },
  ],
}

const mountChoice = (field: Field, modelValue: Record<string, any> = {}) =>
  mount(FieldRenderer, { props: { field, modelValue }, global: { plugins: [ElementPlus] } })

describe('FieldRenderer · choice 互斥分支（Tabs/RadioGroup）', () => {
  it('全单叶 case → 渲染 RadioGroup（非 Tabs）', () => {
    const w = mountChoice(bwChoice)
    expect(w.findComponent({ name: 'ElRadioGroup' }).exists()).toBe(true)
    expect(w.findComponent({ name: 'ElTabs' }).exists()).toBe(false)
  })

  it('任一 case 含多字段 → 渲染 Tabs', () => {
    const w = mountChoice(tabsChoice)
    expect(w.findComponent({ name: 'ElTabs' }).exists()).toBe(true)
    expect(w.findComponent({ name: 'ElRadioGroup' }).exists()).toBe(false)
  })

  it('激活 case 由数据推断：kbps 有值 → 展示 kbps 分支输入', () => {
    const w = mountChoice(bwChoice, { 'bandwidth-kbps': 64 })
    expect(w.findComponent({ name: 'ElRadioGroup' }).props('modelValue')).toBe('bandwidth-kbps')
  })

  it('编辑激活成员 → emit 扁平成员键（sibling scope）', async () => {
    const w = mountChoice(bwChoice) // 默认激活首个 case bandwidth-mbps
    await w.findComponent({ name: 'ElInputNumber' }).vm.$emit('update:modelValue', 1000)
    const emits = w.emitted('update:modelValue')!
    expect(emits.at(-1)![0]).toEqual({ bandwidth: 1000 })
  })

  it('切换 case → 清空非激活分支成员键', async () => {
    const w = mountChoice(bwChoice, { bandwidth: 1000 }) // 先激活 mbps 且有值
    await w.findComponent({ name: 'ElRadioGroup' }).vm.$emit('update:modelValue', 'bandwidth-kbps')
    const emits = w.emitted('update:modelValue')!
    // 切到 kbps：bandwidth（mbps 成员）被清除，scope 不再含它
    expect(emits.at(-1)![0]).toEqual({})
  })
})
