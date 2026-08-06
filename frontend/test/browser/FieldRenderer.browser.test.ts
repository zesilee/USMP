import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import FieldRenderer from '../../src/components/config/FieldRenderer.vue'
import type { Field } from '../../src/utils/crdSchemaParser'

// 真 Chromium 验证：嵌套 list 字段（VLAN member-ports）渲染成可重复的子表单行 +
// 枚举下拉真实落地。这是本次 VLAN 交付新增的核心渲染能力。
const memberPortList: Field = {
  path: '/vlan/vlans/vlan/member-ports/member-port',
  type: 'list',
  label: '端口成员',
  fields: [
    { path: '/vlan/vlans/vlan/member-ports/member-port/interface-name', type: 'string', label: 'interface-name' },
    {
      path: '/vlan/vlans/vlan/member-ports/member-port/access-type',
      type: 'enum',
      label: 'access-type',
      options: [
        { label: 'access', value: 'access' },
        { label: 'trunk', value: 'trunk' },
      ],
    },
  ],
}

describe('FieldRenderer 嵌套 list（真浏览器）', () => {
  it('应把 list 渲染成可重复子表单行，含枚举下拉与添加按钮', async () => {
    const wrapper = mount(FieldRenderer, {
      global: { plugins: [ElementPlus] },
      props: { field: memberPortList, modelValue: [{ 'interface-name': 'GE0/0/1', 'access-type': 'trunk' }] },
      attachTo: document.body,
    })
    await wrapper.vm.$nextTick()

    // 一行子表单已渲染：interface-name 输入 + access-type 下拉真实落地
    expect(wrapper.findAllComponents({ name: 'ElSelect' }).length).toBeGreaterThanOrEqual(1)
    expect(wrapper.findAllComponents({ name: 'ElInput' }).length).toBeGreaterThanOrEqual(1)
    // 「添加端口成员」按钮存在
    expect(document.body.textContent).toContain('添加端口成员')

    wrapper.unmount()
  })

  it('点击添加应新增一行（emit 更新后的数组）', async () => {
    const wrapper = mount(FieldRenderer, {
      global: { plugins: [ElementPlus] },
      props: { field: memberPortList, modelValue: [] },
      attachTo: document.body,
    })
    await wrapper.vm.$nextTick()

    const addBtn = wrapper.findAllComponents({ name: 'ElButton' }).find((b) => b.text().includes('添加'))
    expect(addBtn).toBeTruthy()
    await addBtn!.trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect(emitted![emitted!.length - 1][0]).toEqual([{}])

    wrapper.unmount()
  })
})

// 真 Chromium 验证：leafref 输入禁自由文本（FE-19）。契约靠 el-select 无 allow-create
// 兜底——此用例锁死该行为：即使 options 为空，键入任意文本+回车/失焦也不得产生值。
describe('FieldRenderer leafref 禁自由文本（真浏览器）', () => {
  const leafrefField: Field = {
    path: 'if-name',
    type: 'string',
    label: 'if-name',
    leafRef: '/ifm:ifm/ifm:interfaces/ifm:interface/ifm:name',
    options: [],
  }

  it('空 options 的 leafref 下拉：键入文本+回车/失焦不 emit 任何值', async () => {
    const wrapper = mount(FieldRenderer, {
      global: { plugins: [ElementPlus] },
      props: { field: leafrefField, modelValue: '' },
      attachTo: document.body,
    })
    await wrapper.vm.$nextTick()

    const select = wrapper.find('[data-test="leafref-select"]')
    expect(select.exists()).toBe(true)
    // 无普通文本框兜底
    expect(wrapper.find('.field-scalar').exists()).toBe(false)

    const input = select.find('input')
    await input.trigger('click')
    await input.setValue('200GE-fake/0/0')
    await input.trigger('keydown', { key: 'Enter' })
    await input.trigger('blur')
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('update:modelValue')).toBeFalsy()

    wrapper.unmount()
  })

  it('有 options 时仍只能选既有项（filterable 过滤不产生新值）', async () => {
    const wrapper = mount(FieldRenderer, {
      global: { plugins: [ElementPlus] },
      props: {
        field: { ...leafrefField, options: [{ label: '200GE0/1/0', value: '200GE0/1/0' }] },
        modelValue: '',
      },
      attachTo: document.body,
    })
    await wrapper.vm.$nextTick()

    const input = wrapper.find('[data-test="leafref-select"] input')
    await input.trigger('click')
    await input.setValue('no-such-interface')
    await input.trigger('keydown', { key: 'Enter' })
    await wrapper.vm.$nextTick()

    // 过滤无命中 → 回车不选中任何值
    expect(wrapper.emitted('update:modelValue')).toBeFalsy()

    wrapper.unmount()
  })
})
