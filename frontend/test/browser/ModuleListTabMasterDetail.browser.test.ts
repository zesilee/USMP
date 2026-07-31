import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import ModuleListTab from '../../src/components/config/ModuleListTab.vue'
import { getConfig, setConfig, getDeviceReconcile } from '../../src/api'
import { deriveTabs } from '../../src/utils/moduleConsole'
import type { Field } from '../../src/utils/crdSchemaParser'
import { ifmNestedSchema, seedRows } from '../views/moduleConsole.fixture'

// 真 Chromium（F3）：happy-dom 伪造不了的三类交互——el-table 列头筛选弹层、
// el-popover 列设置弹层（teleport）、master-detail 真实点击链路与二级 Tab 切换。
vi.mock('../../src/api')

// 在 interface list 上加嵌套 list 子节点：详情区二级 Tab 覆盖「子表格 Tab」形态。
const nestedList: Field = {
  path: '/ifm/interfaces/interface/trap-thresholds',
  type: 'list',
  label: 'trap-threshold',
  fields: [
    { path: '/ifm/interfaces/interface/trap-thresholds/trap-threshold/kind', type: 'string', label: 'kind', isKey: true },
  ],
}
const baseTab = deriveTabs(ifmNestedSchema.fields).find((t) => t.name === 'interfaces')!
const richTab = {
  ...baseTab,
  listField: { ...baseTab.listField!, fields: [...(baseTab.listField!.fields || []), nestedList] },
}

let mounted: VueWrapper[] = []
let pinia: ReturnType<typeof createPinia>
function mountTab() {
  const w = mount(ModuleListTab, {
    props: { tab: richTab, rootName: 'ifm', device: '10.0.0.1' },
    global: { plugins: [pinia, ElementPlus] },
    attachTo: document.body,
  })
  mounted.push(w)
  return w
}

afterEach(() => {
  // 真浏览器共享同一 document：不卸载会让多张表叠在 body 上（弹层查找被上例污染）。
  mounted.forEach((w) => w.unmount())
  mounted = []
})

beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
  vi.resetAllMocks()
  vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: { interface: seedRows } } } } as any)
  vi.mocked(setConfig).mockResolvedValue({ data: { data: { reconciliation: { triggered: true } } } } as any)
  vi.mocked(getDeviceReconcile).mockResolvedValue({ data: { data: { statuses: [] } } } as any)
})

describe('ModuleListTab（真浏览器）· master-detail 与弹层（FE-11/FE-21）', () => {
  it('点行 → 详情区展开；二级 Tab 切到嵌套 list 子表格；关闭收起', async () => {
    const w = mountTab()
    await vi.waitFor(() => {
      expect(w.element.querySelectorAll('.el-table__body tr').length).toBe(5)
    }, { timeout: 3000 })

    // 真实点击行 → 详情区展开、行高亮
    ;(w.element.querySelectorAll('.el-table__body tr')[3] as HTMLElement).click()
    await vi.waitFor(() => {
      expect(w.element.querySelector('[data-test="item-detail-pane"]')).toBeTruthy()
      expect(w.element.querySelector('.el-table__body tr.current-row')).toBeTruthy()
    }, { timeout: 3000 })
    expect(w.element.querySelector('[data-test="detail-breadcrumb"]')!.textContent).toContain('200GE0/1/0.1')

    // 二级 Tab：主 Tab + 嵌套 list 子 Tab；切到子表格 Tab 渲染 list 编辑器
    const tabItems = [...w.element.querySelectorAll('[data-test="item-detail-pane"] .el-tabs__item')]
    const sub = tabItems.find((n) => n.textContent?.includes('trap-threshold')) as HTMLElement
    expect(sub).toBeTruthy()
    sub.click()
    await vi.waitFor(() => {
      expect(w.element.querySelector('[data-test="item-detail-pane"] .field-list')).toBeTruthy()
    }, { timeout: 3000 })

    // 关闭 → 收起
    ;(w.element.querySelector('[data-test="detail-close"]') as HTMLElement).click()
    await vi.waitFor(() => {
      expect(w.element.querySelector('[data-test="item-detail-pane"]')).toBeFalsy()
    }, { timeout: 3000 })
  })

  it('enum 列头筛选弹层：勾选 sub-interface → 2 行；重置 → 5 行', async () => {
    const w = mountTab()
    await vi.waitFor(() => {
      expect(w.element.querySelectorAll('.el-table__body tr').length).toBe(5)
    }, { timeout: 3000 })

    // 打开 class 列的筛选弹层（teleport 到 body）
    const triggers = w.element.querySelectorAll('.el-table__column-filter-trigger')
    expect(triggers.length).toBeGreaterThan(0)
    ;(triggers[0] as HTMLElement).click()
    await vi.waitFor(() => {
      expect(document.querySelector('.el-table-filter')).toBeTruthy()
    }, { timeout: 3000 })

    // 勾选 sub-interface（点真实 input，等勾选态可见后再确认）
    const label = [...document.querySelectorAll('.el-table-filter .el-checkbox')].find(
      (n) => n.textContent?.includes('sub-interface'),
    ) as HTMLElement
    expect(label).toBeTruthy()
    ;(label.querySelector('input[type="checkbox"]') as HTMLElement).click()
    await vi.waitFor(() => {
      expect(label.classList.contains('is-checked')).toBe(true)
    }, { timeout: 3000 })
    const confirmBtn = [...document.querySelectorAll('.el-table-filter button')].find(
      (n) => n.textContent?.includes('筛选') || n.textContent?.toLowerCase().includes('confirm'),
    ) as HTMLElement
    confirmBtn.click()
    await vi.waitFor(() => {
      expect(w.element.querySelectorAll('.el-table__body tr').length).toBe(2)
    }, { timeout: 3000 })
  })

  it('列设置弹层：取消勾选 description → 该列隐藏', async () => {
    const w = mountTab()
    await vi.waitFor(() => {
      expect(w.element.querySelectorAll('.el-table__body tr').length).toBe(5)
    }, { timeout: 3000 })

    ;(w.element.querySelector('[data-test="column-settings"]') as HTMLElement).click()
    await vi.waitFor(() => {
      expect(document.querySelector('.col-settings')).toBeTruthy()
    }, { timeout: 3000 })

    const headerTexts = () =>
      [...w.element.querySelectorAll('.el-table__header th .cell')].map((n) => n.textContent?.trim())
    expect(headerTexts()).toContain('description')

    const box = [...document.querySelectorAll('.col-settings .el-checkbox')].find(
      (n) => n.textContent?.includes('description'),
    ) as HTMLElement
    expect(box).toBeTruthy()
    ;(box.querySelector('input[type="checkbox"]') as HTMLElement).click()
    await vi.waitFor(() => {
      expect(headerTexts()).not.toContain('description')
    }, { timeout: 3000 })
  })
})

describe('ModuleListTab（真浏览器）· 批量删除菜单与标记视图（FE-11 二期）', () => {
  it('勾选两行 → 更多▾（teleport 菜单）→ 批量删除确认 → 待删除标记×2 + 取消删除还原', async () => {
    const { useChangesetStore } = await import('../../src/stores/changeset')
    const { ElMessageBox } = await import('element-plus')
    const w = mountTab()
    await new Promise((r) => setTimeout(r, 50))

    // 真实 checkbox 勾选前两行
    const boxes = document.body.querySelectorAll('.el-table__body .el-checkbox__original')
    ;(boxes[0] as HTMLElement).click()
    ;(boxes[1] as HTMLElement).click()
    await w.vm.$nextTick()

    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as any)
    // 打开 teleport 下拉并点批量删除
    await w.find('[data-test="batch-more"]').trigger('click')
    await new Promise((r) => setTimeout(r, 100))
    const item = document.body.querySelector('[data-test="batch-delete"]') as HTMLElement
    expect(item).toBeTruthy()
    item.click()
    await new Promise((r) => setTimeout(r, 50))

    const cs = useChangesetStore()
    expect(cs.countFor('10.0.0.1')).toBe(2)
    expect(document.body.querySelectorAll('[data-test="mark-delete"]').length).toBe(2)
    expect(document.body.querySelectorAll('[data-test="undelete-btn"]').length).toBe(2)

    // 取消删除一条 → 标记还原
    ;(document.body.querySelector('[data-test="undelete-btn"]') as HTMLElement).click()
    await new Promise((r) => setTimeout(r, 50))
    expect(cs.countFor('10.0.0.1')).toBe(1)
    confirmSpy.mockRestore()
  })
})
