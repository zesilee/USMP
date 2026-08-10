import type { LogEntry } from '../types/api'
import { OUTCOME_TO_STATE, type DisplayState } from '../composables/useFleetOverview'
import { i18n } from '../i18n'

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

// 从 YANG path 派生操作类型标签（FE-26，模型驱动）：取首段模块名——带命名空间前缀时
// 前缀即模块名（vlan:vlans → vlan），否则段名本身；已知模块显示菜单标题（与左树称谓
// 一致），未知模块回退段名，空路径回退通用标签。
export function opLabelOf(path: string, moduleTitles?: Record<string, string>): string {
  const seg = (path || '').split('/').filter(Boolean)[0]?.split(':')[0] ?? ''
  if (!seg) return i18n.global.t('logs.opGeneric')
  return moduleTitles?.[seg] ?? seg
}

// 纯函数：审计记录 → 日志行。保序（后端 newest-first）；缺失字段安全降级（R08）。
export function deriveLogRows(logs: LogEntry[], moduleTitles?: Record<string, string>): LogRow[] {
  return (logs ?? []).map((l) => ({
    id: l.id ?? '',
    timestamp: l.timestamp ?? '',
    device: l.device_ip ?? '',
    path: l.path ?? '',
    opLabel: opLabelOf(l.path ?? '', moduleTitles),
    summary: l.summary ?? '',
    actor: l.actor ?? '',
    reconcileState: OUTCOME_TO_STATE[l.outcome ?? ''] ?? 'unknown',
  }))
}
