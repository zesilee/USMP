import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import router from '../../src/router'
import Devices from '../../src/views/Devices.vue'
import ReconcileChip from '../../src/components/dashboard/ReconcileChip.vue'
import ElementPlus from 'element-plus'
import { listDevices, getFleetReconcile, getDeviceStatus } from '../../src/api'

vi.mock('../../src/api')

const devicesEnvelope = {
  data: {
    success: true,
    data: {
      devices: [
        { ip: '192.168.1.1', port: 830, online: true, role: 'DCGW' },
        { ip: '192.168.1.2', port: 830, online: true },
        { ip: '192.168.1.3', port: 830, online: false },
      ],
      stats: { active_connections: 2, total_connections: 3, errors: 0 },
    },
  },
}
const fleetEnvelope = {
  data: {
    success: true,
    data: {
      devices: [
        { device_id: '192.168.1.1', outcome: 'converged', last_run: '2026-07-06T10:00:00Z' },
        { device_id: '192.168.1.2', outcome: 'drifted', last_run: '2026-07-06T09:00:00Z' },
      ],
    },
  },
}

// 用真实路由表挂载：桩路由曾自带已删除的 name:'interface' 路由，把「查看配置」
// 跳转失效（路由不存在）掩盖成绿灯——导航目标必须以 src/router 为唯一事实源。
function mountView() {
  return mount(Devices, { global: { plugins: [ElementPlus, createPinia(), router] } })
}

describe('Devices View · 设备管理列表', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(listDevices).mockResolvedValue(devicesEnvelope as any)
    vi.mocked(getFleetReconcile).mockResolvedValue(fleetEnvelope as any)
    vi.mocked(getDeviceStatus).mockResolvedValue({ data: { success: true, data: { running: true, connected: true } } } as any)
  })

  it('渲染搜索框与设备表', async () => {
    const w = mountView()
    await flushPromises()
    expect(w.find('.el-input').exists()).toBe(true)
    expect(w.find('.el-table').exists()).toBe(true)
  })

  it('join 对账聚合派生收敛态：在线映结局、离线恒 off', async () => {
    const w = mountView()
    await flushPromises()
    const vm = w.vm as any
    const byIp = (ip: string) => vm.rows.find((r: any) => r.ip === ip)
    expect(byIp('192.168.1.1').reconcileState).toBe('conv')
    expect(byIp('192.168.1.2').reconcileState).toBe('drift')
    expect(byIp('192.168.1.3').reconcileState).toBe('off')
    expect(byIp('192.168.1.3').session).toBe('disconnected')
  })

  it('渲染收敛态 chip（ReconcileChip）每行一枚', async () => {
    const w = mountView()
    await flushPromises()
    const chips = w.findAllComponents(ReconcileChip)
    expect(chips.length).toBe(3)
  })

  it('role 列：有值渲染标签、缺省显占位（BR-14 展示）', async () => {
    const w = mountView()
    await flushPromises()
    const tags = w.findAll('[data-test="device-role"]')
    expect(tags.length).toBe(1)
    expect(tags[0].text()).toBe('DCGW')
  })

  it('会话 chip 文案随在线态', async () => {
    const w = mountView()
    await flushPromises()
    expect(w.text()).toContain('已连接')
    expect(w.text()).toContain('断开')
  })

  it('状态筛选：仅离线', async () => {
    const w = mountView()
    await flushPromises()
    const vm = w.vm as any
    vm.statusFilter = 'offline'
    await flushPromises()
    expect(vm.filteredRows).toHaveLength(1)
    expect(vm.filteredRows[0].ip).toBe('192.168.1.3')
  })

  it('搜索按 IP 过滤', async () => {
    const w = mountView()
    await flushPromises()
    const vm = w.vm as any
    vm.searchKeyword = '1.1'
    await flushPromises()
    expect(vm.filteredRows.map((r: any) => r.ip)).toEqual(['192.168.1.1'])
  })

  it('筛选变化时回到第一页（避免停在越界空页）', async () => {
    const w = mountView()
    await flushPromises()
    const vm = w.vm as any
    vm.currentPage = 3
    vm.statusFilter = 'online'
    await flushPromises()
    expect(vm.currentPage).toBe(1)
  })

  it('查看配置：跳转真实存在的模块控制台路由并携带设备 IP（回归：点击无反应）', async () => {
    const w = mountView()
    await flushPromises()
    const viewBtn = w.findAll('.el-button').find((b) => b.text() === '查看配置')
    expect(viewBtn).toBeTruthy()
    await viewBtn!.trigger('click')
    // 目标路由组件懒加载，导航跨事件循环 tick 落定，轮询等待而非 flushPromises
    await vi.waitFor(() => expect(router.currentRoute.value.path).toBe('/module/ifm'))
    expect(router.currentRoute.value.query.device).toBe('192.168.1.1')
  })

  it('对账聚合拉取失败不阻断设备表（收敛态降级）', async () => {
    vi.mocked(getFleetReconcile).mockRejectedValue(new Error('reconcile down'))
    const w = mountView()
    await flushPromises()
    const vm = w.vm as any
    expect(vm.rows).toHaveLength(3) // 设备仍在
    expect(vm.rows.find((r: any) => r.ip === '192.168.1.1').reconcileState).toBe('unknown') // 在线无记录
  })
})
