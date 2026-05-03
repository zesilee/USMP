import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import DynamicForm from '../../src/components/config/DynamicForm.vue'
import ElementPlus from 'element-plus'

const mockFields = [
  { path: 'name', type: 'string' as const, label: '名称', required: true },
  { path: 'description', type: 'string' as const, label: '描述' },
  { path: 'enabled', type: 'boolean' as const, label: '启用' }
]

describe('DynamicForm Component', () => {
  it('should render ElForm component', () => {
    const wrapper = mount(DynamicForm, {
      props: { fields: mockFields, modelValue: {} },
      global: { plugins: [ElementPlus] }
    })
    expect(wrapper.findComponent({ name: 'ElForm' }).exists()).toBe(true)
  })

  it('should render FieldRenderer for each field', () => {
    const wrapper = mount(DynamicForm, {
      props: { fields: mockFields, modelValue: {} },
      global: { plugins: [ElementPlus] }
    })
    const formItems = wrapper.findAllComponents({ name: 'ElFormItem' })
    expect(formItems.length).toBe(mockFields.length)
  })

  it('should expose validate method', () => {
    const wrapper = mount(DynamicForm, {
      props: { fields: mockFields, modelValue: {} },
      global: { plugins: [ElementPlus] }
    })
    expect(typeof (wrapper.vm as any).validate).toBe('function')
  })

  it('should expose resetFields method', () => {
    const wrapper = mount(DynamicForm, {
      props: { fields: mockFields, modelValue: {} },
      global: { plugins: [ElementPlus] }
    })
    expect(typeof (wrapper.vm as any).resetFields).toBe('function')
  })

  it('should expose getFormData method', () => {
    const wrapper = mount(DynamicForm, {
      props: { fields: mockFields, modelValue: { name: 'test' } },
      global: { plugins: [ElementPlus] }
    })
    expect(typeof (wrapper.vm as any).getFormData).toBe('function')
  })
})
