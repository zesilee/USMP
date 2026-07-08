import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import FieldRenderer from '../../src/components/config/FieldRenderer.vue'
import type { Field } from '../../src/utils/crdSchemaParser'

// 真 Chromium 验证 choice 互斥控件的真实渲染与交互（happy-dom 对 el-tabs 激活态/
// el-radio 真实点击不可靠）。语料取自真实 IFM bandwidth-type（两单叶 case）与 tabs 形态。
const bwChoice: Field = {
  path: '/ifm/interfaces/interface/bandwidth-type',
  type: 'choice',
  label: 'bandwidth-type',
  cases: [
    { name: 'bandwidth-mbps', label: 'bandwidth-mbps', fields: [{ path: '/ifm/interfaces/interface/bandwidth', type: 'number', label: 'bandwidth' }] },
    { name: 'bandwidth-kbps', label: 'bandwidth-kbps', fields: [{ path: '/ifm/interfaces/interface/bandwidth-kbps', type: 'number', label: 'bandwidth-kbps' }] },
  ],
}

const tabsChoice: Field = {
  path: '/x/mode',
  type: 'choice',
  label: 'mode',
  cases: [
    { name: 'manual', label: 'manual', fields: [{ path: '/x/a', type: 'number', label: 'a' }, { path: '/x/b', type: 'number', label: 'b' }] },
    { name: 'auto', label: 'auto', fields: [{ path: '/x/c', type: 'number', label: 'c' }] },
  ],
}

describe('FieldRenderer choice（真浏览器）', () => {
  it('全单叶 case 渲染为真实 RadioGroup，点击切换 case 清空旧分支成员', async () => {
    const w = mount(FieldRenderer, {
      global: { plugins: [ElementPlus] },
      props: { field: bwChoice, modelValue: { bandwidth: 1000 } },
      attachTo: document.body,
    })
    await w.vm.$nextTick()

    // 两个真实 radio 落地
    const radios = document.querySelectorAll('.el-radio')
    expect(radios.length).toBe(2)
    expect(document.body.textContent).toContain('bandwidth-mbps')
    expect(document.body.textContent).toContain('bandwidth-kbps')

    // 真实点击第二个 radio（bandwidth-kbps）→ 切 case → 清空 bandwidth（mbps 成员）
    ;(radios[1] as HTMLElement).click()
    await w.vm.$nextTick()
    const emitted = w.emitted('update:modelValue')!
    expect(emitted.at(-1)![0]).toEqual({})

    w.unmount()
  })

  it('含多字段 case 渲染为真实 Tabs，点击切换 tab', async () => {
    const w = mount(FieldRenderer, {
      global: { plugins: [ElementPlus] },
      props: { field: tabsChoice, modelValue: {} },
      attachTo: document.body,
    })
    await w.vm.$nextTick()

    // 真实 el-tabs 头渲染两个 case 名
    const tabItems = document.querySelectorAll('.el-tabs__item')
    expect(tabItems.length).toBe(2)
    const labels = Array.from(tabItems).map((n) => (n.textContent || '').trim())
    expect(labels).toContain('manual')
    expect(labels).toContain('auto')

    // 点击 auto tab → 激活切换（emit 记录切换）
    const autoTab = Array.from(tabItems).find((n) => (n.textContent || '').includes('auto')) as HTMLElement
    autoTab.click()
    await w.vm.$nextTick()
    // 切到 auto：manual 分支成员（a/b）被清空 → scope 仍为空对象
    const emitted = w.emitted('update:modelValue')
    if (emitted) expect(emitted.at(-1)![0]).toEqual({})

    w.unmount()
  })
})
