import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { setConfig, getConfig, getDeviceReconcile } from '../api'
import {
  deriveReconcileProgress,
  outcomeToPhase,
  parseRun,
  selectStatus,
  type ReconcilePhase,
} from '../utils/reconcileProgress'

export interface UseConfigSubmitOptions {
  configPath: string // 配置 API 路径（含 vlan:/ifm:ifm 前缀）
  listKey: string // POST body 包裹 list 的键
  pollIntervalMs?: number // 轮询 reconcile 间隔（默认 1500ms）
  maxPolls?: number // 最多轮询次数（默认 10 → 约 15s 上限）
  delay?: (ms: number) => Promise<void> // 可注入延时（测试用即时 resolve）
}

const realDelay = (ms: number) => new Promise<void>((r) => setTimeout(r, ms))
const isTerminal = (p: ReconcilePhase) => p === 'converged' || p === 'drifted' || p === 'error'

// 下发抽屉的真实对账编排：edit-config 下发 → force_refresh 回读（触发缓存回填/确认可读）
// → 轮询单设备 reconcile 结局直到收敛/漂移/失败或超时。全过程驱动 phase，供进度 UI 展示。
export function useConfigSubmit(opts: UseConfigSubmitOptions) {
  const phase = ref<ReconcilePhase>('idle')
  const timedOut = ref(false)
  const error = ref<string | null>(null)
  const progress = computed(() => deriveReconcileProgress(phase.value))

  const interval = opts.pollIntervalMs ?? 1500
  const maxPolls = opts.maxPolls ?? 10
  const delay = opts.delay ?? realDelay
  // in-flight 标志：同步置位（在任何 await 之前），使守卫免疫 baseline 读的网络往返
  // 窗口——不能靠 phase 判定（baseline await 期间 phase 仍是 idle）。R09 自安全。
  let running = false

  function reset() {
    phase.value = 'idle'
    timedOut.value = false
    error.value = null
  }

  // 调用前表单校验须已通过（§9：校验不过不提交）。run 从「下发」步开始（校验步视为已完成）。
  async function run(
    ip: string,
    item: Record<string, any>,
    hooks: { onPhase?: (p: ReconcilePhase) => void } = {},
  ) {
    // in-flight 守卫：一条对账链未结束时忽略重复 run（防并发写同一 phase 竞态 R09）。
    // UI 层提交后即切进度视图已基本互斥，此处再兜底一层，让 composable 自身可复用安全。
    // 用同步 running 标志（非 phase）——baseline await 期间 phase 仍是 idle，靠 phase 会漏守。
    if (running) return
    running = true

    const set = (p: ReconcilePhase) => {
      phase.value = p
      hooks.onPhase?.(p)
    }
    reset()

    try {
      // 0) 记录下发前该 path 的最近对账时刻 baseline。轮询只认 last_run 推进过 baseline 的
      //    「新一次」对账结局，避免把「推送前就已 converged 的旧态」误当本次下发的终态。
      let baselineRun = 0
      try {
        const b = await getDeviceReconcile(ip)
        baselineRun = parseRun(selectStatus(b.data?.data?.statuses, opts.configPath)?.last_run)
      } catch {
        /* 拿不到 baseline 视为 0（宽松）——首次下发/从未对账时任何新记录都算推进 */
      }

      // 1) 编码并下发 edit-config
      set('pushing')
      try {
        const res = await setConfig(ip, opts.configPath, { [opts.listKey]: [item] })
        // 软归属警告（FE-18/BR-11）：命中业务意图认领路径时非阻断提示，流程照常。
        const warn = (res.data as any)?.data?.ownershipWarning
        if (warn?.message) {
          ElMessage.warning(`${warn.message}（${(warn.intents || []).join('、')}）`)
        }
      } catch (e: any) {
        error.value = e?.response?.data?.message || e?.message || '下发失败'
        set('error')
        return
      }

      // 2) 强制回读实际态（绕过缓存 + 回填），失败不致命——config 已下发，对账结局以 reconcile 为准
      set('reading')
      try {
        await getConfig(ip, opts.configPath, true)
      } catch {
        /* 回读失败不阻断：继续以 reconcile 轮询为权威收敛信号 */
      }

      // 3) 轮询单设备 reconcile 结局，直到出现 last_run 推进过 baseline 的「新一次」终态或超时。
      for (let i = 0; i < maxPolls; i++) {
        try {
          const res = await getDeviceReconcile(ip)
          const st = selectStatus(res.data?.data?.statuses, opts.configPath)
          // 仅当这一次对账是本次下发触发的（last_run 已推进）才认其终态。
          if (st && parseRun(st.last_run) > baselineRun) {
            const next = outcomeToPhase(st.outcome)
            if (isTerminal(next)) {
              set(next)
              return
            }
          }
        } catch {
          /* reconcile 查询报错视为非终态，继续轮询（设备可能尚未产生对账记录） */
        }
        if (i < maxPolls - 1) await delay(interval)
      }
      // 超时仍无终态：停在 reading + 标注 timedOut，不误报成功（可去概览大盘继续观察）
      timedOut.value = true
    } finally {
      running = false
    }
  }

  return { phase, timedOut, error, progress, run, reset }
}
