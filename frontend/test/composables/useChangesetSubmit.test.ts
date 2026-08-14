import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useChangesetSubmit } from '../../src/hooks/useChangesetSubmit'
import { useChangesetStore } from '../../src/stores/changeset'
import { commitChangeset, getConfig, getDeviceReconcile } from '../../src/api'
import * as gate from '../../src/composables/ownershipGate'

// 提交编排套件（沿用换壳：Pinia+ElMessageBox → zustand+renderHook+gate mock，
// 断言语义零改动）。
vi.mock('../../src/api')

const DEV = '10.0.0.1'
const VLAN_PATH = '/vlan:vlan/vlan:vlans'
const noDelay = () => Promise.resolve()

const S = () => useChangesetStore.getState()

function seed(n = 1) {
  for (let i = 0; i < n; i++) {
    S().upsert(DEV, {
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
}

const okCommit = { data: { code: 0, success: true, data: { status: 'COMMITTED', reconciliation: { triggered: true } } } }

function mountFlow(opts: Parameters<typeof useChangesetSubmit>[0] = {}) {
  return renderHook(() => useChangesetSubmit({ delay: noDelay, ...opts }))
}

beforeEach(() => {
  useChangesetStore.setState({ byDevice: {} })
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
    seed(2)
    const { result } = mountFlow()
    let committed = false
    await act(async () => {
      committed = await result.current.run(DEV)
    })

    expect(committed).toBe(true)
    expect(vi.mocked(commitChangeset)).toHaveBeenCalledTimes(1)
    const req = vi.mocked(commitChangeset).mock.calls[0][0]
    expect(req.device).toBe(DEV)
    expect(req.entries).toHaveLength(2)
    expect(S().countFor(DEV)).toBe(0)
    // force 回读涉及锚点（去重后 1 个）
    expect(vi.mocked(getConfig)).toHaveBeenCalledWith(DEV, VLAN_PATH, true)
    expect(result.current.phase).toBe('converged')
  })

  it('提交失败：error 相位、如实透出信封 message、变更集原样保留（R08/§9）', async () => {
    seed(1)
    vi.mocked(commitChangeset).mockResolvedValue({
      data: { code: 502, success: false, message: '提交失败（设备已整体回退）: edit-config rejected' },
    } as any)
    const { result } = mountFlow()
    let committed = true
    await act(async () => {
      committed = await result.current.run(DEV)
    })

    expect(committed).toBe(false)
    expect(result.current.phase).toBe('error')
    expect(result.current.error).toContain('整体回退')
    expect(S().countFor(DEV)).toBe(1)
    expect(vi.mocked(getConfig)).not.toHaveBeenCalled()
  })

  it('commit 命中 node-unsupported：友好文案替代原始 message，变更集保留（FE-24）', async () => {
    seed(1)
    vi.mocked(commitChangeset).mockResolvedValue({
      data: { code: 500, success: false, message: 'edit-config unknown-element cards', data: { reason: 'node-unsupported' } },
    } as any)
    const { result } = mountFlow()
    await act(async () => {
      expect(await result.current.run(DEV)).toBe(false)
    })
    expect(result.current.phase).toBe('error')
    expect(result.current.error).toBe('部分配置项此设备不支持，已整体取消提交')
    expect(result.current.error).not.toContain('unknown-element')
    expect(S().countFor(DEV)).toBe(1)
  })

  it('归属硬锁 409 → 确认 → 携 force 重发；取消 → 中止且变更集保留', async () => {
    seed(1)
    const rejected = { data: { code: 409, success: false, message: '路径由业务意图管理', data: { intents: ['default/biz-1'] } } }
    vi.mocked(commitChangeset).mockResolvedValueOnce(rejected as any).mockResolvedValueOnce(okCommit as any)
    const confirmSpy = vi.spyOn(gate, 'confirmOwnershipOverride').mockResolvedValue(true)

    const { result } = mountFlow()
    await act(async () => {
      expect(await result.current.run(DEV)).toBe(true)
    })
    expect(vi.mocked(commitChangeset).mock.calls[1][1]).toBe(true)
    confirmSpy.mockRestore()

    // 取消分支
    seed(1)
    vi.mocked(commitChangeset).mockReset()
    vi.mocked(commitChangeset).mockResolvedValue(rejected as any)
    vi.spyOn(gate, 'confirmOwnershipOverride').mockResolvedValue(false)
    const { result: flow2 } = mountFlow()
    await act(async () => {
      expect(await flow2.current.run(DEV)).toBe(false)
    })
    expect(vi.mocked(commitChangeset)).toHaveBeenCalledTimes(1)
    expect(S().countFor(DEV)).toBe(1)
    expect(flow2.current.phase).toBe('idle')
  })

  it('对账超时：停在 reading + timedOut，不误报成功；提交事实成立（变更集已清）', async () => {
    seed(1)
    vi.mocked(getDeviceReconcile).mockReset()
    vi.mocked(getDeviceReconcile).mockResolvedValue({ data: { data: { statuses: [] } } } as any)
    const { result } = mountFlow({ maxPolls: 3 })
    let committed = false
    await act(async () => {
      committed = await result.current.run(DEV)
    })

    expect(committed).toBe(true)
    expect(result.current.timedOut).toBe(true)
    expect(result.current.phase).toBe('reading')
    expect(S().countFor(DEV)).toBe(0)
  })

  it('空变更集：直接返回 false、零请求', async () => {
    const { result } = mountFlow()
    await act(async () => {
      expect(await result.current.run(DEV)).toBe(false)
    })
    expect(vi.mocked(commitChangeset)).not.toHaveBeenCalled()
  })

  it('回读失败不阻断：getConfig 抛错仍继续轮询至终局（§9）', async () => {
    seed(1)
    vi.mocked(getConfig).mockRejectedValue(new Error('read failed'))
    const { result } = mountFlow()
    await act(async () => {
      expect(await result.current.run(DEV)).toBe(true)
    })
    expect(result.current.phase).toBe('converged')
  })
})
