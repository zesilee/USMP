import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useChangesetSubmit } from '../../src/composables/useChangesetSubmit'
import { useChangesetStore } from '../../src/stores/changeset'
import { commitChangeset, getConfig, getDeviceReconcile } from '../../src/api'
import { ElMessageBox } from 'element-plus'

vi.mock('../../src/api')

const DEV = '10.0.0.1'
const VLAN_PATH = '/vlan:vlan/vlan:vlans'
const noDelay = () => Promise.resolve()

function seed(n = 1) {
  const s = useChangesetStore()
  for (let i = 0; i < n; i++) {
    s.upsert(DEV, {
      op: 'update',
      path: VLAN_PATH,
      listKey: 'vlan',
      keyValue: String(10 + i),
      payload: { id: 10 + i, description: 'x' },
      cleared: [],
      baseline: null,
      label: `vlan ${10 + i}`,
    })
  }
  return s
}

const okCommit = { data: { code: 0, success: true, data: { status: 'COMMITTED', reconciliation: { triggered: true } } } }

beforeEach(() => {
  setActivePinia(createPinia())
  vi.resetAllMocks()
  vi.mocked(commitChangeset).mockResolvedValue(okCommit as any)
  vi.mocked(getConfig).mockResolvedValue({ data: { data: {} } } as any)
  vi.mocked(getDeviceReconcile)
    .mockResolvedValueOnce({ data: { data: { statuses: [] } } } as any)
    .mockResolvedValue({
      data: { data: { statuses: [{ path: 'vlan:vlan/vlan:vlans', last_run: '2026-07-31T12:00:00Z', outcome: 'converged' }] } },
    } as any)
})

describe('useChangesetSubmit · 提交编排（FE-03 攒批）', () => {
  it('成功链：commit → 清空变更集 → force 回读涉及锚点 → 轮询至 converged', async () => {
    const s = seed(2)
    const flow = useChangesetSubmit({ delay: noDelay })
    const committed = await flow.run(DEV)

    expect(committed).toBe(true)
    expect(vi.mocked(commitChangeset)).toHaveBeenCalledTimes(1)
    const req = vi.mocked(commitChangeset).mock.calls[0][0]
    expect(req.device).toBe(DEV)
    expect(req.entries).toHaveLength(2)
    expect(s.countFor(DEV)).toBe(0)
    // force 回读涉及锚点（去重后 1 个）
    expect(vi.mocked(getConfig)).toHaveBeenCalledWith(DEV, VLAN_PATH, true)
    expect(flow.phase.value).toBe('converged')
  })

  it('提交失败：error 相位、如实透出信封 message、变更集原样保留（R08/§9）', async () => {
    const s = seed(1)
    vi.mocked(commitChangeset).mockResolvedValue({
      data: { code: 502, success: false, message: '提交失败（设备已整体回退）: edit-config rejected' },
    } as any)
    const flow = useChangesetSubmit({ delay: noDelay })
    const committed = await flow.run(DEV)

    expect(committed).toBe(false)
    expect(flow.phase.value).toBe('error')
    expect(flow.error.value).toContain('整体回退')
    expect(s.countFor(DEV)).toBe(1, )
    expect(vi.mocked(getConfig)).not.toHaveBeenCalled()
  })

  it('归属硬锁 409 → 确认 → 携 force 重发；取消 → 中止且变更集保留', async () => {
    const s = seed(1)
    const rejected = { data: { code: 409, success: false, message: '路径由业务意图管理', data: { intents: ['default/biz-1'] } } }
    vi.mocked(commitChangeset).mockResolvedValueOnce(rejected as any).mockResolvedValueOnce(okCommit as any)
    const confirmSpy = vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as any)

    const flow = useChangesetSubmit({ delay: noDelay })
    expect(await flow.run(DEV)).toBe(true)
    expect(vi.mocked(commitChangeset).mock.calls[1][1]).toBe(true)
    confirmSpy.mockRestore()

    // 取消分支
    const s2 = seed(1)
    vi.mocked(commitChangeset).mockReset()
    vi.mocked(commitChangeset).mockResolvedValue(rejected as any)
    vi.spyOn(ElMessageBox, 'confirm').mockRejectedValue('cancel')
    const flow2 = useChangesetSubmit({ delay: noDelay })
    expect(await flow2.run(DEV)).toBe(false)
    expect(vi.mocked(commitChangeset)).toHaveBeenCalledTimes(1)
    expect(s2.countFor(DEV)).toBe(1)
    expect(flow2.phase.value).toBe('idle')
  })

  it('对账超时：停在 reading + timedOut，不误报成功；提交事实成立（变更集已清）', async () => {
    const s = seed(1)
    vi.mocked(getDeviceReconcile).mockReset()
    vi.mocked(getDeviceReconcile).mockResolvedValue({ data: { data: { statuses: [] } } } as any)
    const flow = useChangesetSubmit({ delay: noDelay, maxPolls: 3 })
    const committed = await flow.run(DEV)

    expect(committed).toBe(true)
    expect(flow.timedOut.value).toBe(true)
    expect(flow.phase.value).toBe('reading')
    expect(s.countFor(DEV)).toBe(0)
  })

  it('空变更集：直接返回 false、零请求', async () => {
    const flow = useChangesetSubmit({ delay: noDelay })
    expect(await flow.run(DEV)).toBe(false)
    expect(vi.mocked(commitChangeset)).not.toHaveBeenCalled()
  })

  it('回读失败不阻断：getConfig 抛错仍继续轮询至终局（§9）', async () => {
    seed(1)
    vi.mocked(getConfig).mockRejectedValue(new Error('read failed'))
    const flow = useChangesetSubmit({ delay: noDelay })
    expect(await flow.run(DEV)).toBe(true)
    expect(flow.phase.value).toBe('converged')
  })
})
