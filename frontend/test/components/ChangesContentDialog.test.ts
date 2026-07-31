import { describe, it, expect, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import ChangesContentDialog from '../../src/components/config/ChangesContentDialog.vue'
import { useChangesetStore } from '../../src/stores/changeset'

const DEV = '10.0.0.1'
const VLAN_PATH = '/vlan:vlan/vlan:vlans'

function mountDialog() {
  return mount(ChangesContentDialog, {
    props: { visible: true, device: DEV },
    global: { plugins: [ElementPlus] },
    attachTo: document.body,
  })
}

beforeEach(() => setActivePinia(createPinia()))

function seedThree() {
  const s = useChangesetStore()
  s.upsert(DEV, {
    op: 'create',
    path: VLAN_PATH,
    listKey: 'vlan',
    keyValue: '20',
    payload: { id: 20, name: 'newbie' },
    cleared: [],
    baseline: null,
    label: 'vlan 20',
  })
  s.upsert(DEV, {
    op: 'update',
    path: VLAN_PATH,
    listKey: 'vlan',
    keyValue: '10',
    payload: { id: 10, description: 'after-desc' },
    cleared: ['name'],
    baseline: { id: 10, description: 'before-desc', name: 'old-name' },
    label: 'vlan 10',
  })
  s.markDelete(DEV, { path: VLAN_PATH, listKey: 'vlan', keyValue: '30', label: 'vlan 30' })
}

describe('ChangesContentDialog · 变更内容弹窗（FE-23，纯前端渲染）', () => {
  it('图例计数：增加(1) 修改(1) 删除(1)', async () => {
    seedThree()
    const w = mountDialog()
    await flushPromises()
    await w.vm.$nextTick()
    const legend = document.body.querySelector('[data-test="changes-legend"]')!.textContent!
    expect(legend).toContain('增加(1)')
    expect(legend).toContain('修改(1)')
    expect(legend).toContain('删除(1)')
  })

  it('树形三列：条目行携标签，字段行展示 变更前/变更后（修改黄、新值绿、删除红）', async () => {
    seedThree()
    const w = mountDialog()
    await flushPromises()
    await w.vm.$nextTick()
    const body = document.body.textContent!
    expect(body).toContain('vlan 10')
    expect(body).toContain('before-desc')
    expect(body).toContain('after-desc')
    // cleared 叶 = 删除行：变更前有旧值
    expect(body).toContain('old-name')
    expect(document.body.querySelector('[data-test="changes-table"] .cell-removed')).toBeTruthy()
    expect(document.body.querySelector('[data-test="changes-table"] .cell-added')).toBeTruthy()
  })

  it('待创建条目：字段全部为变更后新值（绿）', async () => {
    seedThree()
    const w = mountDialog()
    await flushPromises()
    await w.vm.$nextTick()
    expect(document.body.textContent).toContain('newbie')
  })

  it('空变更集：空态文案，无表格行', async () => {
    const w = mountDialog()
    await flushPromises()
    await w.vm.$nextTick()
    expect(document.body.textContent).toContain('暂无待提交变更')
  })
})
