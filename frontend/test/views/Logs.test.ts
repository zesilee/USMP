import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import Logs from '../../src/views/Logs.vue'
import ElementPlus from 'element-plus'
import axios from 'axios'

vi.mock('axios')

const mockLogs = [
  { id: '1', time: '2026-05-03 10:00:00', device: '192.168.1.1', type: '接口配置', status: 'success', user: 'admin', detail: '修改 GigabitEthernet1/0/1 配置' },
  { id: '2', time: '2026-05-03 09:30:00', device: '192.168.1.2', type: 'VLAN配置', status: 'success', user: 'admin', detail: '创建 VLAN 100' },
  { id: '3', time: '2026-05-03 09:00:00', device: '192.168.1.3', type: '路由配置', status: 'failed', user: 'operator', detail: '下发静态路由失败' }
]

describe('Logs View', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.mocked(axios.get).mockResolvedValue({ data: { data: mockLogs, total: 3 } })
  })

  it('should render search input field', () => {
    const wrapper = mount(Logs, {
      global: { plugins: [ElementPlus, createPinia()] }
    })
    expect(wrapper.find('.el-input').exists()).toBe(true)
  })

  it('should render logs table', async () => {
    const wrapper = mount(Logs, {
      global: { plugins: [ElementPlus, createPinia()] }
    })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.el-table').exists()).toBe(true)
  })
})
