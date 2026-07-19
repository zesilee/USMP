import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import ModuleConsolePage from '../../src/views/ModuleConsolePage.vue'
import { getYangSchema, getConfig, listDevices } from '../../src/api'
import { ifmNestedSchema } from './moduleConsole.fixture'

vi.mock('../../src/api')

// 设备管理「查看配置」跳转携带 ?device=<ip>：控制台须预选该设备（回归：参数曾被忽略）。
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { module: 'ifm' }, query: { device: '192.168.1.2' } }),
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
  vi.mocked(listDevices).mockResolvedValue({
    data: {
      success: true,
      data: { devices: [{ ip: '192.168.1.1', port: 830, online: true }, { ip: '192.168.1.2', port: 830, online: true }] },
    },
  } as any)
})

describe('ModuleConsolePage · query.device 预选设备', () => {
  it('路由携带 ?device= 时初始化选中该设备', async () => {
    const w = mountPage()
    await flushPromises()
    expect((w.vm as any).selectedDevice).toBe('192.168.1.2')
  })
})
