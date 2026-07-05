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
