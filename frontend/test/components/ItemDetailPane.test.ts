import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import ItemDetailPane from '../../src/components/config/ItemDetailPane.vue'
import { setConfig, getDeviceReconcile } from '../../src/api'
import { deriveTabs } from '../../src/utils/moduleConsole'
import type { Field } from '../../src/utils/crdSchemaParser'
import { ifmNestedSchema, seedRows } from '../views/moduleConsole.fixture'

vi.mock('../../src/api')

// 在共享 fixture 的 interface list 上追加嵌套子节点（不影响列/搜索派生：
// group/list 不入列），覆盖 FE-21 二级 Tab：嵌套 group→子表单、嵌套 list→子表格。
const nestedGroup: Field = {
  path: '/ifm/interfaces/interface/statistics-cfg',
  type: 'group',
  label: 'statistics-cfg',
  fields: [
    { path: '/ifm/interfaces/interface/statistics-cfg/interval', type: 'number', label: 'interval' },
  ],
}
const nestedList: Field = {
  path: '/ifm/interfaces/interface/trap-thresholds',
  type: 'list',
  label: 'trap-threshold',
  fields: [
    { path: '/ifm/interfaces/interface/trap-thresholds/trap-threshold/kind', type: 'string', label: 'kind', isKey: true },
  ],
}
const baseList = ifmNestedSchema.fields.find((f) => f.label === 'interfaces')!.fields![0]
const richList: Field = { ...baseList, fields: [...(baseList.fields || []), nestedGroup, nestedList] }
const richTab = {
  ...deriveTabs(ifmNestedSchema.fields).find((t) => t.name === 'interfaces')!,
  listField: richList,
}

function mountPane(extra: Record<string, any> = {}) {
  return mount(ItemDetailPane, {
    props: {
      tab: richTab,
      rootName: 'ifm',
      device: '10.0.0.1',
      mode: 'edit',
      row: { ...seedRows[3] },
      postKey: 'interface',
      ...extra,
    },
    global: { plugins: [createPinia(), ElementPlus] },
  })
}

beforeEach(() => {
  // resetAllMocks 而非 clearAllMocks：Once 队列跨用例残留会让 baseline 桩错位，
  // 轮询永远读到空 statuses → 真延时打满超时。
  vi.resetAllMocks()
  vi.mocked(setConfig).mockResolvedValue({ data: { data: { reconciliation: { triggered: true } } } } as any)
  // baseline 轮 statuses 为空（parseRun=0），下发后首轮即 converged（last_run 推进）
  // → 对账链一轮收尾，测试不吃真实轮询延时。
  vi.mocked(getDeviceReconcile)
    .mockResolvedValueOnce({ data: { data: { statuses: [] } } } as any)
    .mockResolvedValue({
      data: { data: { statuses: [{ path: '/ifm:ifm/ifm:interfaces', last_run: '2026-07-31T12:00:00Z', outcome: 'converged' }] } },
    } as any)
})

describe('ItemDetailPane · 面包屑与关闭（FE-21）', () => {
  it('编辑态面包屑 = list 标签 > 主键值；有关闭按钮', async () => {
    const w = mountPane()
    await flushPromises()
    const crumb = w.find('[data-test="detail-breadcrumb"]').text()
    expect(crumb).toContain('interface')
    expect(crumb).toContain('200GE0/1/0.1')
    expect(w.find('[data-test="detail-close"]').exists()).toBe(true)
  })

  it('创建态面包屑展示创建文案；点关闭 emit close', async () => {
    const w = mountPane({ mode: 'create', row: null })
    await flushPromises()
    expect(w.find('[data-test="detail-breadcrumb"]').text()).toContain('创建')
    await w.find('[data-test="detail-close"]').trigger('click')
    expect(w.emitted('close')).toBeTruthy()
  })
})

describe('ItemDetailPane · 二级 Tab（deriveDetailTabs 驱动）', () => {
  it('主表单 Tab + 嵌套 group 子表单 Tab + 嵌套 list 子表格 Tab', async () => {
    const w = mountPane()
    await flushPromises()
    const tabNames = w.findAll('.el-tabs__item').map((n) => n.text().trim())
    expect(tabNames[0]).toBe('interface')
    expect(tabNames).toContain('statistics-cfg')
    expect(tabNames).toContain('trap-threshold')
  })

  it('无嵌套子节点 → 单主 Tab 不渲染 Tab 头（退化边界）', async () => {
    const flatTab = deriveTabs(ifmNestedSchema.fields).find((t) => t.name === 'interfaces')!
    const w = mountPane({ tab: flatTab })
    await flushPromises()
    expect(w.findAll('.el-tabs__item').length).toBe(0)
    // 表单仍在
    expect(w.findAll('.el-form-item').length).toBeGreaterThan(0)
  })
})

describe('ItemDetailPane · 表单编辑（FE-11 门禁语义迁移）', () => {
  it('编辑态：key 叶与 operationExclude∋update 叶禁用；创建态全可编', async () => {
    const w = mountPane()
    await flushPromises()
    const vm = w.vm as any
    expect(vm.isFieldDisabled(richList.fields!.find((f: Field) => f.label === 'name'))).toBe(true)
    expect(vm.isFieldDisabled(richList.fields!.find((f: Field) => f.label === 'class'))).toBe(true)
    expect(vm.isFieldDisabled(richList.fields!.find((f: Field) => f.label === 'description'))).toBe(false)

    const wc = mountPane({ mode: 'create', row: null })
    await flushPromises()
    const vmc = wc.vm as any
    expect(vmc.isFieldDisabled(richList.fields!.find((f: Field) => f.label === 'name'))).toBe(false)
    expect(vmc.isFieldDisabled(richList.fields!.find((f: Field) => f.label === 'class'))).toBe(false)
  })

  it('dirty 暴露：改字段后 dirty=true，供父层切行确认', async () => {
    const w = mountPane()
    await flushPromises()
    const vm = w.vm as any
    expect(vm.dirty).toBe(false)
    vm.form.formData['description'] = 'changed'
    await flushPromises()
    expect(vm.dirty).toBe(true)
  })

  it('提交：调 setConfig 且 payload 含变更；成功 emit saved（携主键值）', async () => {
    const w = mountPane()
    await flushPromises()
    const vm = w.vm as any
    vm.form.formData['description'] = 'new-desc'
    await flushPromises()
    await vm.submit()
    await flushPromises()
    expect(vi.mocked(setConfig)).toHaveBeenCalledTimes(1)
    const [ip, path, payload] = vi.mocked(setConfig).mock.calls[0]
    expect(ip).toBe('10.0.0.1')
    expect(path).toContain('ifm:interfaces')
    expect(JSON.stringify(payload)).toContain('new-desc')
    expect(w.emitted('saved')?.[0]?.[0]).toBe('200GE0/1/0.1')
  })

  it('行切换（row prop 变化）重置表单为新行数据', async () => {
    const w = mountPane()
    await flushPromises()
    const vm = w.vm as any
    vm.form.formData['description'] = 'draft'
    await w.setProps({ row: { ...seedRows[0] } })
    await flushPromises()
    expect(vm.form.formData['name']).toBe('200GE0/1/0')
    expect(vm.form.formData['description']).toBeUndefined()
    expect(vm.dirty).toBe(false)
  })
})
