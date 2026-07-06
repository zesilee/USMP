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
