import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useConfigSubmit } from '../../src/composables/useConfigSubmit'
import { setConfig, getConfig, getDeviceReconcile } from '../../src/api'

vi.mock('../../src/api')

const immediate = () => Promise.resolve()
const opts = { configPath: 'huawei-vlan:vlan/vlans', listKey: 'vlans', pollIntervalMs: 0, maxPolls: 5, delay: immediate }

describe('useConfigSubmit · 下发→回读→轮询对账', () => {
  beforeEach(() => {
    vi.clearAllMocks() // 清调用计数（默认无 clearMocks 配置，否则跨用例累积）
    vi.mocked(setConfig).mockResolvedValue({ data: { data: { reconciliation: { triggered: true } } } } as any)
    vi.mocked(getConfig).mockResolvedValue({ data: { data: { data: {} } } } as any)
  })

  it('走 pushing→reading→converged，且以 force_refresh 回读', async () => {
    vi.mocked(getDeviceReconcile).mockResolvedValue({ data: { data: { outcome: 'converged' } } } as any)
    const s = useConfigSubmit(opts)
    const seen: string[] = []
    await s.run('10.0.0.1', { id: 100 }, { onPhase: (p) => seen.push(p) })

    expect(setConfig).toHaveBeenCalledWith('10.0.0.1', opts.configPath, { vlans: [{ id: 100 }] })
    expect(getConfig).toHaveBeenCalledWith('10.0.0.1', opts.configPath, true) // 强制回读
    expect(s.phase.value).toBe('converged')
    expect(s.progress.value.outcome).toBe('converged')
    expect(seen).toContain('pushing')
    expect(seen).toContain('reading')
  })

  it('轮询到 reconciling 时继续，直到出现终态', async () => {
    vi.mocked(getDeviceReconcile)
      .mockResolvedValueOnce({ data: { data: { outcome: 'reconciling' } } } as any)
      .mockResolvedValueOnce({ data: { data: { outcome: 'reconciling' } } } as any)
      .mockResolvedValueOnce({ data: { data: { outcome: 'drifted' } } } as any)
    const s = useConfigSubmit(opts)
    await s.run('10.0.0.1', { id: 100 })
    expect(getDeviceReconcile).toHaveBeenCalledTimes(3)
    expect(s.phase.value).toBe('drifted')
  })

  it('setConfig 失败 → phase=error 且不回读/不轮询', async () => {
    vi.mocked(setConfig).mockRejectedValue({ message: '下发失败' })
    const s = useConfigSubmit(opts)
    await s.run('10.0.0.1', { id: 100 })
    expect(s.phase.value).toBe('error')
    expect(s.error.value).toBe('下发失败')
    expect(getConfig).not.toHaveBeenCalled()
    expect(getDeviceReconcile).not.toHaveBeenCalled()
  })

  it('回读失败不致命——setConfig 已成功，仍继续轮询对账', async () => {
    vi.mocked(getConfig).mockRejectedValue({ message: 'read fail' })
    vi.mocked(getDeviceReconcile).mockResolvedValue({ data: { data: { outcome: 'converged' } } } as any)
    const s = useConfigSubmit(opts)
    await s.run('10.0.0.1', { id: 100 })
    expect(s.phase.value).toBe('converged')
  })

  it('轮询耗尽仍无终态 → timedOut，phase 停在 reading（不误报成功）', async () => {
    vi.mocked(getDeviceReconcile).mockResolvedValue({ data: { data: { outcome: 'reconciling' } } } as any)
    const s = useConfigSubmit({ ...opts, maxPolls: 3 })
    await s.run('10.0.0.1', { id: 100 })
    expect(getDeviceReconcile).toHaveBeenCalledTimes(3)
    expect(s.timedOut.value).toBe(true)
    expect(s.phase.value).toBe('reading')
  })

  it('reconcile 查询报错视为非终态，继续轮询', async () => {
    vi.mocked(getDeviceReconcile)
      .mockRejectedValueOnce({ message: '404' })
      .mockResolvedValueOnce({ data: { data: { outcome: 'converged' } } } as any)
    const s = useConfigSubmit(opts)
    await s.run('10.0.0.1', { id: 100 })
    expect(s.phase.value).toBe('converged')
  })

  it('对账进行中重复 run 被忽略（in-flight 守卫，防并发竞态）', async () => {
    let release: () => void = () => {}
    const gate = new Promise<void>((r) => (release = r))
    vi.mocked(setConfig).mockImplementation(() => gate.then(() => ({ data: {} }) as any))
    vi.mocked(getDeviceReconcile).mockResolvedValue({ data: { data: { outcome: 'converged' } } } as any)
    const s = useConfigSubmit(opts)
    const first = s.run('10.0.0.1', { id: 100 }) // 卡在 setConfig（pushing）
    await Promise.resolve()
    await s.run('10.0.0.1', { id: 200 }) // 应被守卫忽略，立即返回
    expect(setConfig).toHaveBeenCalledTimes(1)
    release()
    await first
    expect(s.phase.value).toBe('converged')
  })

  it('reset 回到 idle', async () => {
    vi.mocked(getDeviceReconcile).mockResolvedValue({ data: { data: { outcome: 'converged' } } } as any)
    const s = useConfigSubmit(opts)
    await s.run('10.0.0.1', { id: 100 })
    s.reset()
    expect(s.phase.value).toBe('idle')
    expect(s.timedOut.value).toBe(false)
    expect(s.error.value).toBeNull()
  })
})
