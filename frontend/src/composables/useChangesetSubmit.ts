import { computed, ref } from 'vue'
import { commitChangeset, getConfig, getDeviceReconcile } from '../api'
import { i18n } from '../i18n'
import { useChangesetStore } from '../stores/changeset'
import { nodeUnsupportedFromEnvelope } from '../utils/nodeSupport'
import { confirmOwnershipOverride, ownershipRejectionOf } from './ownershipGate'
import {
  deriveReconcileProgress,
  outcomeToPhase,
  parseRun,
  selectStatus,
  type ReconcilePhase,
} from '../utils/reconcileProgress'

/**
 * 变更集提交编排（FE-03 攒批）：提交配置 → 批量原子下发（CS-04）→ force 回读
 * 涉及锚点 → 轮询对账结局。成功清空该设备变更集；失败如实报错、变更集原样
 * 保留（R08/§9）。归属硬锁 409 → 阻断确认 → force 重发（BR-11 口径）。
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

  const phase = ref<ReconcilePhase>('idle')
  const timedOut = ref(false)
  const error = ref('')
  const progress = computed(() => deriveReconcileProgress(phase.value))
  let running = false

  function reset() {
    phase.value = 'idle'
    timedOut.value = false
    error.value = ''
  }

  /** 提交当前设备变更集；返回 true=设备已提交（对账结局另由 phase 表达）。 */
  async function run(device: string): Promise<boolean> {
    if (running || !device) return false
    running = true
    timedOut.value = false
    error.value = ''
    try {
      const req = changeset.toRequest(device)
      if (!req.entries.length) {
        phase.value = 'idle'
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

      phase.value = 'pushing'
      let res
      try {
        res = await commitChangeset(req)
        // 归属硬锁（BR-11）：整体 409 → 阻断确认 → force 重发；取消则中止（变更集保留）。
        const rej = ownershipRejectionOf(res)
        if (rej) {
          if (!(await confirmOwnershipOverride(rej))) {
            phase.value = 'idle'
            return false
          }
          res = await commitChangeset(req, true)
        }
      } catch (e) {
        // 网络层/意外异常兜底（R08）：如实报错，变更集保留。
        error.value = e instanceof Error ? e.message : String(e)
        phase.value = 'error'
        return false
      }
      const env = (res?.data ?? {}) as { success?: boolean; message?: string }
      if (!env.success) {
        // 设备不支持（FE-24/BR-12）：结构化 reason 转友好文案——2PC 整批已被
        // 后端拒绝，原始 message（unknown-element 细节）不直接透给用户。
        error.value = nodeUnsupportedFromEnvelope(env)
          ? i18n.global.t('console.nodeUnsupportedCommit')
          : env.message || 'commit failed'
        phase.value = 'error'
        return false
      }

      // 设备已整体生效：清空变更集（即使后续回读/对账超时，提交事实不变——诚实语义）。
      changeset.clear(device)

      phase.value = 'reading'
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
            phase.value = phases.includes('error')
              ? 'error'
              : phases.includes('drifted')
                ? 'drifted'
                : 'converged'
            return true
          }
        } catch {
          /* 轮询失败继续下一轮（R08） */
        }
        await delay(pollInterval)
      }
      timedOut.value = true // 停在 reading：不误报成功（FE-03 对账超时）
      return true
    } finally {
      running = false
    }
  }

  return { phase, timedOut, error, progress, run, reset }
}
