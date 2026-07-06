import { describe, it, expect } from 'vitest'
import { deriveOverview, type DeviceInput, type FleetInput } from '../../src/composables/useFleetOverview'

describe('deriveOverview · 车队概览派生（纯函数）', () => {
  it('空车队：全 0、收敛率 0、台账/最近皆空', () => {
    const o = deriveOverview([], {})
    expect(o.total).toBe(0)
    expect(o.online).toBe(0)
    expect(o.convergenceRate).toBe(0)
    expect(o.pendingCount).toBe(0)
    expect(o.ledger).toEqual([])
    expect(o.recent).toEqual([])
    expect(o.segments.every((s) => s.count === 0)).toBe(true)
  })

  const devices: DeviceInput[] = [
    { ip: '10.0.0.1', online: true },
    { ip: '10.0.0.2', online: true },
    { ip: '10.0.0.3', online: true },
    { ip: '10.0.0.4', online: true },
    { ip: '10.0.0.5', online: true }, // 在线但无对账记录 → unknown
    { ip: '10.0.0.6', online: false }, // 离线
  ]
  const fleet: FleetInput = {
    summary: { converged: 1, reconciling: 1, drifted: 1, error: 1 },
    devices: [
      { device_id: '10.0.0.1', outcome: 'converged', last_run: '2026-07-06T01:00:00Z' },
      { device_id: '10.0.0.2', outcome: 'reconciling', last_run: '2026-07-06T02:00:00Z' },
      { device_id: '10.0.0.3', outcome: 'drifted', last_run: '2026-07-06T03:00:00Z' },
      { device_id: '10.0.0.4', outcome: 'error', last_run: '2026-07-06T04:00:00Z' },
    ],
  }

  it('混合态：在线/离线计数 + 四态 + unknown 派生', () => {
    const o = deriveOverview(devices, fleet)
    expect(o.total).toBe(6)
    expect(o.online).toBe(5)
    expect(o.offline).toBe(1)
    expect(o.counts).toEqual({ conv: 1, recon: 1, drift: 1, error: 1, off: 1, unknown: 1 })
    expect(o.unknownCount).toBe(1)
  })

  it('收敛率 = 已收敛/总设备 取整', () => {
    expect(deriveOverview(devices, fleet).convergenceRate).toBe(17) // 1/6
  })

  it('待处理 = 收敛中 + 漂移 + 失败', () => {
    expect(deriveOverview(devices, fleet).pendingCount).toBe(3)
  })

  it('segbar 需处理段 = 漂移 + 失败（原型四段口径）', () => {
    const seg = deriveOverview(devices, fleet).segments
    const attention = seg.find((s) => s.key === 'attention')!
    expect(attention.count).toBe(2)
    expect(seg.map((s) => s.key)).toEqual(['conv', 'recon', 'attention', 'off'])
  })

  it('台账：需处理 + 离线，按严重度(失败>漂移>收敛中>离线)排序；已收敛不入台账', () => {
    const led = deriveOverview(devices, fleet).ledger
    expect(led.map((r) => `${r.ip}:${r.state}`)).toEqual([
      '10.0.0.4:error',
      '10.0.0.3:drift',
      '10.0.0.2:recon',
      '10.0.0.6:off',
    ])
    expect(led.some((r) => r.state === 'conv')).toBe(false)
  })

  it('最近对账：仅有对账记录者，按时间倒序', () => {
    const rec = deriveOverview(devices, fleet).recent
    expect(rec.map((r) => r.ip)).toEqual(['10.0.0.4', '10.0.0.3', '10.0.0.2', '10.0.0.1'])
  })

  it('离线设备结局记为 offline，且不计入 online', () => {
    const o = deriveOverview([{ ip: '1.1.1.1', online: false }], {})
    expect(o.counts.off).toBe(1)
    expect(o.online).toBe(0)
    expect(o.ledger[0].outcome).toBe('offline')
  })

  it('Go time.Time 零值 last_run 视为无有效时刻，不入最近对账', () => {
    const o = deriveOverview([{ ip: '2.2.2.2', online: true }], {
      devices: [{ device_id: '2.2.2.2', outcome: 'converged', last_run: '0001-01-01T00:00:00Z' }],
    })
    expect(o.recent).toEqual([])
    expect(o.counts.conv).toBe(1)
  })

  it('全部收敛：收敛率 100、台账空', () => {
    const o = deriveOverview(
      [
        { ip: 'a', online: true },
        { ip: 'b', online: true },
      ],
      {
        devices: [
          { device_id: 'a', outcome: 'converged', last_run: '2026-07-06T01:00:00Z' },
          { device_id: 'b', outcome: 'converged', last_run: '2026-07-06T02:00:00Z' },
        ],
      },
    )
    expect(o.convergenceRate).toBe(100)
    expect(o.ledger).toEqual([])
  })
})
