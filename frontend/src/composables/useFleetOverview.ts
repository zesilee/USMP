import { ref } from 'vue'
import { listDevices, getFleetReconcile } from '../api'
import type { ReconcileOutcome } from '../types/api'
import { i18n } from '../i18n'

// 车队概览派生：把 /devices（在线态）与 /reconcile/status（对账四态聚合，PR-B1/B3 真数据）
// 合成大盘所需的收敛率、四态分段、待对账台账、最近对账。
//
// 数据边界（诚实）：
// - device_id 与 device.ip 同为设备 IP（reconcile.Request.DeviceID=ip），可直接 join。
// - status 模型只有 path/outcome/diff_count/last_error，无 was→now 值级差异；台账因此
//   展示「设备 + 结局 + 最近对账时刻」，值级 diff 属 PR-2 配置对账抽屉。
// - summary 仅含「已对账」设备；从未对账(unknown) = 在线但无对账记录，本函数据设备表派生。

/** 设备在线态（取自 /devices）。 */
export interface DeviceInput {
  ip: string
  online: boolean
}

/** 设备级对账 rollup（取自 /reconcile/status devices[]）。 */
export interface RollupInput {
  device_id?: string
  outcome?: string
  last_run?: string
}

export interface FleetInput {
  summary?: Record<string, number>
  devices?: RollupInput[]
}

/** 展示态：离线优先于对账态；在线设备映射其对账结局。 */
export type DisplayState = 'conv' | 'recon' | 'drift' | 'error' | 'off' | 'unknown'

export interface StateCounts {
  conv: number
  recon: number
  drift: number
  error: number
  off: number
  unknown: number
}

export interface LedgerRow {
  ip: string
  outcome: ReconcileOutcome | 'offline'
  state: DisplayState
  lastRun: string | null
}

export interface SegbarSegment {
  key: 'conv' | 'recon' | 'attention' | 'off'
  label: string
  count: number
  /** flex-grow 比例（= count，view 直接用）。 */
  grow: number
}

export interface Overview {
  total: number
  online: number
  offline: number
  counts: StateCounts
  /** 收敛率 = 已收敛 / 总设备，取整百分比；总数 0 时为 0。 */
  convergenceRate: number
  /** 待处理 = 收敛中 + 已漂移 + 失败（在线但未收敛）。 */
  pendingCount: number
  /** 未对账 = 在线但无对账记录。 */
  unknownCount: number
  /** segbar 四段（原型口径：需处理 = 漂移 + 失败），count>0 才渲染。 */
  segments: SegbarSegment[]
  /** 待对账台账：需处理 + 离线设备，按严重度→时间排序。 */
  ledger: LedgerRow[]
  /** 最近对账：有对账记录的设备，按时间倒序。 */
  recent: LedgerRow[]
}

// 后端 reconcile outcome → 展示态（收敛/收敛中/漂移/失败/未对账）。离线态由调用方另判。
// 导出供 deviceRows 等复用，保持「outcome→态」单一真源。
export const OUTCOME_TO_STATE: Record<string, DisplayState> = {
  converged: 'conv',
  reconciling: 'recon',
  drifted: 'drift',
  error: 'error',
  unknown: 'unknown',
}

// 台账排序严重度：失败 > 漂移 > 收敛中 > 离线（离线是「读不到」而非「配置错」）。
const STATE_SEVERITY: Record<DisplayState, number> = {
  error: 4,
  drift: 3,
  recon: 2,
  off: 1,
  conv: 0,
  unknown: 0,
}

function emptyCounts(): StateCounts {
  return { conv: 0, recon: 0, drift: 0, error: 0, off: 0, unknown: 0 }
}

/** last_run 是否为「有效时刻」（非空、非零值 0001-01-01）。导出供 deviceRows 复用。 */
export function normalizeLastRun(raw?: string): string | null {
  if (!raw) return null
  if (raw.startsWith('0001-01-01')) return null // Go time.Time 零值
  return raw
}

function lastRunMillis(row: LedgerRow): number {
  if (!row.lastRun) return 0
  const t = Date.parse(row.lastRun)
  return Number.isNaN(t) ? 0 : t
}

/**
 * 纯函数：由设备表 + 对账聚合派生完整概览。无副作用、不依赖时钟，确定性可测。
 */
export function deriveOverview(devices: DeviceInput[], fleet: FleetInput): Overview {
  const reconcileMap = new Map<string, RollupInput>()
  for (const r of fleet.devices ?? []) {
    if (r.device_id) reconcileMap.set(r.device_id, r)
  }

  const counts = emptyCounts()
  const ledger: LedgerRow[] = []
  const recent: LedgerRow[] = []

  let online = 0
  for (const d of devices) {
    let state: DisplayState
    let outcome: ReconcileOutcome | 'offline'
    let lastRun: string | null = null

    if (!d.online) {
      state = 'off'
      outcome = 'offline'
    } else {
      online++
      const rec = reconcileMap.get(d.ip)
      const oc = (rec?.outcome ?? 'unknown') as string
      state = OUTCOME_TO_STATE[oc] ?? 'unknown'
      outcome = (rec?.outcome as ReconcileOutcome) ?? 'unknown'
      lastRun = normalizeLastRun(rec?.last_run)
    }
    counts[state]++

    const row: LedgerRow = { ip: d.ip, outcome, state, lastRun }
    if (state === 'error' || state === 'drift' || state === 'recon' || state === 'off') {
      ledger.push(row)
    }
    if (lastRun) recent.push(row)
  }

  ledger.sort(
    (a, b) => STATE_SEVERITY[b.state] - STATE_SEVERITY[a.state] || lastRunMillis(b) - lastRunMillis(a),
  )
  recent.sort((a, b) => lastRunMillis(b) - lastRunMillis(a))

  const total = devices.length
  const convergenceRate = total > 0 ? Math.round((counts.conv / total) * 100) : 0
  const pendingCount = counts.recon + counts.drift + counts.error

  const t = i18n.global.t
  const segments: SegbarSegment[] = [
    { key: 'conv', label: t('common.state.conv'), count: counts.conv, grow: counts.conv },
    { key: 'recon', label: t('common.state.recon'), count: counts.recon, grow: counts.recon },
    {
      key: 'attention',
      label: t('common.state.attention'),
      count: counts.drift + counts.error,
      grow: counts.drift + counts.error,
    },
    { key: 'off', label: t('common.state.off'), count: counts.off, grow: counts.off },
  ]

  return {
    total,
    online,
    offline: total - online,
    counts,
    convergenceRate,
    pendingCount,
    unknownCount: counts.unknown,
    segments,
    ledger,
    recent,
  }
}

const EMPTY_OVERVIEW: Overview = deriveOverview([], {})

/**
 * 组合式：拉取 /devices + /reconcile/status，派生概览；任一失败降级为错误态（R08 不崩）。
 */
export function useFleetOverview() {
  const overview = ref<Overview>(EMPTY_OVERVIEW)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function load() {
    loading.value = true
    error.value = null
    try {
      const [devRes, fleetRes] = await Promise.all([listDevices(), getFleetReconcile()])
      const devices: DeviceInput[] = (devRes.data?.data?.devices ?? []).map((d) => ({
        ip: d.ip ?? '',
        online: d.online ?? false,
      }))
      overview.value = deriveOverview(devices, fleetRes.data?.data ?? {})
    } catch (e: any) {
      error.value = e?.response?.data?.message || e?.message || i18n.global.t('common.loadFailed')
      overview.value = EMPTY_OVERVIEW
    } finally {
      loading.value = false
    }
  }

  return { overview, loading, error, load }
}
