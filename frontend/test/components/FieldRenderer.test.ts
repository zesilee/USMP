import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import FieldRenderer from '../../src/components/config/FieldRenderer.vue'
import ElementPlus from 'element-plus'

const baseField = {
  path: 'test',
  type: 'string' as const,
  label: '测试字段',
  placeholder: '请输入'
}

describe('FieldRenderer Component', () => {
  it('should render ElInput for string type', () => {
    const wrapper = mount(FieldRenderer, {
      props: { field: baseField, modelValue: '' },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.findComponent({ name: 'ElInput' }).exists()).toBe(true)
  })

  it('should render ElInputNumber for number type', () => {
    const wrapper = mount(FieldRenderer, {
      props: { field: { ...baseField, type: 'number' }, modelValue: 0 },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.findComponent({ name: 'ElInputNumber' }).exists()).toBe(true)
  })

  it('should render ElSwitch for boolean type', () => {
    const wrapper = mount(FieldRenderer, {
      props: { field: { ...baseField, type: 'boolean' }, modelValue: false },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.findComponent({ name: 'ElSwitch' }).exists()).toBe(true)
  })

  it('should render ElSelect for enum type', () => {
    const wrapper = mount(FieldRenderer, {
      props: {
        field: {
          ...baseField,
          type: 'enum',
          options: [{ label: '选项1', value: '1' }, { label: '选项2', value: '2' }]
        },
        modelValue: ''
      },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.findComponent({ name: 'ElSelect' }).exists()).toBe(true)
  })
})
