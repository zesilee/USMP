import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import ElementPlus from 'element-plus'
import ItemDetailPane from '../../src/components/config/ItemDetailPane.vue'
import { setConfig, getDeviceReconcile } from '../../src/api'
import { useChangesetStore } from '../../src/stores/changeset'
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

let pinia: Pinia

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
    global: { plugins: [pinia, ElementPlus] },
  })
}

beforeEach(() => {
  // 共享 pinia：mount 与测试内 useChangesetStore 必须同实例（断言变更集内容）。
  pinia = createPinia()
  setActivePinia(pinia)
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

  it('确定=入变更集（FE-21 攒批）：零网络请求、条目携 op/payload/baseline，emit staged', async () => {
    const w = mountPane()
    await flushPromises()
    const vm = w.vm as any
    vm.form.formData['description'] = 'new-desc'
    await flushPromises()
    await vm.submit()
    await flushPromises()
    expect(vi.mocked(setConfig)).not.toHaveBeenCalled()

    const cs = useChangesetStore()
    const entry = cs.entryFor('10.0.0.1', '/ifm:ifm/ifm:interfaces', '200GE0/1/0.1')!
    expect(entry).toBeTruthy()
    expect(entry.op).toBe('update')
    expect(entry.payload?.['description']).toBe('new-desc')
    expect(entry.baseline?.['admin-status']).toBe('down')
    expect(w.emitted('staged')?.[0]?.[0]).toBe('200GE0/1/0.1')
  })

  it('创建态确定：op=create 入集（FE-21 创建态入集）', async () => {
    const w = mountPane({ mode: 'create', row: null })
    await flushPromises()
    const vm = w.vm as any
    vm.form.formData['name'] = 'GE9/9/9'
    vm.form.formData['class'] = 'main-interface'
    await flushPromises()
    await vm.submit()
    await flushPromises()
    const entry = useChangesetStore().entryFor('10.0.0.1', '/ifm:ifm/ifm:interfaces', 'GE9/9/9')!
    expect(entry?.op).toBe('create')
    expect(vi.mocked(setConfig)).not.toHaveBeenCalled()
  })

  it('编辑变更集已有条目：以最新值回填并合并更新（FE-21 合并语义）', async () => {
    const cs = useChangesetStore()
    cs.upsert('10.0.0.1', {
      op: 'update',
      path: '/ifm:ifm/ifm:interfaces',
      listKey: 'interface',
      keyValue: '200GE0/1/0.1',
      payload: { name: '200GE0/1/0.1', description: 'pending-v1' },
      cleared: [],
      baseline: { ...seedRows[3] },
      label: 'interface 200GE0/1/0.1',
    })
    const w = mountPane()
    await flushPromises()
    const vm = w.vm as any
    expect(vm.form.formData['description']).toBe('pending-v1')

    vm.form.formData['description'] = 'pending-v2'
    await flushPromises()
    await vm.submit()
    await flushPromises()
    expect(cs.countFor('10.0.0.1')).toBe(1)
    const entry = cs.entryFor('10.0.0.1', '/ifm:ifm/ifm:interfaces', '200GE0/1/0.1')!
    expect(entry.payload?.['description']).toBe('pending-v2')
    expect(entry.baseline?.['admin-status']).toBe('down')
    w.unmount()
  })

  it('NCE 控件规范（FE-22）：key 叶钥匙图标；三列栅格与整行控件类', async () => {
    const w = mountPane()
    await flushPromises()
    // key 叶（name）label 带钥匙图标
    expect(w.find('[data-test="key-icon"]').exists()).toBe(true)
    // 表单挂三列栅格类；leaf-list/嵌套容器项占整行
    expect(w.find('.config-form--grid').exists()).toBe(true)
  })

  it('字段级清除·基线无值（FE-22 边界）：本次新填又清掉→不入 diff/payload/cleared', async () => {
    const w = mountPane()
    await flushPromises()
    const vm = w.vm as any
    // description 基线无值，本次新填再清除 → 仅置空
    vm.form.formData['description'] = 'to-clear'
    await flushPromises()
    const clearBtn = w.find('[data-test="clear-description"]')
    expect(clearBtn.exists()).toBe(true)
    await clearBtn.trigger('click')
    await flushPromises()
    expect(vm.form.formData['description']).toBeUndefined()
    expect(vm.form.diff.value.some((d: any) => d.key === 'description')).toBe(false)
    expect('description' in vm.form.visiblePayload()).toBe(false)
    // key 叶编辑态禁用 → 无清除钮；readonly 同理
    expect(w.find('[data-test="clear-name"]').exists()).toBe(false)
  })

  it('字段级清除·基线有值=删除意图（FE-22 二期语义）：diff 现 remove、确定后 cleared 入集', async () => {
    const w = mountPane()
    await flushPromises()
    const vm = w.vm as any
    // admin-status 基线值 down → 清除 = 删除意图
    const clearBtn = w.find('[data-test="clear-admin-status"]')
    expect(clearBtn.exists()).toBe(true)
    await clearBtn.trigger('click')
    await flushPromises()
    const rm = vm.form.diff.value.find((d: any) => d.key === 'admin-status')
    expect(rm?.op).toBe('remove')
    expect(vm.dirty).toBe(true, )

    await vm.submit()
    await flushPromises()
    const entry = useChangesetStore().entryFor('10.0.0.1', '/ifm:ifm/ifm:interfaces', '200GE0/1/0.1')!
    expect(entry.cleared).toContain('admin-status')
    expect('admin-status' in (entry.payload ?? {})).toBe(false)
  })

  it('必填字段清除后提交被权威门禁拦截（FE-22 负路径）', async () => {
    const w = mountPane({ mode: 'create', row: null })
    await flushPromises()
    const vm = w.vm as any
    vm.form.formData['name'] = 'GE0/0/1'
    await flushPromises()
    const clearBtn = w.find('[data-test="clear-name"]')
    expect(clearBtn.exists()).toBe(true) // 创建态 key 可编辑，可清除
    await clearBtn.trigger('click')
    await flushPromises()
    expect(vm.form.blocked.value).toBe(true)
    await vm.submit()
    expect(vi.mocked(setConfig)).not.toHaveBeenCalled()
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
