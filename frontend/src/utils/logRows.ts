import type { LogEntry } from '../types/api'
import { OUTCOME_TO_STATE, type DisplayState } from '../composables/useFleetOverview'

// 操作日志单行：审计事实 + 当前对账态（由 /logs 的 LogEntry 派生）。
export interface LogRow {
  id: string
  timestamp: string
  device: string
  path: string
  opLabel: string // 从 YANG path 派生的操作类型
  summary: string // 提交摘要（诚实：非值级 was→now，后端无值级历史）
  actor: string // 无鉴权来源，恒 "system"
  reconcileState: DisplayState // 当前对账结局（live-join）→ ReconcileChip 态
}

// 从 YANG path 派生一个可读的操作类型标签。
export function opLabelOf(path: string): string {
  const p = (path || '').toLowerCase()
  if (p.includes('vlan')) return 'VLAN 配置'
  if (p.includes('ifm') || p.includes('interface')) return '接口配置'
  if (p.includes('system')) return '系统配置'
  if (p.includes('route')) return '路由配置'
  return '配置变更'
}

// 纯函数：审计记录 → 日志行。保序（后端 newest-first）；缺失字段安全降级（R08）。
export function deriveLogRows(logs: LogEntry[]): LogRow[] {
  return (logs ?? []).map((l) => ({
    id: l.id ?? '',
    timestamp: l.timestamp ?? '',
    device: l.device_ip ?? '',
    path: l.path ?? '',
    opLabel: opLabelOf(l.path ?? ''),
    summary: l.summary ?? '',
    actor: l.actor ?? '',
    reconcileState: OUTCOME_TO_STATE[l.outcome ?? ''] ?? 'unknown',
  }))
}
