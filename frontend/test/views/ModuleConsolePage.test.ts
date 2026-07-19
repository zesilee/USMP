import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import ModuleConsolePage from '../../src/views/ModuleConsolePage.vue'
import { useDeviceStore } from '../../src/stores/device'
import { getYangSchema, getConfig } from '../../src/api'
import { ifmNestedSchema } from './moduleConsole.fixture'

vi.mock('../../src/api')

// 路由仅提供 :module 参数（页面零 per-module props，FE-10）。
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { module: 'ifm' } }),
}))

let pinia: ReturnType<typeof createPinia>
function mountPage() {
  return mount(ModuleConsolePage, {
    global: { plugins: [pinia, ElementPlus] },
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  // 全局设备上下文：Tab 内容区以已选设备为前提（未选走引导空态，另测）。
  pinia = createPinia()
  setActivePinia(pinia)
  useDeviceStore().selectDevice('192.168.1.1')
  vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: ifmNestedSchema } } as any)
  vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: {} } } } as any)
})

describe('ModuleConsolePage · Tab 由模块根派生（零模块硬编码，FE-10）', () => {
  it('根子节点派生一级 Tab：global/damp 表单、interfaces/auto-recovery-times 列表', async () => {
    const w = mountPage()
    await flushPromises()
    await flushPromises() // res 懒加载重标（UI-03）落定
    const labels = w.findAll('.el-tabs__item').map((n) => n.text().trim())
    // UI-03：Tab 标签经 snd res 本地化（zh 默认）；tab name 仍为 YANG 节点名。
    expect(labels).toEqual(['全局配置属性', '接口物理状态振荡抑制使能', '接口列表', '自动恢复时间列表'])
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
    await flushPromises() // res 懒加载重标（UI-03）落定
    const crumb = () => w.findAll('.el-breadcrumb__inner').map((n) => n.text().trim())
    expect(crumb()).toEqual(['配置', 'huawei', 'ifm', '全局配置属性'])
    ;(w.vm as any).activeTab = 'interfaces'
    await flushPromises()
    expect(crumb()).toEqual(['配置', 'huawei', 'ifm', '接口列表'])
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
