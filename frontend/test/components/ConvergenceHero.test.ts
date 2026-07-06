import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ConvergenceHero from '../../src/components/dashboard/ConvergenceHero.vue'
import { deriveOverview } from '../../src/composables/useFleetOverview'

const MIXED = deriveOverview(
  [
    { ip: '10.0.0.1', online: true },
    { ip: '10.0.0.2', online: true },
    { ip: '10.0.0.3', online: true },
    { ip: '10.0.0.4', online: true },
    { ip: '10.0.0.5', online: true },
    { ip: '10.0.0.6', online: false },
  ],
  {
    devices: [
      { device_id: '10.0.0.1', outcome: 'converged', last_run: '2026-07-06T01:00:00Z' },
      { device_id: '10.0.0.2', outcome: 'reconciling', last_run: '2026-07-06T02:00:00Z' },
      { device_id: '10.0.0.3', outcome: 'drifted', last_run: '2026-07-06T03:00:00Z' },
      { device_id: '10.0.0.4', outcome: 'error', last_run: '2026-07-06T04:00:00Z' },
    ],
  },
)

describe('ConvergenceHero · 收敛率 hero', () => {
  it('渲染收敛率百分比', () => {
    const w = mount(ConvergenceHero, { props: { overview: MIXED } })
    expect(w.find('.conv-pct').text()).toContain('17') // 1/6
  })

  it('渲染四态图例及计数', () => {
    const w = mount(ConvergenceHero, { props: { overview: MIXED } })
    const rows = w.findAll('.legend-row')
    expect(rows.length).toBe(4)
    const text = w.find('.legend').text()
    expect(text).toContain('已收敛')
    expect(text).toContain('需处理')
  })

  it('分段条只渲染 count>0 的段', () => {
    const w = mount(ConvergenceHero, { props: { overview: MIXED } })
    // conv1/recon1/attention2/off1 全 >0 → 4 段
    expect(w.findAll('.segbar span').length).toBe(4)
  })

  it('待处理>0 时展示收敛中 chip', () => {
    const w = mount(ConvergenceHero, { props: { overview: MIXED } })
    expect(w.find('.chip').classes()).toContain('recon')
  })

  it('有未对账设备时展示脚注', () => {
    const w = mount(ConvergenceHero, { props: { overview: MIXED } })
    expect(w.find('.legend-foot').exists()).toBe(true)
    expect(w.find('.legend-foot').text()).toContain('未对账')
  })

  it('空车队：收敛率 0、分段条渲染空段、无 chip、无脚注', () => {
    const empty = deriveOverview([], {})
    const w = mount(ConvergenceHero, { props: { overview: empty } })
    expect(w.find('.conv-pct').text()).toContain('0')
    expect(w.find('.s-empty').exists()).toBe(true)
    expect(w.find('.chip').exists()).toBe(false)
    expect(w.find('.legend-foot').exists()).toBe(false)
  })

  it('全部收敛：展示已收敛 chip', () => {
    const all = deriveOverview(
      [{ ip: 'a', online: true }],
      { devices: [{ device_id: 'a', outcome: 'converged', last_run: '2026-07-06T01:00:00Z' }] },
    )
    const w = mount(ConvergenceHero, { props: { overview: all } })
    expect(w.find('.conv-pct').text()).toContain('100')
    expect(w.find('.chip').classes()).toContain('conv')
  })
})
