import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import ModuleConsolePage from '../../src/views/ModuleConsolePage.vue'
import { useDeviceStore } from '../../src/stores/device'
import { getYangSchema, getConfig, listDevices } from '../../src/api'
import { ifmNestedSchema } from './moduleConsole.fixture'

vi.mock('../../src/api')

// 路由 mock：reactive（页面 watch route.query.device，需要真实响应式触发），
// query 每用例可变（vi.mock 工厂提升，须 vi.hoisted 中转）。
const mockRouteBox = vi.hoisted(() => ({ current: null as any }))
vi.mock('vue-router', async () => {
  const { reactive } = await import('vue')
  mockRouteBox.current = reactive({ params: { module: 'ifm' }, query: {} as Record<string, unknown> })
  return { useRoute: () => mockRouteBox.current }
})
const mockRoute = () => mockRouteBox.current

function mountPage(pinia: ReturnType<typeof createPinia>) {
  return mount(ModuleConsolePage, {
    global: { plugins: [pinia, ElementPlus] },
  })
}

let pinia: ReturnType<typeof createPinia>

beforeEach(() => {
  vi.clearAllMocks()
  mockRoute().query = {}
  pinia = createPinia()
  setActivePinia(pinia)
  vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: ifmNestedSchema } } as any)
  vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: {} } } } as any)
  vi.mocked(listDevices).mockResolvedValue({
    data: {
      success: true,
      data: { devices: [{ ip: '192.168.1.1', port: 830, online: true }, { ip: '192.168.1.2', port: 830, online: true }] },
    },
  } as any)
})

// 全局设备上下文（FE-10）：选一次设备，模块间切换保持；深链写入同一上下文；
// 未选设备引导空态而非静默空数据。
describe('ModuleConsolePage · 全局设备上下文', () => {
  it('跨模块保持：上一页选中的设备在重挂载（切模块）后沿用', async () => {
    const store = useDeviceStore()
    const first = mountPage(pinia)
    await flushPromises()
    store.selectDevice('192.168.1.2')
    await flushPromises()
    first.unmount()

    const second = mountPage(pinia) // 模拟左树切到另一模块
    await flushPromises()
    expect(store.selectedDeviceIp).toBe('192.168.1.2')
    expect(second.find('[data-test="select-device-empty"]').exists()).toBe(false)
    expect(second.findAll('.el-tabs__item').length).toBeGreaterThan(0)
  })

  it('?device= 深链写入全局上下文（覆盖旧值）', async () => {
    const store = useDeviceStore()
    store.selectDevice('192.168.1.1')
    mockRoute().query = { device: '192.168.1.2' }
    mountPage(pinia)
    await flushPromises()
    expect(store.selectedDeviceIp).toBe('192.168.1.2')
  })

  it('挂载后 query 变化（前进/后退复用组件）重新写入上下文', async () => {
    const store = useDeviceStore()
    mockRoute().query = { device: '192.168.1.1' }
    mountPage(pinia)
    await flushPromises()
    expect(store.selectedDeviceIp).toBe('192.168.1.1')

    // 模拟浏览器后退到携带另一 device 的历史条目（组件复用，仅 query 变化）
    mockRoute().query = { device: '192.168.1.2' }
    await flushPromises()
    expect(store.selectedDeviceIp).toBe('192.168.1.2')
  })

  it('重复 query 参数（数组）取首个，不写入逗号拼接垃圾值', async () => {
    const store = useDeviceStore()
    mockRoute().query = { device: ['192.168.1.1', '192.168.1.2'] }
    mountPage(pinia)
    await flushPromises()
    expect(store.selectedDeviceIp).toBe('192.168.1.1')
  })

  it('无 query 时沿用 store 现值（不清空）', async () => {
    const store = useDeviceStore()
    store.selectDevice('192.168.1.1')
    mountPage(pinia)
    await flushPromises()
    expect(store.selectedDeviceIp).toBe('192.168.1.1')
  })

  it('未选设备：渲染引导空态、隐藏 Tab；选中后恢复', async () => {
    const store = useDeviceStore()
    const w = mountPage(pinia)
    await flushPromises()
    expect(w.find('[data-test="select-device-empty"]').exists()).toBe(true)
    expect(w.findAll('.el-tabs__item')).toHaveLength(0)

    store.selectDevice('192.168.1.1')
    await flushPromises()
    expect(w.find('[data-test="select-device-empty"]').exists()).toBe(false)
    expect(w.findAll('.el-tabs__item').length).toBeGreaterThan(0)
  })

  it('schema 加载失败：错误提示可见、设备选择仍可用（R08 不回退）', async () => {
    vi.mocked(getYangSchema).mockRejectedValue(new Error('boom'))
    const store = useDeviceStore()
    store.selectDevice('192.168.1.1')
    const w = mountPage(pinia)
    await flushPromises()
    expect(w.find('.el-alert').exists()).toBe(true)
    expect(w.find('.el-select').exists()).toBe(true)
  })

  it('schema 失败且未选设备：只显示错误告警，不并排「先选设备」引导（引导无效）', async () => {
    vi.mocked(getYangSchema).mockRejectedValue(new Error('boom'))
    const w = mountPage(pinia)
    await flushPromises()
    expect(w.find('.el-alert').exists()).toBe(true)
    expect(w.find('[data-test="select-device-empty"]').exists()).toBe(false)
  })
})
