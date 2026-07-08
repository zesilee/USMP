import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import ModuleConsolePage from '../../src/views/ModuleConsolePage.vue'
import { getYangSchema, getConfig } from '../../src/api'
import { ifmNestedSchema } from './moduleConsole.fixture'

vi.mock('../../src/api')

// 路由仅提供 :module 参数（页面零 per-module props，FE-10）。
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { module: 'ifm' } }),
}))

function mountPage() {
  return mount(ModuleConsolePage, {
    global: { plugins: [createPinia(), ElementPlus] },
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: ifmNestedSchema } } as any)
  vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: {} } } } as any)
})

describe('ModuleConsolePage · Tab 由模块根派生（零模块硬编码，FE-10）', () => {
  it('根子节点派生一级 Tab：global/damp 表单、interfaces/auto-recovery-times 列表', async () => {
    const w = mountPage()
    await flushPromises()
    const labels = w.findAll('.el-tabs__item').map((n) => n.text().trim())
    expect(labels).toEqual(['global', 'damp', 'interfaces', 'auto-recovery-times'])
    const vm = w.vm as any
    const kinds = Object.fromEntries(vm.tabs.map((t: any) => [t.name, t.kind]))
    expect(kinds).toEqual({
      global: 'form',
      damp: 'form',
      interfaces: 'list',
      'auto-recovery-times': 'list',
    })
  })

  it('面包屑 = 配置/厂商/模块/激活 Tab，随 Tab 切换联动', async () => {
    const w = mountPage()
    await flushPromises()
    const crumb = () => w.findAll('.el-breadcrumb__inner').map((n) => n.text().trim())
    expect(crumb()).toEqual(['配置', 'huawei', 'ifm', 'global'])
    ;(w.vm as any).activeTab = 'interfaces'
    await flushPromises()
    expect(crumb()).toEqual(['配置', 'huawei', 'ifm', 'interfaces'])
  })

  it('schema 加载失败：错误提示可见、页面不崩（R08）', async () => {
    vi.mocked(getYangSchema).mockRejectedValue(new Error('boom'))
    const w = mountPage()
    await flushPromises()
    expect(w.find('.el-alert').exists()).toBe(true)
    expect(w.text()).toContain('boom')
    expect(w.findAll('.el-tabs__item')).toHaveLength(0)
  })
})
