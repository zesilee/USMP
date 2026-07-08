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

describe('FieldRenderer · 动态缺省占位与单位后缀（FE-15）', () => {
  it('dynamicDefault 字段空值展示「系统自动分配」占位', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: { path: '/ifm/x/admin-status', type: 'string', label: 'admin-status', dynamicDefault: true },
        modelValue: undefined,
      },
      global: { plugins: [ElementPlus] },
    })
    expect(w.find('input').attributes('placeholder')).toContain('系统自动分配')
  })

  it('显式 placeholder 优先于动态缺省占位', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: { path: '/x/a', type: 'string', label: 'a', dynamicDefault: true, placeholder: '自定义' },
        modelValue: undefined,
      },
      global: { plugins: [ElementPlus] },
    })
    expect(w.find('input').attributes('placeholder')).toBe('自定义')
  })

  it('units 在 string 输入框渲染单位后缀', () => {
    const w = mount(FieldRenderer, {
      props: {
        field: { path: '/x/bw', type: 'string', label: 'bw', units: 'bit/s' },
        modelValue: '100',
      },
      global: { plugins: [ElementPlus] },
    })
    expect(w.find('.field-units').text()).toBe('bit/s')
  })

  it('units 在 number 输入框渲染单位后缀；无 units 不渲染', () => {
    const withUnits = mount(FieldRenderer, {
      props: {
        field: { path: '/x/mtu', type: 'number', label: 'mtu', units: 'octets' },
        modelValue: 1500,
      },
      global: { plugins: [ElementPlus] },
    })
    expect(withUnits.find('.field-units').text()).toBe('octets')

    const without = mount(FieldRenderer, {
      props: {
        field: { path: '/x/mtu', type: 'number', label: 'mtu' },
        modelValue: 1500,
      },
      global: { plugins: [ElementPlus] },
    })
    expect(without.find('.field-units').exists()).toBe(false)
  })
})
