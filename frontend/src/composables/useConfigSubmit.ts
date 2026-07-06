import { ref, computed } from 'vue'
import { setConfig, getConfig, getDeviceReconcile } from '../api'
import { deriveReconcileProgress, outcomeToPhase, type ReconcilePhase } from '../utils/reconcileProgress'

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
    if (phase.value !== 'idle' && !isTerminal(phase.value)) return

    const set = (p: ReconcilePhase) => {
      phase.value = p
      hooks.onPhase?.(p)
    }
    reset()

    // 1) 编码并下发 edit-config
    set('pushing')
    try {
      await setConfig(ip, opts.configPath, { [opts.listKey]: [item] })
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

    // 3) 轮询单设备 reconcile 结局直到终态或超时
    for (let i = 0; i < maxPolls; i++) {
      try {
        const res = await getDeviceReconcile(ip)
        const next = outcomeToPhase(res.data?.data?.outcome)
        if (isTerminal(next)) {
          set(next)
          return
        }
      } catch {
        /* reconcile 查询报错视为非终态，继续轮询（设备可能尚未产生对账记录） */
      }
      if (i < maxPolls - 1) await delay(interval)
    }
    // 超时仍无终态：停在 reading + 标注 timedOut，不误报成功（可去概览大盘继续观察）
    timedOut.value = true
  }

  return { phase, timedOut, error, progress, run, reset }
}
