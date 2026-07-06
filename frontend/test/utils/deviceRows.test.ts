import { describe, it, expect } from 'vitest'
import { deriveDeviceRows } from '../../src/utils/deviceRows'
import type { Device } from '../../src/stores/device'

const dev = (over: Partial<Device>): Device => ({
  id: over.ip ?? '0', ip: over.ip ?? '0', name: over.name ?? '', vendor: over.vendor ?? '',
  model: over.model ?? '', status: over.status ?? 'online', lastSync: over.lastSync ?? '',
  ...over,
})

describe('deriveDeviceRows · 设备表 + 对账聚合 join', () => {
  const devices: Device[] = [
    dev({ ip: '10.0.0.1', name: 'Core-01', vendor: 'Huawei', model: 'CE6881', status: 'online' }),
    dev({ ip: '10.0.2.13', name: 'Acc-03', vendor: 'H3C', model: 'S6520', status: 'online' }),
    dev({ ip: '10.0.2.21', name: 'Acc-11', vendor: 'Cisco', model: 'C9300', status: 'offline' }),
  ]
  const fleet = {
    devices: [
      { device_id: '10.0.0.1', outcome: 'converged', last_run: '2026-07-06T10:00:00Z' },
      { device_id: '10.0.2.13', outcome: 'drifted', last_run: '2026-07-06T09:00:00Z' },
    ],
  }

  it('在线设备映射对账结局，离线设备恒为 off', () => {
    const rows = deriveDeviceRows(devices, fleet)
    expect(rows.find((r) => r.ip === '10.0.0.1')!.reconcileState).toBe('conv')
    expect(rows.find((r) => r.ip === '10.0.2.13')!.reconcileState).toBe('drift')
    expect(rows.find((r) => r.ip === '10.0.2.21')!.reconcileState).toBe('off') // 离线优先
  })

  it('在线但无对账记录 → unknown（从未对账）', () => {
    const rows = deriveDeviceRows([dev({ ip: '10.0.0.9', status: 'online' })], fleet)
    expect(rows[0].reconcileState).toBe('unknown')
  })

  it('session 由在线态派生（connected/disconnected）', () => {
    const rows = deriveDeviceRows(devices, fleet)
    expect(rows.find((r) => r.ip === '10.0.0.1')!.session).toBe('connected')
    expect(rows.find((r) => r.ip === '10.0.2.21')!.session).toBe('disconnected')
  })

  it('vendorModel 合并厂商·型号，缺失项不留空点', () => {
    const rows = deriveDeviceRows(
      [dev({ ip: '1', vendor: 'Huawei', model: 'CE6881' }), dev({ ip: '2', vendor: 'H3C', model: '' })],
      { devices: [] },
    )
    expect(rows[0].vendorModel).toBe('Huawei · CE6881')
    expect(rows[1].vendorModel).toBe('H3C') // 无型号不产生「H3C · 」
  })

  it('负载 load 一期为 null（无 gNMI 遥测端点，诚实占位）', () => {
    const rows = deriveDeviceRows(devices, fleet)
    expect(rows.every((r) => r.load === null)).toBe(true)
  })

  it('透传 id/ip/name/lastSync 供表格与操作列使用', () => {
    const rows = deriveDeviceRows([dev({ ip: '10.0.0.1', name: 'Core-01', lastSync: 't1' })], fleet)
    expect(rows[0]).toMatchObject({ id: '10.0.0.1', ip: '10.0.0.1', name: 'Core-01', lastSync: 't1' })
  })

  it('空/异常输入安全降级（R08）', () => {
    expect(deriveDeviceRows([], {})).toEqual([])
    expect(deriveDeviceRows(null as any, {})).toEqual([])
    expect(deriveDeviceRows(devices, {})).toHaveLength(3) // fleet 缺 devices 不崩，全 unknown/off
  })

  it('未知 outcome 值兜底为 unknown（不崩）', () => {
    const rows = deriveDeviceRows([dev({ ip: '10.0.0.1', status: 'online' })], {
      devices: [{ device_id: '10.0.0.1', outcome: 'garbage' }],
    })
    expect(rows[0].reconcileState).toBe('unknown')
  })
})
