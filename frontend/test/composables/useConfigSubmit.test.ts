import { describe, it, expect, vi, beforeEach } from 'vitest'
import { flushPromises } from '@vue/test-utils'
import { useConfigSubmit } from '../../src/composables/useConfigSubmit'
import { setConfig, getConfig, getDeviceReconcile } from '../../src/api'

vi.mock('../../src/api')

const immediate = () => Promise.resolve()
const opts = { configPath: 'huawei-vlan:vlan/vlans', listKey: 'vlans', pollIntervalMs: 0, maxPolls: 5, delay: immediate }

// 单设备 reconcile 响应：statuses[] 承载对应 path 的 outcome + last_run。
const recon = (outcome: string, lastRun: string, path = '/huawei-vlan:vlan/vlans') =>
  ({ data: { data: { statuses: [{ path, outcome, last_run: lastRun }] } } }) as any
const noStatuses = { data: { data: { statuses: [] } } } as any
const T0 = '2026-07-06T10:00:00Z'
const T1 = '2026-07-06T10:00:05Z'
const T2 = '2026-07-06T10:00:10Z'

describe('useConfigSubmit · 下发→回读→轮询对账', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(setConfig).mockResolvedValue({ data: { data: { reconciliation: { triggered: true } } } } as any)
    vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: {} } } } as any)
  })

  it('走 pushing→reading→converged，且以 force_refresh 回读', async () => {
    vi.mocked(getDeviceReconcile).mockResolvedValueOnce(noStatuses).mockResolvedValue(recon('converged', T1))
    const s = useConfigSubmit(opts)
    const seen: string[] = []
    await s.run('10.0.0.1', { id: 100 }, { onPhase: (p) => seen.push(p) })

    expect(setConfig).toHaveBeenCalledWith('10.0.0.1', opts.configPath, { vlans: [{ id: 100 }] })
    expect(getConfig).toHaveBeenCalledWith('10.0.0.1', opts.configPath, true)
    expect(s.phase.value).toBe('converged')
    expect(seen).toContain('pushing')
    expect(seen).toContain('reading')
  })

  // 关键回归：设备在下发前就已 converged（last_run=T0）。轮询首轮仍是同一条旧记录
  // （last_run 未推进）→ 不得当作本次终态；须等到 last_run 推进（T1>T0）的新一次对账。
  it('首轮命中「推送前的陈旧 converged」不误判终态，等 last_run 推进', async () => {
    vi.mocked(getDeviceReconcile)
      .mockResolvedValueOnce(recon('converged', T0)) // baseline：推送前已 converged
      .mockResolvedValueOnce(recon('converged', T0)) // 首轮：同一旧记录，last_run 未变
      .mockResolvedValue(recon('converged', T1)) // 之后：新一次对账，last_run 推进
    const s = useConfigSubmit(opts)
    await s.run('10.0.0.1', { id: 100 })
    expect(s.phase.value).toBe('converged')
    // baseline(1) + 陈旧首轮(2) + 推进终态(3)：证明没在陈旧首轮就终止
    expect(getDeviceReconcile).toHaveBeenCalledTimes(3)
  })

  it('轮询到 reconciling 时继续，直到出现推进后的终态', async () => {
    vi.mocked(getDeviceReconcile)
      .mockResolvedValueOnce(noStatuses) // baseline
      .mockResolvedValueOnce(recon('reconciling', T1))
      .mockResolvedValueOnce(recon('reconciling', T1))
      .mockResolvedValueOnce(recon('drifted', T2))
    const s = useConfigSubmit(opts)
    await s.run('10.0.0.1', { id: 100 })
    expect(getDeviceReconcile).toHaveBeenCalledTimes(4) // 1 baseline + 3 polls
    expect(s.phase.value).toBe('drifted')
  })

  it('setConfig 失败 → phase=error 且不回读/不轮询（仅 baseline 一次读）', async () => {
    vi.mocked(getDeviceReconcile).mockResolvedValue(noStatuses)
    vi.mocked(setConfig).mockRejectedValue({ message: '下发失败' })
    const s = useConfigSubmit(opts)
    await s.run('10.0.0.1', { id: 100 })
    expect(s.phase.value).toBe('error')
    expect(s.error.value).toBe('下发失败')
    expect(getConfig).not.toHaveBeenCalled()
    expect(getDeviceReconcile).toHaveBeenCalledTimes(1) // 仅 baseline，无轮询
  })

  it('回读失败不致命——setConfig 已成功，仍继续轮询对账', async () => {
    vi.mocked(getConfig).mockRejectedValue({ message: 'read fail' })
    vi.mocked(getDeviceReconcile).mockResolvedValueOnce(noStatuses).mockResolvedValue(recon('converged', T1))
    const s = useConfigSubmit(opts)
    await s.run('10.0.0.1', { id: 100 })
    expect(s.phase.value).toBe('converged')
  })

  it('轮询耗尽仍无推进终态 → timedOut，phase 停在 reading（不误报成功）', async () => {
    vi.mocked(getDeviceReconcile).mockResolvedValueOnce(noStatuses).mockResolvedValue(recon('reconciling', T1))
    const s = useConfigSubmit({ ...opts, maxPolls: 3 })
    await s.run('10.0.0.1', { id: 100 })
    expect(getDeviceReconcile).toHaveBeenCalledTimes(4) // 1 baseline + 3 polls
    expect(s.timedOut.value).toBe(true)
    expect(s.phase.value).toBe('reading')
  })

  it('reconcile 查询报错视为非终态，继续轮询', async () => {
    vi.mocked(getDeviceReconcile)
      .mockResolvedValueOnce(noStatuses) // baseline
      .mockRejectedValueOnce({ message: '404' })
      .mockResolvedValue(recon('converged', T1))
    const s = useConfigSubmit(opts)
    await s.run('10.0.0.1', { id: 100 })
    expect(s.phase.value).toBe('converged')
  })

  it('对账进行中重复 run 被忽略（in-flight 守卫，防并发竞态）', async () => {
    let release: () => void = () => {}
    const gate = new Promise<void>((r) => (release = r))
    vi.mocked(setConfig).mockImplementation(() => gate.then(() => ({ data: {} }) as any))
    vi.mocked(getDeviceReconcile).mockResolvedValueOnce(noStatuses).mockResolvedValue(recon('converged', T1))
    const s = useConfigSubmit(opts)
    const first = s.run('10.0.0.1', { id: 100 }) // baseline 读完后卡在 setConfig（pushing）
    await flushPromises() // 推进到 setConfig 挂起点（gate 未释放，phase=pushing）
    await s.run('10.0.0.1', { id: 200 }) // 守卫忽略，立即返回（不再读 baseline/不下发）
    expect(setConfig).toHaveBeenCalledTimes(1)
    release()
    await first
    expect(s.phase.value).toBe('converged')
  })

  it('baseline 读期间的并发 run 也被守卫忽略（同步 in-flight 标志）', async () => {
    let releaseBaseline: () => void = () => {}
    const gate = new Promise<void>((r) => (releaseBaseline = r))
    // 首个 getDeviceReconcile（baseline）挂起，模拟网络往返窗口；之后的 poll 返回推进终态。
    vi.mocked(getDeviceReconcile)
      .mockImplementationOnce(() => gate.then(() => noStatuses) as any)
      .mockResolvedValue(recon('converged', T1))
    const s = useConfigSubmit(opts)
    const first = s.run('10.0.0.1', { id: 100 }) // 卡在 baseline 读
    await s.run('10.0.0.1', { id: 200 }) // baseline 窗口内并发 → 守卫忽略
    expect(setConfig).not.toHaveBeenCalled() // 第二次未下发；第一次还卡在 baseline
    releaseBaseline()
    await first
    expect(setConfig).toHaveBeenCalledTimes(1) // 只有第一次下发
    expect(s.phase.value).toBe('converged')
  })

  it('reset 回到 idle', async () => {
    vi.mocked(getDeviceReconcile).mockResolvedValueOnce(noStatuses).mockResolvedValue(recon('converged', T1))
    const s = useConfigSubmit(opts)
    await s.run('10.0.0.1', { id: 100 })
    s.reset()
    expect(s.phase.value).toBe('idle')
    expect(s.timedOut.value).toBe(false)
    expect(s.error.value).toBeNull()
  })
})
