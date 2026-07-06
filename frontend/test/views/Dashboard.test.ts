import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import Dashboard from '../../src/views/Dashboard.vue'
import ElementPlus from 'element-plus'
import { createRouter, createWebHistory } from 'vue-router'
import * as api from '../../src/api'

vi.mock('../../src/api')

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: {} },
    { path: '/config/vlan', component: {} },
    { path: '/logs', component: {} },
  ],
})

function seed(devices: any[], fleet: any) {
  ;(api.listDevices as any).mockResolvedValue({ data: { data: { devices } } })
  ;(api.getFleetReconcile as any).mockResolvedValue({ data: { data: fleet } })
}

async function mountDash() {
  const wrapper = mount(Dashboard, { global: { plugins: [router, ElementPlus] } })
  await flushPromises()
  return wrapper
}

const MIXED_DEVICES = [
  { ip: '10.0.0.1', online: true },
  { ip: '10.0.0.2', online: true },
  { ip: '10.0.0.3', online: true },
  { ip: '10.0.0.4', online: true },
  { ip: '10.0.0.5', online: true },
  { ip: '10.0.0.6', online: false },
]
const MIXED_FLEET = {
  summary: { converged: 1, reconciling: 1, drifted: 1, error: 1 },
  devices: [
    { device_id: '10.0.0.1', outcome: 'converged', last_run: '2026-07-06T01:00:00Z' },
    { device_id: '10.0.0.2', outcome: 'reconciling', last_run: '2026-07-06T02:00:00Z' },
    { device_id: '10.0.0.3', outcome: 'drifted', last_run: '2026-07-06T03:00:00Z' },
    { device_id: '10.0.0.4', outcome: 'error', last_run: '2026-07-06T04:00:00Z' },
  ],
}

describe('Dashboard View · 车队概览（真数据）', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('渲染收敛率 hero 与统计栈真数据', async () => {
    seed(MIXED_DEVICES, MIXED_FLEET)
    const w = await mountDash()
    expect(w.find('.conv-pct').text()).toContain('17') // 1/6
    const stats = w.findAll('.stat-v')
    expect(stats[0].text()).toContain('5') // online
    expect(stats[0].text()).toContain('6') // total
    expect(stats[1].text()).toContain('3') // pending
    expect(stats[2].text()).toContain('1') // unknown
  })

  it('待对账台账列出非收敛设备（4 行），已收敛不入台账', async () => {
    seed(MIXED_DEVICES, MIXED_FLEET)
    const w = await mountDash()
    const tables = w.findAll('.tbl')
    const ledgerRows = tables[0].findAll('tbody tr')
    expect(ledgerRows.length).toBe(4)
    expect(tables[0].text()).not.toContain('10.0.0.1') // converged 不在台账
    expect(tables[0].text()).toContain('10.0.0.4') // error 在台账
  })

  it('最近对账列出有记录设备', async () => {
    seed(MIXED_DEVICES, MIXED_FLEET)
    const w = await mountDash()
    const tables = w.findAll('.tbl')
    expect(tables[1].findAll('tbody tr').length).toBe(4)
  })

  it('空车队：收敛率 0 且台账空态提示暂无设备', async () => {
    seed([], { summary: {}, devices: [] })
    const w = await mountDash()
    expect(w.find('.conv-pct').text()).toContain('0')
    expect(w.text()).toContain('暂无设备')
  })

  it('加载失败：渲染错误告警且不崩', async () => {
    ;(api.listDevices as any).mockRejectedValue(new Error('网络中断'))
    ;(api.getFleetReconcile as any).mockResolvedValue({ data: { data: {} } })
    const w = await mountDash()
    expect(w.find('.load-error').exists()).toBe(true)
    expect(w.find('.conv-pct').text()).toContain('0')
  })
})
