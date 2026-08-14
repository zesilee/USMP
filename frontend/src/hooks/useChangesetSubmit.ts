import { useCallback, useRef, useState } from 'react'
import { commitChangeset, getConfig, getDeviceReconcile } from '../api'
import { i18n } from '../i18n'
import { useChangesetStore } from '../stores/changeset'
import { nodeUnsupportedFromEnvelope } from '../utils/nodeSupport'
import { confirmOwnershipOverride, ownershipRejectionOf } from '../composables/ownershipGate'
import {
  deriveReconcileProgress,
  outcomeToPhase,
  parseRun,
  selectStatus,
  type ReconcilePhase,
} from '../utils/reconcileProgress'

/**
 * 变更集提交编排（FE-03 攒批，React hook——语义自旧 Vue composable 逐行平移）：
 * 提交配置 → 批量原子下发（CS-04）→ force 回读涉及锚点 → 轮询对账结局。
 * 成功清空该设备变更集；失败如实报错、变更集原样保留（R08/§9）。
 * 归属硬锁 409 → 阻断确认 → force 重发（BR-11 口径）。
 */
export interface UseChangesetSubmitOptions {
  pollIntervalMs?: number
  maxPolls?: number
  /** 测试注入的延时器；缺省真实 setTimeout。 */
  delay?: (ms: number) => Promise<void>
}

export function useChangesetSubmit(opts: UseChangesetSubmitOptions = {}) {
  const pollInterval = opts.pollIntervalMs ?? 1500
  const maxPolls = opts.maxPolls ?? 10
  const delay = opts.delay ?? ((ms: number) => new Promise<void>((r) => setTimeout(r, ms)))

  const changeset = useChangesetStore()

  const [phase, setPhase] = useState<ReconcilePhase>('idle')
  const [timedOut, setTimedOut] = useState(false)
  const [error, setError] = useState('')
  const runningRef = useRef(false)
  // 终局同步引用：run() 返回后调用方立即读 phase 会拿到旧渲染闭包值——ref 兜真值。
  const phaseRef = useRef<ReconcilePhase>('idle')
  const setPhaseBoth = (p: ReconcilePhase) => {
    phaseRef.current = p
    setPhase(p)
  }

  const reset = useCallback(() => {
    setPhaseBoth('idle')
    setTimedOut(false)
    setError('')
  }, [])

  /** 提交当前设备变更集；返回 true=设备已提交（对账结局另由 phase 表达）。 */
  const run = useCallback(
    async (device: string): Promise<boolean> => {
      if (runningRef.current || !device) return false
      runningRef.current = true
      setTimedOut(false)
      setError('')
      try {
        const req = changeset.toRequest(device)
        if (!req.entries.length) {
          setPhaseBoth('idle')
          return false
        }
        const paths = [...new Set(req.entries.map((e) => e.path))]

        // 对账基线：只认推进过本次提交前 last_run 的新结局。
        const baseline: Record<string, number> = {}
        try {
          const base = await getDeviceReconcile(device)
          const statuses = base.data?.data?.statuses ?? []
          for (const p of paths) baseline[p] = parseRun(selectStatus(statuses, p)?.last_run)
        } catch {
          for (const p of paths) baseline[p] = 0
        }

        setPhaseBoth('pushing')
        let res
        try {
          res = await commitChangeset(req)
          // 归属硬锁（BR-11）：整体 409 → 阻断确认 → force 重发；取消中止（变更集保留）。
          const rej = ownershipRejectionOf(res)
          if (rej) {
            if (!(await confirmOwnershipOverride(rej))) {
              setPhaseBoth('idle')
              return false
            }
            res = await commitChangeset(req, true)
          }
        } catch (e) {
          setError(e instanceof Error ? e.message : String(e))
          setPhaseBoth('error')
          return false
        }
        const env = (res?.data ?? {}) as { success?: boolean; message?: string }
        if (!env.success) {
          // 设备不支持（FE-24/BR-12）：结构化 reason 转友好文案——2PC 整批被拒。
          setError(
            nodeUnsupportedFromEnvelope(env)
              ? i18n.global.t('console.nodeUnsupportedCommit')
              : env.message || 'commit failed',
          )
          setPhaseBoth('error')
          return false
        }

        // 设备已整体生效：清空变更集（即使后续回读/对账超时，提交事实不变——诚实语义）。
        changeset.clear(device)

        setPhaseBoth('reading')
        for (const p of paths) {
          try {
            await getConfig(device, p, true)
          } catch {
            /* 回读失败不阻断（§9）：列表下次加载自会拉取 */
          }
        }

        for (let i = 0; i < maxPolls; i++) {
          try {
            const r = await getDeviceReconcile(device)
            const statuses = r.data?.data?.statuses ?? []
            const phases: ReconcilePhase[] = []
            let allAdvanced = true
            for (const p of paths) {
              const st = selectStatus(statuses, p)
              if (!st || parseRun(st.last_run) <= (baseline[p] ?? 0)) {
                allAdvanced = false
                break
              }
              phases.push(outcomeToPhase(st.outcome))
            }
            if (allAdvanced && phases.length) {
              // 结局取最差（error > drifted > converged）：任一路径未收敛即如实呈现。
              setPhaseBoth(
                phases.includes('error') ? 'error' : phases.includes('drifted') ? 'drifted' : 'converged',
              )
              return true
            }
          } catch {
            /* 轮询失败继续下一轮（R08） */
          }
          await delay(pollInterval)
        }
        setTimedOut(true) // 停在 reading：不误报成功（FE-03 对账超时）
        return true
      } finally {
        runningRef.current = false
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [changeset, pollInterval, maxPolls],
  )

  return {
    phase,
    phaseRef,
    timedOut,
    error,
    progress: deriveReconcileProgress(phase),
    run,
    reset,
  }
}
