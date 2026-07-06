import { describe, it, expect } from 'vitest'
import { deriveReconcileProgress, outcomeToPhase, type ReconcilePhase } from '../../src/utils/reconcileProgress'

describe('deriveReconcileProgress · 下发对账三步进度', () => {
  const stepStates = (phase: ReconcilePhase) => deriveReconcileProgress(phase).steps.map((s) => s.state)

  it('idle 三步皆等待，无结局', () => {
    const p = deriveReconcileProgress('idle')
    expect(stepStates('idle')).toEqual(['wait', 'wait', 'wait'])
    expect(p.outcome).toBeNull()
    expect(p.done).toBe(false)
  })

  it('validating/pushing/reading 逐步推进 active', () => {
    expect(stepStates('validating')).toEqual(['active', 'wait', 'wait'])
    expect(stepStates('pushing')).toEqual(['done', 'active', 'wait'])
    expect(stepStates('reading')).toEqual(['done', 'done', 'active'])
    expect(deriveReconcileProgress('reading').done).toBe(false)
  })

  it('converged → 三步完成、结局收敛、done', () => {
    const p = deriveReconcileProgress('converged')
    expect(p.steps.map((s) => s.state)).toEqual(['done', 'done', 'done'])
    expect(p.outcome).toBe('converged')
    expect(p.done).toBe(true)
  })

  it('drifted → 回读步完成但结局漂移', () => {
    const p = deriveReconcileProgress('drifted')
    expect(p.steps.map((s) => s.state)).toEqual(['done', 'done', 'done'])
    expect(p.outcome).toBe('drifted')
    expect(p.done).toBe(true)
  })

  it('error → 回读步 error、结局失败', () => {
    const p = deriveReconcileProgress('error')
    expect(p.steps[2].state).toBe('error')
    expect(p.outcome).toBe('error')
    expect(p.done).toBe(true)
  })

  it('每步都带标题与副标题', () => {
    const p = deriveReconcileProgress('idle')
    expect(p.steps.map((s) => s.key)).toEqual(['validate', 'push', 'read'])
    expect(p.steps.every((s) => s.title && s.sub)).toBe(true)
  })
})

describe('outcomeToPhase · 后端 reconcile outcome → 抽屉阶段', () => {
  it('converged/drifted/error 直接映射为终态', () => {
    expect(outcomeToPhase('converged')).toBe('converged')
    expect(outcomeToPhase('drifted')).toBe('drifted')
    expect(outcomeToPhase('error')).toBe('error')
  })

  it('reconciling/unknown/缺失 → 仍在回读中（非终态）', () => {
    expect(outcomeToPhase('reconciling')).toBe('reading')
    expect(outcomeToPhase('unknown')).toBe('reading')
    expect(outcomeToPhase(undefined)).toBe('reading')
    expect(outcomeToPhase('garbage')).toBe('reading')
  })
})
