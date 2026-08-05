import { describe, it, expect } from 'vitest'
import { nodeUnsupportedFromEnvelope, nodeUnsupportedFromError } from '../../src/utils/nodeSupport'

// F1（FE-24）：设备不支持节点判定——一律以响应体结构化 reason 为准（D5），
// 禁止错误文案字符串匹配。信封为 HTTP 200 统一格式（axios 不 reject）。

describe('nodeUnsupportedFromEnvelope · 信封判定（FE-24）', () => {
  it('success=false 且 data.reason=node-unsupported → true', () => {
    expect(
      nodeUnsupportedFromEnvelope({
        code: 500,
        success: false,
        message: 'unsupported node on device',
        data: { reason: 'node-unsupported' },
      }),
    ).toBe(true)
  })

  it('success=true 时即使 data 带 reason 也不判定（成功响应不劫持）', () => {
    expect(
      nodeUnsupportedFromEnvelope({ code: 0, success: true, data: { reason: 'node-unsupported' } }),
    ).toBe(false)
  })

  it('普通错误信封（无 reason）→ false（负路径：设备离线等不误转）', () => {
    expect(nodeUnsupportedFromEnvelope({ code: 502, success: false, message: '设备离线' })).toBe(false)
  })

  it('reason 为其他值 → false（不做文案/前缀猜测）', () => {
    expect(
      nodeUnsupportedFromEnvelope({ code: 500, success: false, data: { reason: 'timeout' } }),
    ).toBe(false)
  })

  it('null/undefined/畸形体 → false 不抛（R08）', () => {
    expect(nodeUnsupportedFromEnvelope(null)).toBe(false)
    expect(nodeUnsupportedFromEnvelope(undefined)).toBe(false)
    expect(nodeUnsupportedFromEnvelope('oops')).toBe(false)
    expect(nodeUnsupportedFromEnvelope({ data: null })).toBe(false)
  })
})

describe('nodeUnsupportedFromError · axios 错误对象判定（FE-24）', () => {
  it('e.response.data.data.reason 命中 → true', () => {
    expect(
      nodeUnsupportedFromError({
        response: { data: { code: 500, success: false, data: { reason: 'node-unsupported' } } },
      }),
    ).toBe(true)
  })

  it('普通 Error / 无 response / 畸形体 → false 不抛（R08）', () => {
    expect(nodeUnsupportedFromError(new Error('network down'))).toBe(false)
    expect(nodeUnsupportedFromError({ response: { data: { message: '设备离线' } } })).toBe(false)
    expect(nodeUnsupportedFromError(undefined)).toBe(false)
  })
})
