import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import ModuleConsolePage from '../../src/views/ModuleConsolePage.vue'
import RpcExecuteTab from '../../src/components/config/RpcExecuteTab.vue'
import { useDeviceStore } from '../../src/stores/device'
import { getYangSchema, getConfig } from '../../src/api'
import { ifmNestedSchema } from './moduleConsole.fixture'

vi.mock('../../src/api')

// 路由提供 :module（+可选 :rpcName，FE-19 rpc 直达）参数（页面零 per-module props）。
const routeState = vi.hoisted(() => ({ params: { module: 'ifm' } as Record<string, string> }))
vi.mock('vue-router', () => ({
  useRoute: () => routeState,
}))

let pinia: ReturnType<typeof createPinia>
function mountPage() {
  return mount(ModuleConsolePage, {
    global: { plugins: [pinia, ElementPlus] },
  })
}

beforeEach(() => {
  vi.clearAllMocks()
  routeState.params = { module: 'ifm' }
  // 全局设备上下文：Tab 内容区以已选设备为前提（未选走引导空态，另测）。
  pinia = createPinia()
  setActivePinia(pinia)
  useDeviceStore().selectDevice('192.168.1.1')
  vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: ifmNestedSchema } } as any)
  vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: {} } } } as any)
})

// 含 rpc 的 schema（真实 huawei-ifm rpc restart-if + input if-name）。
const ifmSchemaWithRpc = {
  module: 'ifm',
  title: 'ifm',
  vendor: 'huawei',
  fields: ifmNestedSchema.fields,
  rpcs: [
    {
      name: 'restart-if',
      label: 'restart-if',
      highRisk: true,
      input: [{ path: 'if-name', type: 'string', label: 'if-name' }],
    },
  ],
}

describe('ModuleConsolePage · Tab 由模块根派生（零模块硬编码，FE-10）', () => {
  it('根子节点派生一级 Tab：global/damp 表单、interfaces/auto-recovery-times 列表', async () => {
    const w = mountPage()
    await flushPromises()
    await flushPromises() // res 懒加载重标（UI-03）落定
    // 只取一级 Tab 栏（FE-02 后表单 Tab 内还有嵌套分组二级 Tab，裸查会混入）。
    const labels = w
      .findAll('.console-tabs > .el-tabs__header .el-tabs__item')
      .map((n) => n.text().trim())
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

  it('rpc 不再进 Tab 栏（FE-19：导航落点迁移到左树）', async () => {
    vi.mocked(getYangSchema).mockResolvedValue({
      data: { success: true, data: ifmSchemaWithRpc },
    } as any)
    const w = mountPage()
    await flushPromises()
    await flushPromises() // res 懒加载重标（UI-03）落定
    const labels = w.findAll('.el-tabs__item').map((n) => n.text().trim())
    // Tab 栏仅配置容器，无任何 rpc Tab（本地化名与原始名都不该出现）。
    expect(labels).not.toContain('重启接口')
    expect(labels).not.toContain('restart-if')
    const vm = w.vm as any
    expect(vm.tabs.some((t: any) => t.kind === 'rpc')).toBe(false)
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

describe('ModuleConsolePage · rpc 直达路由（FE-19）', () => {
  beforeEach(() => {
    vi.mocked(getYangSchema).mockResolvedValue({
      data: { success: true, data: ifmSchemaWithRpc },
    } as any)
  })

  it('rpc 模式：仅渲染该 rpc 执行面板（无 Tab 栏），面包屑含本地化 rpc 名', async () => {
    routeState.params = { module: 'ifm', rpcName: 'restart-if' }
    const w = mountPage()
    await flushPromises()
    await flushPromises() // res 懒加载重标落定
    // 仅一个 rpc 执行面板、无配置 Tab 栏。
    const panel = w.findComponent(RpcExecuteTab)
    expect(panel.exists()).toBe(true)
    expect((panel.props('tab') as any).rpc.name).toBe('restart-if')
    // 面板拿到的是本地化后的 rpc（UI-03：标签与 input 叶同查表）。
    expect((panel.props('tab') as any).rpc.label).toBe('重启接口')
    expect((panel.props('tab') as any).rpc.input[0].label).toBe('重启接口名')
    expect(w.findAll('.el-tabs__item')).toHaveLength(0)
    const crumb = w.findAll('.el-breadcrumb__inner').map((n) => n.text().trim())
    expect(crumb).toEqual(['配置', 'huawei', 'ifm', '重启接口'])
  })

  it('未知 rpcName：明确错误提示、页面不崩（R08）', async () => {
    routeState.params = { module: 'ifm', rpcName: 'no-such-rpc' }
    const w = mountPage()
    await flushPromises()
    await flushPromises()
    expect(w.findComponent(RpcExecuteTab).exists()).toBe(false)
    const alert = w.find('[data-test="rpc-not-found"]')
    expect(alert.exists()).toBe(true)
    expect(alert.text()).toContain('no-such-rpc')
  })

  it('未选设备：rpc 模式同走引导空态（FE-10 口径）', async () => {
    routeState.params = { module: 'ifm', rpcName: 'restart-if' }
    setActivePinia(pinia)
    useDeviceStore().selectDevice('')
    const w = mountPage()
    await flushPromises()
    expect(w.find('[data-test="select-device-empty"]').exists()).toBe(true)
    expect(w.findComponent(RpcExecuteTab).exists()).toBe(false)
  })
})
