import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import ModuleConsolePage from '../../src/views/ModuleConsolePage.vue'
import { useDeviceStore } from '../../src/stores/device'
import { getYangSchema, getConfig, listDevices, getOwnership } from '../../src/api'

// 真 Chromium：el-select teleport 弹层的真实点选链路（happy-dom 伪造不了）——
// 弹层项点击 → 全局设备上下文（store）写入 → 引导空态退场、Tab 出现（FE-10）。
vi.mock('../../src/api')

// 精简 ifm 嵌套 schema：一个 list 根子节点足以派生出「接口列表」Tab。
const nestedSchema = {
  title: 'ifm',
  vendor: 'huawei',
  fields: [
    {
      path: '/ifm/interfaces',
      type: 'group',
      label: 'interfaces',
      fields: [
        {
          path: '/ifm/interfaces/interface',
          type: 'list',
          label: 'interface',
          fields: [{ path: '/ifm/interfaces/interface/name', type: 'string', label: 'name', required: true }],
        },
      ],
    },
  ],
}

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { module: 'ifm' }, query: {} }),
}))

describe('ModuleConsolePage（真浏览器）· 设备下拉写全局上下文', () => {
  let pinia: ReturnType<typeof createPinia>

  beforeEach(() => {
    pinia = createPinia()
    setActivePinia(pinia)
    vi.mocked(getYangSchema).mockResolvedValue({ data: { success: true, data: nestedSchema } } as any)
    vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: {} } } } as any)
    vi.mocked(getOwnership).mockResolvedValue({ data: { success: true, data: { claims: [] } } } as any)
    vi.mocked(listDevices).mockResolvedValue({
      data: {
        success: true,
        data: { devices: [{ ip: '192.168.1.1', port: 830, online: true }, { ip: '192.168.1.2', port: 830, online: true }] },
      },
    } as any)
  })

  it('弹层点选设备 → store 写入、空态退场、Tab 渲染', async () => {
    const wrapper = mount(ModuleConsolePage, {
      global: { plugins: [pinia, ElementPlus] },
      attachTo: document.body,
    })
    const store = useDeviceStore()

    // 初始未选设备：引导空态可见
    await vi.waitFor(() => {
      expect(document.querySelector('[data-test="select-device-empty"]')).toBeTruthy()
    }, { timeout: 3000 })

    // 打开 el-select 弹层（teleport 到 body），点选第二台设备
    ;(wrapper.element.querySelector('.el-select') as HTMLElement).click()
    await vi.waitFor(() => {
      const items = document.querySelectorAll('.el-select-dropdown__item')
      expect(items.length).toBeGreaterThanOrEqual(2)
    }, { timeout: 3000 })
    const target = [...document.querySelectorAll('.el-select-dropdown__item')].find(
      (n) => n.textContent?.includes('192.168.1.2'),
    ) as HTMLElement
    target.click()

    // 弹层点选真实落到全局上下文，空态退场、Tab 出现
    await vi.waitFor(() => {
      expect(store.selectedDeviceIp).toBe('192.168.1.2')
      expect(document.querySelector('[data-test="select-device-empty"]')).toBeFalsy()
      expect(document.querySelectorAll('.el-tabs__item').length).toBeGreaterThan(0)
    }, { timeout: 3000 })

    wrapper.unmount()
  })
})
