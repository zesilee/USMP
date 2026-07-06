// 下发抽屉的对账阶段状态机。idle→validating→pushing→reading→(converged|drifted|error)。
// 前三个为编排进行时的本地阶段，后三个为后端 reconcile outcome 的终态映射。
export type ReconcilePhase =
  | 'idle'
  | 'validating'
  | 'pushing'
  | 'reading'
  | 'converged'
  | 'drifted'
  | 'error'

export type StepState = 'wait' | 'active' | 'done' | 'error'
export type FinalOutcome = 'converged' | 'drifted' | 'error'

export interface ProgressStep {
  key: 'validate' | 'push' | 'read'
  title: string
  sub: string
  state: StepState
}

export interface ReconcileProgress {
  steps: ProgressStep[]
  outcome: FinalOutcome | null
  done: boolean
}

// 三步固定文案（对齐设计原型的 reconcile 步骤）。
const STEP_META: { key: ProgressStep['key']; title: string; sub: string }[] = [
  { key: 'validate', title: '校验期望态', sub: 'YANG 约束通过' },
  { key: 'push', title: '编码并下发 edit-config', sub: 'NETCONF SSH 830 · commit' },
  { key: 'read', title: '回读实际态并对齐', sub: '缓存失效 · gNMI 确认' },
]

// 各阶段 → [validate, push, read] 三步状态；read 的终态由 outcome 决定。
const STEP_STATES: Record<ReconcilePhase, [StepState, StepState, StepState]> = {
  idle: ['wait', 'wait', 'wait'],
  validating: ['active', 'wait', 'wait'],
  pushing: ['done', 'active', 'wait'],
  reading: ['done', 'done', 'active'],
  converged: ['done', 'done', 'done'],
  drifted: ['done', 'done', 'done'],
  error: ['done', 'done', 'error'],
}

const OUTCOME_OF: Partial<Record<ReconcilePhase, FinalOutcome>> = {
  converged: 'converged',
  drifted: 'drifted',
  error: 'error',
}

export function deriveReconcileProgress(phase: ReconcilePhase): ReconcileProgress {
  const states = STEP_STATES[phase] ?? STEP_STATES.idle
  return {
    steps: STEP_META.map((m, i) => ({ ...m, state: states[i] })),
    outcome: OUTCOME_OF[phase] ?? null,
    done: phase === 'converged' || phase === 'drifted' || phase === 'error',
  }
}

// 单设备 reconcile 的单条 path 状态（取自 /devices/:ip/reconcile 的 statuses[]）。
export interface ReconcileStatusLike {
  path?: string
  outcome?: string
  last_run?: string
}

// 把 last_run 解析为毫秒时刻；空/Go 零值(0001-01-01)/非法 → 0（视为"从未对账"）。
export function parseRun(lastRun?: string | null): number {
  if (!lastRun || lastRun.startsWith('0001-01-01')) return 0
  const t = Date.parse(lastRun)
  return Number.isNaN(t) ? 0 : t
}

// 从 statuses[] 选出与目标 path 对应的状态；无精确匹配则回退到 last_run 最新的一条。
// path 归一去前导斜杠（后端 status.path = "/" + configPath）。用于按 last_run 推进判定终态。
export function selectStatus(
  statuses: ReconcileStatusLike[] | undefined | null,
  configPath: string,
): ReconcileStatusLike | null {
  const list = statuses ?? []
  if (!list.length) return null
  const norm = (p?: string) => (p ?? '').replace(/^\/+/, '')
  const target = norm(configPath)
  const matched = list.filter((s) => norm(s.path) === target)
  const pool = matched.length ? matched : list
  let best: ReconcileStatusLike | null = null
  let bestRun = -1
  for (const s of pool) {
    const r = parseRun(s.last_run)
    if (r >= bestRun) {
      bestRun = r
      best = s
    }
  }
  return best
}

// 后端 reconcile outcome → 抽屉阶段。终态直传；reconciling/unknown/缺失/未知一律视为仍在回读。
export function outcomeToPhase(outcome: string | undefined | null): ReconcilePhase {
  switch (outcome) {
    case 'converged':
      return 'converged'
    case 'drifted':
      return 'drifted'
    case 'error':
      return 'error'
    default:
      return 'reading'
  }
}
