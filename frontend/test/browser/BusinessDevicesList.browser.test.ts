import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import { ref } from 'vue'
import FieldRenderer from '../../src/components/config/FieldRenderer.vue'
import type { Field } from '../../src/utils/crdSchemaParser'

// F3（矩阵 A2 前端面 / FE-17）——真 Chromium：业务意图 devices 嵌套 list 的
// 增/改/删全流程。嵌套 list 行内子表单 + leaf-list 是 happy-dom 渲染近似最重的
// 区域，必须真浏览器兜底（§5.6 F3 军规）。

const devicesField: Field = {
  path: '/business-vlan-service/devices',
  type: 'list',
  label: '设备清单',
  fields: [
    { path: '/business-vlan-service/devices/ip', type: 'string', label: '设备 IP', required: true, isKey: true },
    { path: '/business-vlan-service/devices/access-ports', type: 'leaf-list', label: 'Access 口' },
    { path: '/business-vlan-service/devices/trunk-ports', type: 'leaf-list', label: 'Trunk 口' },
  ],
} as unknown as Field

function mountList(initial: any[] = []) {
  // 直挂 FieldRenderer（浏览器构建无运行时模板编译器，不能用内联 template），
  // onUpdate 回写 props 模拟 v-model。
  const model = ref<any[]>(initial)
  const wrapper: any = mount(FieldRenderer as any, {
    props: {
      field: devicesField,
      modelValue: model.value,
      'onUpdate:modelValue': async (v: any) => {
        model.value = v
        await wrapper.setProps({ modelValue: v })
      },
    },
    global: { plugins: [ElementPlus] },
    attachTo: document.body,
  })
  return { wrapper, model }
}

describe('业务 devices 嵌套 list（真浏览器，FE-17）', () => {
  it('add：添加两行并填 IP，模型收到两条设备', async () => {
    const { wrapper, model } = mountList()

    const addBtn = wrapper.findAll('button').find((b) => b.text().includes('添加'))
    expect(addBtn).toBeTruthy()
    await addBtn!.trigger('click')
    await addBtn!.trigger('click')
    await vi_waitRows(2)

    const ipInputs = ipFields()
    await setInput(ipInputs[0], '10.0.0.1')
    await setInput(ipInputs[1], '10.0.0.2')

    expect(model.value).toHaveLength(2)
    expect(model.value[0].ip).toBe('10.0.0.1')
    expect(model.value[1].ip).toBe('10.0.0.2')
    wrapper.unmount()
  })

  it('edit：既有行修改 IP 即时写回模型', async () => {
    const { wrapper, model } = mountList([{ ip: '10.0.0.1', 'access-ports': ['GE0/0/1'] }])
    await vi_waitRows(1)

    const ipInput = ipFields()[0]
    expect((ipInput as HTMLInputElement).value).toBe('10.0.0.1')
    await setInput(ipInput, '10.0.0.9')
    expect(model.value[0].ip).toBe('10.0.0.9')
    // leaf-list 既有值真实渲染（值在 input.value 里，不在 textContent）。
    const hasPort = Array.from(document.body.querySelectorAll('input')).some(
      (i) => (i as HTMLInputElement).value === 'GE0/0/1',
    )
    expect(hasPort).toBe(true)
    wrapper.unmount()
  })

  it('remove：删除行后模型收缩且 DOM 收起', async () => {
    const { wrapper, model } = mountList([{ ip: '10.0.0.1' }, { ip: '10.0.0.2' }])
    await vi_waitRows(2)

    const removeBtns = wrapper.findAll('button').filter((b) => b.text().includes('删除'))
    expect(removeBtns.length).toBeGreaterThanOrEqual(2)
    await removeBtns[0].trigger('click')

    await vi.waitFor(() => {
      expect(model.value).toHaveLength(1)
      expect(ipFields()).toHaveLength(1)
    }, { timeout: 3000 })
    expect(model.value[0].ip).toBe('10.0.0.2')
    wrapper.unmount()
  })
})

// 每个 list 行里「设备 IP」子字段的 input（嵌套 leaf-list 的 input 不计入）。
function ipFields(): HTMLInputElement[] {
  const rows = Array.from(document.body.querySelectorAll('.list-row:not(.leaf-list-row)'))
  return rows
    .map((row) => {
      const sub = Array.from(row.querySelectorAll('.sub-field')).find((s) =>
        s.querySelector('.field-label')?.textContent?.includes('设备 IP'),
      )
      return sub?.querySelector('input') as HTMLInputElement
    })
    .filter(Boolean) as HTMLInputElement[]
}

async function setInput(el: HTMLInputElement, value: string) {
  el.value = value
  el.dispatchEvent(new Event('input', { bubbles: true }))
  await Promise.resolve()
}

async function vi_waitRows(n: number) {
  await vi.waitFor(() => {
    expect(ipFields().length).toBeGreaterThanOrEqual(n)
  }, { timeout: 3000 })
}
