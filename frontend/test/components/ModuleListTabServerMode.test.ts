import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import ElementPlus from 'element-plus'
import ModuleListTab from '../../src/components/config/ModuleListTab.vue'
import { getConfig } from '../../src/api'
import { useChangesetStore } from '../../src/stores/changeset'
import { deriveTabs, configPathFor } from '../../src/utils/moduleConsole'
import { ifmNestedSchema, seedRows } from '../views/moduleConsole.fixture'

// F2（FE-25）：列表 Tab 双模式——阈值自适应服务端分页。
// mock 后端：带 query 返回 BR-13 ListPage 形状并模拟过滤/排序/切片。

vi.mock('../../src/api')

const interfacesTab = deriveTabs(ifmNestedSchema.fields).find((t) => t.name === 'interfaces')!

let pinia: Pinia

function mountTab() {
  return mount(ModuleListTab, {
    props: { tab: interfacesTab, rootName: 'ifm', device: '10.0.0.1' },
    global: { plugins: [pinia, ElementPlus] },
  })
}

type Q = { limit: number; offset?: number; filters?: string[]; sort?: string; sortDir?: string }

// serverStub：n 行「设备数据」，按 BR-13 语义响应分页查询。
function serverStub(n: number) {
  const all = Array.from({ length: n }, (_, i) => ({
    name: `GE1/0/${i}`,
    class: i % 5 === 0 ? 'sub-interface' : 'main-interface',
    type: '200GE',
    mtu: 1000 + i,
  }))
  vi.mocked(getConfig).mockImplementation(((_ip: string, _path: string, _force?: boolean, _state?: boolean, query?: Q) => {
    if (!query) {
      return Promise.resolve({ data: { data: { data: { interface: all } } } } as any)
    }
    let rows = all
    for (const f of query.filters ?? []) {
      const m = f.match(/^(.+?)(==|~=)(.*)$/)!
      const [, k, op, v] = m
      rows = rows.filter((r: any) =>
        op === '==' ? String(r[k]) === v : String(r[k]).toLowerCase().includes(v.toLowerCase()),
      )
    }
    if (query.sort) {
      rows = [...rows].sort((a: any, b: any) => {
        const d = a[query.sort!] < b[query.sort!] ? -1 : a[query.sort!] > b[query.sort!] ? 1 : 0
        return query.sortDir === 'desc' ? -d : d
      })
    }
    const offset = query.offset ?? 0
    return Promise.resolve({
      data: {
        data: {
          data: { rows: rows.slice(offset, offset + query.limit), total: rows.length, limit: query.limit, offset },
          cached: false,
          source: 'device',
        },
      },
    } as any)
  }) as any)
}

function lastQuery(): Q | undefined {
  return vi.mocked(getConfig).mock.calls.at(-1)?.[4] as Q | undefined
}

beforeEach(() => {
  pinia = createPinia()
  setActivePinia(pinia)
  vi.resetAllMocks()
})

describe('FE-25 · 小表纯前端模式（≤200 行零回归）', () => {
  it('首读一次拿全量（ListPage total≤200），本地搜索翻页不发新请求', async () => {
    vi.mocked(getConfig).mockResolvedValue({
      data: { data: { data: { rows: seedRows, total: seedRows.length, limit: 200, offset: 0 } } },
    } as any)
    const w = mountTab()
    await flushPromises()
    expect(vi.mocked(getConfig).mock.calls).toHaveLength(1)
    expect((vi.mocked(getConfig).mock.calls[0][4] as Q).limit).toBe(200)
    expect(w.findAll('.el-table__body tr')).toHaveLength(5)

    // 本地搜索：零额外请求（现状交互）。
    const vm = w.vm as any
    vm.draft.class = 'sub-interface'
    vm.applySearch()
    await flushPromises()
    expect(w.findAll('.el-table__body tr')).toHaveLength(2)
    expect(vi.mocked(getConfig).mock.calls).toHaveLength(1)
  })

  it('旧整树形状（无 rows/total）走 legacy 客户端路径（旧后端/回退兼容）', async () => {
    vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: { interface: seedRows } } } } as any)
    const w = mountTab()
    await flushPromises()
    expect(w.findAll('.el-table__body tr')).toHaveLength(5)
  })
})

describe('FE-25 · 大表服务端模式（>200 行）', () => {
  it('探测后转服务端模式：翻页带 limit/offset 重新请求，总记录数取响应 total', async () => {
    serverStub(500)
    const w = mountTab()
    await flushPromises()
    // 探测（limit=200）+ 当前页窗口（limit=pageSize）
    expect(vi.mocked(getConfig).mock.calls).toHaveLength(2)
    expect(lastQuery()).toMatchObject({ limit: 10, offset: 0 })
    expect(w.findAll('.el-table__body tr')).toHaveLength(10)
    expect(w.find('[data-test="query-summary"]').text()).toContain('500')

    // 跳第 5 页 → 服务端取窗口 offset=40。
    const vm = w.vm as any
    vm.page = 5
    vm.onPageChange()
    await flushPromises()
    expect(lastQuery()).toMatchObject({ limit: 10, offset: 40 })
    expect(w.find('.el-table__body tr').text()).toContain('GE1/0/40')
  })

  it('高级搜索下推为 filter 参数且页码复位，不在本地对当前页再过滤', async () => {
    serverStub(500)
    const w = mountTab()
    await flushPromises()
    const vm = w.vm as any
    vm.page = 3
    vm.onPageChange()
    await flushPromises()

    vm.draft.class = 'sub-interface'
    vm.applySearch()
    await flushPromises()
    expect(vm.page).toBe(1)
    expect(lastQuery()).toMatchObject({ offset: 0, filters: ['class==sub-interface'] })
    // 100 行命中（i%5==0），当前页 10 行全部来自服务端过滤结果。
    expect(w.find('[data-test="query-summary"]').text()).toContain('100')
    expect(w.findAll('.el-table__body tr')).toHaveLength(10)
  })

  it('排序下推：sort-change 触发带 sort/sort_dir 的重新请求并复位页码', async () => {
    serverStub(500)
    const w = mountTab()
    await flushPromises()
    const vm = w.vm as any
    vm.onSortChange({ prop: 'mtu', order: 'descending' })
    await flushPromises()
    expect(lastQuery()).toMatchObject({ sort: 'mtu', sortDir: 'desc', offset: 0 })
    expect(w.find('.el-table__body tr').text()).toContain('GE1/0/499')
  })

  it('pending create 行本地叠加展示，不计入服务端 total', async () => {
    serverStub(500)
    const w = mountTab()
    await flushPromises()
    const entryPath = '/' + configPathFor('ifm', interfacesTab.field.path)
    useChangesetStore().upsert('10.0.0.1', {
      op: 'create',
      path: entryPath,
      listKey: 'interface',
      keyValue: 'NEWIF',
      payload: { name: 'NEWIF' },
      label: 'interfaces NEWIF',
    })
    await flushPromises()
    // 当前页 10 行 + 本地待创建 1 行；分页总数仍为服务端 total。
    expect(w.findAll('.el-table__body tr')).toHaveLength(11)
    expect(w.find('[data-test="mark-create"]').exists()).toBe(true)
    expect(w.find('[data-test="query-summary"]').text()).toContain('500')
  })

  it('「获取数据源」强刷复位第一页并重新探测（force_refresh + limit=200）', async () => {
    serverStub(500)
    const w = mountTab()
    await flushPromises()
    const vm = w.vm as any
    vm.page = 5
    vm.onPageChange()
    await flushPromises()

    vm.fetchSource()
    await flushPromises()
    expect(vm.page).toBe(1)
    const probeCall = vi.mocked(getConfig).mock.calls.at(-2)!
    expect(probeCall[2]).toBe(true) // force_refresh
    expect((probeCall[4] as Q).limit).toBe(200)
    expect(lastQuery()).toMatchObject({ limit: 10, offset: 0 })
  })

  it('带参被拒（信封 400，如非 list 路径）回退旧读法（R08 降级）', async () => {
    const all = { interface: seedRows }
    vi.mocked(getConfig).mockImplementation(((_ip: string, _p: string, _f?: boolean, _s?: boolean, query?: Q) =>
      Promise.resolve(
        query
          ? ({ data: { code: 400, success: false, message: '分页参数无效' } } as any)
          : ({ data: { data: { data: all } } } as any),
      )) as any)
    const w = mountTab()
    await flushPromises()
    expect(w.findAll('.el-table__body tr')).toHaveLength(5)
    expect(vi.mocked(getConfig).mock.calls).toHaveLength(2)
    expect(vi.mocked(getConfig).mock.calls.at(-1)![4]).toBeUndefined()
  })
})
