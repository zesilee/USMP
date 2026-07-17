import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ElMessageBox } from 'element-plus'
import { ownershipRejectionOf, confirmOwnershipOverride } from '../../src/composables/ownershipGate'

// FE-18 二期（F1）：信封 409 归属拒绝识别 + 阻断确认流（确认→force 重发 / 取消→中止）。

describe('ownershipRejectionOf · 信封 409 识别', () => {
  it('code=409 且 data.intents 为数组 → 返回拒绝详情', () => {
    const res = {
      data: { code: 409, success: false, message: '路径由业务意图管理', data: { intents: ['default/biz-100'] } },
    }
    const rej = ownershipRejectionOf(res as any)
    expect(rej).not.toBeNull()
    expect(rej!.intents).toEqual(['default/biz-100'])
    expect(rej!.message).toContain('业务意图')
  })

  it('code=0 成功信封 → null', () => {
    expect(ownershipRejectionOf({ data: { code: 0, success: true, data: {} } } as any)).toBeNull()
  })

  it('code=400 其他错误 → null（不劫持普通错误）', () => {
    expect(ownershipRejectionOf({ data: { code: 400, success: false, message: 'bad' } } as any)).toBeNull()
  })

  it('409 但缺 intents → null（防御非归属 409）', () => {
    expect(ownershipRejectionOf({ data: { code: 409, success: false, data: {} } } as any)).toBeNull()
  })

  it('空响应 → null 不抛', () => {
    expect(ownershipRejectionOf(undefined as any)).toBeNull()
    expect(ownershipRejectionOf({} as any)).toBeNull()
  })
})

describe('confirmOwnershipOverride · 阻断确认', () => {
  beforeEach(() => vi.restoreAllMocks())

  it('用户确认 → true，确认框列出认领意图', async () => {
    const spy = vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as any)
    const ok = await confirmOwnershipOverride({ intents: ['default/biz-100', 'default/biz-200'], message: '' })
    expect(ok).toBe(true)
    const [text] = spy.mock.calls[0]
    expect(String(text)).toContain('default/biz-100')
    expect(String(text)).toContain('default/biz-200')
    expect(String(text)).toContain('覆盖')
  })

  it('用户取消 → false 不抛', async () => {
    vi.spyOn(ElMessageBox, 'confirm').mockRejectedValue('cancel')
    await expect(confirmOwnershipOverride({ intents: ['default/biz-100'], message: '' })).resolves.toBe(false)
  })
})
