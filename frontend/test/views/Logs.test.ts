import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import Logs from '../../src/views/Logs.vue'
import ReconcileChip from '../../src/components/dashboard/ReconcileChip.vue'
import { getLogs } from '../../src/api'

vi.mock('../../src/api')

const logsEnvelope = {
  data: {
    success: true,
    data: {
      total: 3,
      logs: [
        { id: '3', timestamp: '2026-07-06T10:00:00Z', device_ip: '10.0.0.1', path: '/vlan:vlan/vlan:vlans', summary: 'vlans (2)', actor: 'system', outcome: 'converged', triggered: true },
        { id: '2', timestamp: '2026-07-06T09:30:00Z', device_ip: '10.0.0.2', path: '/ifm:ifm/ifm:interfaces', summary: 'interface (1)', actor: 'system', outcome: 'drifted', triggered: true },
        { id: '1', timestamp: '2026-07-06T09:00:00Z', device_ip: '10.0.0.3', path: '/route:route', summary: 'x', actor: 'admin', outcome: 'error', triggered: false },
      ],
    },
  },
}

function mountView() {
  return mount(Logs, { global: { plugins: [ElementPlus, createPinia()] } })
}

describe('Logs View · 操作日志', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(getLogs).mockResolvedValue(logsEnvelope as any)
  })

  it('渲染搜索框与日志表', async () => {
    const w = mountView()
    await flushPromises()
    expect(w.find('.el-input').exists()).toBe(true)
    expect(w.find('.el-table').exists()).toBe(true)
  })

  it('派生行：opLabel/summary/对账态', async () => {
    const w = mountView()
    await flushPromises()
    const vm = w.vm as any
    expect(vm.rows).toHaveLength(3)
    expect(vm.rows[0]).toMatchObject({ device: '10.0.0.1', opLabel: 'VLAN 配置', summary: 'vlans (2)', reconcileState: 'conv' })
    expect(vm.rows[2].reconcileState).toBe('error')
  })

  it('每行渲染对账结局 chip', async () => {
    const w = mountView()
    await flushPromises()
    expect(w.findAllComponents(ReconcileChip).length).toBe(3)
  })

  it('结局筛选：仅已漂移', async () => {
    const w = mountView()
    await flushPromises()
    const vm = w.vm as any
    vm.statusFilter = 'drift'
    await flushPromises()
    expect(vm.filteredRows).toHaveLength(1)
    expect(vm.filteredRows[0].device).toBe('10.0.0.2')
  })

  it('搜索按设备/操作人子串过滤', async () => {
    const w = mountView()
    await flushPromises()
    const vm = w.vm as any
    vm.searchKeyword = 'admin'
    await flushPromises()
    expect(vm.filteredRows.map((r: any) => r.actor)).toEqual(['admin'])
  })

  it('筛选变化回到第一页', async () => {
    const w = mountView()
    await flushPromises()
    const vm = w.vm as any
    vm.currentPage = 3
    vm.statusFilter = 'conv'
    await flushPromises()
    expect(vm.currentPage).toBe(1)
  })

  it('拉取失败降级空表（R08）', async () => {
    vi.mocked(getLogs).mockRejectedValue(new Error('logs down'))
    const w = mountView()
    await flushPromises()
    expect((w.vm as any).rows).toEqual([])
  })

  it('展示诚实脚注（值级差异待后端 / 操作人 system）', async () => {
    const w = mountView()
    await flushPromises()
    expect(w.find('.footnote').text()).toContain('was→now')
    expect(w.find('.footnote').text()).toContain('system')
  })
})
