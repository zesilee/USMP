import { describe, it, expect } from 'vitest'
import { deriveLogRows, opLabelOf } from '../../src/utils/logRows'
import type { LogEntry } from '../../src/types/api'

// FE-26：标签由路径首段模块名派生（剥命名空间前缀），已知模块用菜单标题、
// 未知模块回退段名、空路径回退通用标签——不按模块名硬编码分支。
const TITLES = { vlan: 'VLAN', ifm: '接口管理' }

describe('opLabelOf · 模型驱动派生操作类型（FE-26）', () => {
  it.each([
    ['/vlan:vlan/vlan:vlans', 'VLAN'],
    ['/ifm:ifm/ifm:interfaces', '接口管理'],
    ['vlan:vlans/vlan', 'VLAN'],
    ['/ntp/ntp-config', 'ntp'],
    ['/interfaces/interface', 'interfaces'],
    ['', '配置变更'],
  ])('%s → %s', (path, label) => {
    expect(opLabelOf(path, TITLES)).toBe(label)
  })

  it('标题映射不可用时降级为段名（R08）', () => {
    expect(opLabelOf('/vlan:vlan/vlan:vlans')).toBe('vlan')
    expect(opLabelOf('/vlan:vlan/vlan:vlans', undefined)).toBe('vlan')
  })
})

describe('deriveLogRows · 审计记录 → 日志行', () => {
  const logs: LogEntry[] = [
    { id: '3', timestamp: 't3', device_ip: '10.0.0.1', path: '/vlan:vlan/vlan:vlans', summary: 'vlans (2)', actor: 'system', outcome: 'converged', triggered: true },
    { id: '2', timestamp: 't2', device_ip: '10.0.0.2', path: '/ifm:ifm/ifm:interfaces', summary: 'interface (1)', actor: 'system', outcome: 'drifted', triggered: true },
    { id: '1', timestamp: 't1', device_ip: '10.0.0.3', path: '/route:route', summary: 'x', actor: 'system', outcome: 'unknown', triggered: false },
  ]

  it('映射 outcome→ReconcileChip 态、path→opLabel（透传标题映射）', () => {
    const rows = deriveLogRows(logs, TITLES)
    expect(rows[0]).toMatchObject({ device: '10.0.0.1', opLabel: 'VLAN', summary: 'vlans (2)', reconcileState: 'conv' })
    expect(rows[1].opLabel).toBe('接口管理')
    expect(rows[2].opLabel).toBe('route') // 未知模块回退段名
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
