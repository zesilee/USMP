import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import ElementPlus from 'element-plus'
import Devices from '../../src/views/Devices.vue'
import { listDevices, getDeviceStatus } from '../../src/api'

// 真 Chromium 组件测试：验证 happy-dom 测不真的东西 —— Element Plus el-table
// 在真实浏览器里把设备数据渲染成可见表格行。这是 happy-dom（近似 DOM，el-table
// 虚拟渲染不落地）无法可靠断言的层次。
vi.mock('../../src/api')

const backendEnvelope = {
  data: {
    success: true,
    data: {
      devices: [
        { ip: '192.168.1.1', port: 830, online: true },
        { ip: '192.168.1.2', port: 830, online: false },
      ],
      stats: { active_connections: 1, total_connections: 2, errors: 0 },
    },
  },
}

const router = createRouter({
  history: createMemoryHistory(),
  routes: [{ path: '/', name: 'dashboard', component: {} }],
})

describe('Devices（真浏览器渲染）', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(listDevices).mockResolvedValue(backendEnvelope as any)
    vi.mocked(getDeviceStatus).mockResolvedValue({ data: { success: true, data: { running: true, connected: true } } } as any)
  })

  it('应把设备真实渲染成表格行并可见', async () => {
    const wrapper = mount(Devices, {
      global: { plugins: [ElementPlus, createPinia(), router] },
      attachTo: document.body,
    })

    // 等 store fetch 完成 + el-table 完成真实渲染
    await vi.waitFor(() => {
      expect(document.body.textContent).toContain('192.168.1.1')
    }, { timeout: 3000 })

    // 真实 DOM 里出现两台设备 + 在线/离线状态标签真实落地
    expect(document.body.textContent).toContain('192.168.1.2')
    const tags = wrapper.findAllComponents({ name: 'ElTag' })
    expect(tags.length).toBeGreaterThanOrEqual(2)

    wrapper.unmount()
  })
})
