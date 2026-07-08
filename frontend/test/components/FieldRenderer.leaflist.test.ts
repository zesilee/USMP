import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import FieldRenderer from '../../src/components/config/FieldRenderer.vue'

const leafList = { path: '/x/tags', type: 'leaf-list' as const, label: '标签' }

describe('FieldRenderer · leaf-list 可增删多值（add/edit/remove）', () => {
  it('渲染已有数组为多行输入', () => {
    const w = mount(FieldRenderer, {
      props: { field: leafList, modelValue: ['a', 'b'] },
      global: { plugins: [ElementPlus] },
    })
    expect(w.findAll('.leaf-list-row').length).toBe(2)
  })

  it('add 追加空项', async () => {
    const w = mount(FieldRenderer, {
      props: { field: leafList, modelValue: ['a'] },
      global: { plugins: [ElementPlus] },
    })
    await w.findAll('button').at(-1)!.trigger('click') // 「添加标签」
    const emits = w.emitted('update:modelValue')!
    expect(emits.at(-1)![0]).toEqual(['a', ''])
  })

  it('edit 改某项', async () => {
    const w = mount(FieldRenderer, {
      props: { field: leafList, modelValue: ['a', 'b'] },
      global: { plugins: [ElementPlus] },
    })
    // 第二个输入框改值
    await w.findAllComponents({ name: 'ElInput' })[1].vm.$emit('update:modelValue', 'B')
    const emits = w.emitted('update:modelValue')!
    expect(emits.at(-1)![0]).toEqual(['a', 'B'])
  })

  it('remove 删某项', async () => {
    const w = mount(FieldRenderer, {
      props: { field: leafList, modelValue: ['a', 'b', 'c'] },
      global: { plugins: [ElementPlus] },
    })
    // 每行一个「删除」按钮，点第一行的
    const delButtons = w.findAll('.leaf-list-row button')
    await delButtons[0].trigger('click')
    const emits = w.emitted('update:modelValue')!
    expect(emits.at(-1)![0]).toEqual(['b', 'c'])
  })

  it('带 options 的 leaf-list 渲染为下拉', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: { ...leafList, options: [{ label: 'up', value: 'up' }, { label: 'down', value: 'down' }] },
        modelValue: ['up'],
      },
      global: { plugins: [ElementPlus] },
    })
    expect(w.findComponent({ name: 'ElSelect' }).exists()).toBe(true)
  })
})
