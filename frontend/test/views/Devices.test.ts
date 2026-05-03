import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import Devices from '../../src/views/Devices.vue'
import ElementPlus from 'element-plus'
import axios from 'axios'

vi.mock('axios')

const mockDevices = [
  { id: '1', ip: '192.168.1.1', name: 'Core-Switch-01', vendor: 'H3C', model: 'S6800', status: 'online', lastSync: '2026-05-03 10:00:00' },
  { id: '2', ip: '192.168.1.2', name: 'Access-Switch-01', vendor: 'Huawei', model: 'S5735', status: 'online', lastSync: '2026-05-03 09:30:00' },
  { id: '3', ip: '192.168.1.3', name: 'Core-Switch-02', vendor: 'Cisco', model: 'N9K', status: 'offline', lastSync: '2026-05-02 18:00:00' }
]

const router = createRouter({
  history: createMemoryHistory(),
  routes: [{ path: '/', name: 'dashboard', component: {} }]
})

describe('Devices View', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(axios.get).mockResolvedValue({ data: { devices: mockDevices } })
    vi.mocked(axios.post).mockResolvedValue({ data: { success: true, message: '成功' } })
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
