import { describe, it, expect } from 'vitest'
import { deriveLogRows, opLabelOf } from '../../src/utils/logRows'
import type { LogEntry } from '../../src/types/api'

describe('opLabelOf · 从 YANG path 派生操作类型', () => {
  it.each([
    ['/vlan:vlan/vlan:vlans', 'VLAN 配置'],
    ['/ifm:ifm/ifm:interfaces', '接口配置'],
    ['/interfaces/interface', '接口配置'],
    ['/system:system', '系统配置'],
    ['/route:route/static', '路由配置'],
    ['/unknown/thing', '配置变更'],
    ['', '配置变更'],
  ])('%s → %s', (path, label) => {
    expect(opLabelOf(path)).toBe(label)
  })
})

describe('deriveLogRows · 审计记录 → 日志行', () => {
  const logs: LogEntry[] = [
    { id: '3', timestamp: 't3', device_ip: '10.0.0.1', path: '/vlan:vlan/vlan:vlans', summary: 'vlans (2)', actor: 'system', outcome: 'converged', triggered: true },
    { id: '2', timestamp: 't2', device_ip: '10.0.0.2', path: '/ifm:ifm/ifm:interfaces', summary: 'interface (1)', actor: 'system', outcome: 'drifted', triggered: true },
    { id: '1', timestamp: 't1', device_ip: '10.0.0.3', path: '/route:route', summary: 'x', actor: 'system', outcome: 'unknown', triggered: false },
  ]

  it('映射 outcome→ReconcileChip 态、path→opLabel', () => {
    const rows = deriveLogRows(logs)
    expect(rows[0]).toMatchObject({ device: '10.0.0.1', opLabel: 'VLAN 配置', summary: 'vlans (2)', reconcileState: 'conv' })
    expect(rows[1].reconcileState).toBe('drift')
    expect(rows[2].reconcileState).toBe('unknown')
  })

  it('保序透传（后端已 newest-first）', () => {
    const rows = deriveLogRows(logs)
    expect(rows.map((r) => r.id)).toEqual(['3', '2', '1'])
  })

  it('缺失字段安全降级：outcome 缺→unknown、字段缺→空串', () => {
    const rows = deriveLogRows([{ id: '9' }])
    expect(rows[0]).toMatchObject({ id: '9', device: '', path: '', summary: '', actor: '', reconcileState: 'unknown' })
  })

  it('未知 outcome 值兜底 unknown', () => {
    const rows = deriveLogRows([{ id: '1', outcome: 'garbage' }])
    expect(rows[0].reconcileState).toBe('unknown')
  })

  it('空/异常输入降级为空数组（R08）', () => {
    expect(deriveLogRows([])).toEqual([])
    expect(deriveLogRows(null as any)).toEqual([])
    expect(deriveLogRows(undefined as any)).toEqual([])
  })
})
