import type { Device } from '../stores/device'
import { OUTCOME_TO_STATE, type DisplayState, type FleetInput, type RollupInput } from '../composables/useFleetOverview'

// 设备管理列表的单行：设备事实 + 会话态 + 对账态（真数据，join /reconcile/status）。
export interface DeviceRow {
  id: string
  ip: string
  name: string
  vendor: string
  model: string
  vendorModel: string // 「厂商 · 型号」合并展示
  online: boolean
  session: 'connected' | 'disconnected'
  reconcileState: DisplayState // 复用 ReconcileChip 六态（离线优先）
  lastSync: string
  load: number[] | null // 负载时序：无 gNMI 遥测端点时为 null（Sparkline 显 —）
}

// 纯函数：把设备表（/devices 派生）与对账聚合（/reconcile/status devices[]）合成列表行。
// 离线优先于对账态；在线设备映射其结局，无记录 → unknown。确定性、可测、无副作用。
export function deriveDeviceRows(devices: Device[], fleet: FleetInput): DeviceRow[] {
  const reconcileMap = new Map<string, RollupInput>()
  for (const r of fleet?.devices ?? []) {
    if (r.device_id) reconcileMap.set(r.device_id, r)
  }
  return (devices ?? []).map((d) => {
    const online = d.status === 'online'
    let reconcileState: DisplayState
    if (!online) {
      reconcileState = 'off'
    } else {
      const oc = (reconcileMap.get(d.ip)?.outcome ?? 'unknown') as string
      reconcileState = OUTCOME_TO_STATE[oc] ?? 'unknown'
    }
    return {
      id: d.id,
      ip: d.ip,
      name: d.name,
      vendor: d.vendor,
      model: d.model,
      vendorModel: [d.vendor, d.model].filter(Boolean).join(' · '),
      online,
      session: online ? 'connected' : 'disconnected',
      reconcileState,
      lastSync: d.lastSync,
      load: null,
    }
  })
}
