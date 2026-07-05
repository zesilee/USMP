import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import Devices from '../../src/views/Devices.vue'
import ElementPlus from 'element-plus'
import { listDevices, getDeviceStatus } from '../../src/api'

vi.mock('../../src/api')

// 真实后端信封: { success, data: { devices: [...], stats } }
// 设备字段为后端 DeviceStatus（ip / online），name/vendor/model 由前端 ip 兜底。
const backendEnvelope = {
  data: {
    success: true,
    data: {
      devices: [
        { ip: '192.168.1.1', port: 830, online: true },
        { ip: '192.168.1.2', port: 830, online: true },
        { ip: '192.168.1.3', port: 830, online: false },
      ],
      stats: { active_connections: 2, total_connections: 3, errors: 0 },
    },
  },
}

const router = createRouter({
  history: createMemoryHistory(),
  routes: [{ path: '/', name: 'dashboard', component: {} }]
})

describe('Devices View', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(listDevices).mockResolvedValue(backendEnvelope as any)
    vi.mocked(getDeviceStatus).mockResolvedValue({ data: { success: true, data: { running: true, connected: true } } } as any)
  })

  it('should render search input field', () => {
    const wrapper = mount(Devices, {
      global: { plugins: [ElementPlus, createPinia(), router] }
    })
    expect(wrapper.find('.el-input').exists()).toBe(true)
  })

  it('should render device table', async () => {
    const wrapper = mount(Devices, {
      global: { plugins: [ElementPlus, createPinia(), router] }
    })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.el-table').exists()).toBe(true)
  })

  it('should display status badges correctly', async () => {
    const wrapper = mount(Devices, {
      global: { plugins: [ElementPlus, createPinia(), router] }
    })
    await wrapper.vm.$nextTick()
    await new Promise(resolve => setTimeout(resolve, 100))
    await wrapper.vm.$nextTick()
    const tags = wrapper.findAllComponents({ name: 'ElTag' })
    expect(tags.length).toBeGreaterThan(0)
  })
})
