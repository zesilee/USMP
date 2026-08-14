import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { useFreshnessStore, DEFAULT_TTL_SECONDS } from '../../src/stores/freshness'
import { resetStores } from './reset'

const S = () => useFreshnessStore.getState()

describe('freshness store · 记录/兜底/重置', () => {
  beforeEach(() => {
    resetStores()
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-07-06T00:00:00Z'))
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('初始空态：hasData=false、TTL 兜底为默认', () => {
    expect(S().hasData).toBe(false)
    expect(S().ttlSeconds).toBe(DEFAULT_TTL_SECONDS)
    expect(S().recordedAt).toBe(0)
  })

  it('record 完整字段：写入年龄/TTL/来源，标记 hasData + recordedAt', () => {
    S().record({ cache_age_seconds: 12, ttl_seconds: 30, source: 'cache' })
    expect(S().ageSeconds).toBe(12)
    expect(S().ttlSeconds).toBe(30)
    expect(S().source).toBe('cache')
    expect(S().hasData).toBe(true)
    expect(S().recordedAt).toBe(Date.parse('2026-07-06T00:00:00Z'))
  })

  it('缺 TTL / TTL≤0 时兜底为默认 30s', () => {
    S().record({ cache_age_seconds: 3 })
    expect(S().ttlSeconds).toBe(DEFAULT_TTL_SECONDS)
    S().record({ cache_age_seconds: 3, ttl_seconds: 0 })
    expect(S().ttlSeconds).toBe(DEFAULT_TTL_SECONDS)
  })

  it('负 age / 缺字段安全兜底为 0 与空串', () => {
    S().record({ cache_age_seconds: -5 })
    expect(S().ageSeconds).toBe(0)
    expect(S().source).toBe('')
  })

  it('reset 清空为初始空态', () => {
    S().record({ cache_age_seconds: 9, ttl_seconds: 30, source: 'device' })
    S().reset()
    expect(S().hasData).toBe(false)
    expect(S().ageSeconds).toBe(0)
    expect(S().recordedAt).toBe(0)
    expect(S().source).toBe('')
  })
})
