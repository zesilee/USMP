import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import ModuleConsolePage from '../../src/views/ModuleConsolePage.vue'
import { useDeviceStore } from '../../src/stores/device'
import * as apiModule from '../../src/api'
import { ifmNestedSchema } from './moduleConsole.fixture'

// FE-18（F2）——原生控制台软归属徽标：选中设备上本模块被业务意图认领时显示徽标；
// 未认领/查询失败不显示（R08 静默降级）。

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { module: 'vlan' } }),
}))

let pinia: ReturnType<typeof createPinia>
function mountPage() {
  return mount(ModuleConsolePage, {
    global: { plugins: [pinia, ElementPlus] },
  })
}

describe('ModuleConsolePage ownership badge (FE-18)', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
    pinia = createPinia()
    setActivePinia(pinia)
    vi.spyOn(apiModule, 'getYangSchema').mockResolvedValue({
      data: { success: true, data: ifmNestedSchema },
    } as any)
  })

  it('设备上本模块被认领 → 显示徽标（含意图数）', async () => {
    vi.spyOn(apiModule, 'getOwnership').mockResolvedValue({
      data: {
        success: true,
        data: {
          device: '10.0.0.1',
          claims: [
            { intent: 'default/biz-100', module: 'vlan', path: '/vlan:vlan/vlan:vlans/vlan[id=100]' },
            { intent: 'default/biz-100', module: 'ifm', path: '/ifm:ifm/ifm:interfaces/interface[name=GE0/0/1]' },
          ],
        },
      },
    } as any)
    const wrapper = mountPage()
    await flushPromises()

    // 选中设备（全局设备上下文）触发归属查询。
    useDeviceStore().selectDevice('10.0.0.1')
    await flushPromises()

    const badge = wrapper.find('[data-test="ownership-badge"]')
    expect(badge.exists()).toBe(true)
    expect(badge.text()).toContain('由业务配置管理 (1)') // 仅本模块（vlan）的认领意图数
  })

  it('带预置设备上下文挂载（查看配置入口/跨页返回）也应立即出徽标', async () => {
    vi.spyOn(apiModule, 'getOwnership').mockResolvedValue({
      data: {
        success: true,
        data: {
          device: '10.0.0.1',
          claims: [{ intent: 'default/biz-100', module: 'vlan', path: '/vlan:vlan/vlan:vlans/vlan[id=100]' }],
        },
      },
    } as any)
    // 挂载前上下文已就位（全局设备上下文的主流程）：watch 注册时源已不再变化，
    // 须 immediate 触发归属查询，否则徽标静默缺失。
    useDeviceStore().selectDevice('10.0.0.1')
    const wrapper = mountPage()
    await flushPromises()
    expect(wrapper.find('[data-test="ownership-badge"]').exists()).toBe(true)
  })

  it('未认领 → 无徽标；查询失败 → 无徽标不报错（R08）', async () => {
    vi.spyOn(apiModule, 'getOwnership').mockResolvedValue({
      data: { success: true, data: { device: '10.0.0.1', claims: [] } },
    } as any)
    const wrapper = mountPage()
    await flushPromises()
    const store = useDeviceStore()
    store.selectDevice('10.0.0.1')
    await flushPromises()
    expect(wrapper.find('[data-test="ownership-badge"]').exists()).toBe(false)

    vi.spyOn(apiModule, 'getOwnership').mockRejectedValue(new Error('boom'))
    store.selectDevice('10.0.0.2')
    await flushPromises()
    expect(wrapper.find('[data-test="ownership-badge"]').exists()).toBe(false)
  })
})
