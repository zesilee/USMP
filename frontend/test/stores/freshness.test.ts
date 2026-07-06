import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useFreshnessStore, DEFAULT_TTL_SECONDS } from '../../src/stores/freshness'

describe('freshness store · 记录/兜底/重置', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-07-06T00:00:00Z'))
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('初始空态：hasData=false、TTL 兜底为默认', () => {
    const s = useFreshnessStore()
    expect(s.hasData).toBe(false)
    expect(s.ttlSeconds).toBe(DEFAULT_TTL_SECONDS)
    expect(s.recordedAt).toBe(0)
  })

  it('record 完整字段：写入年龄/TTL/来源，标记 hasData + recordedAt', () => {
    const s = useFreshnessStore()
    s.record({ cache_age_seconds: 12, ttl_seconds: 30, source: 'cache' })
    expect(s.ageSeconds).toBe(12)
    expect(s.ttlSeconds).toBe(30)
    expect(s.source).toBe('cache')
    expect(s.hasData).toBe(true)
    expect(s.recordedAt).toBe(Date.parse('2026-07-06T00:00:00Z'))
  })

  it('缺 TTL / TTL≤0 时兜底为默认 30s', () => {
    const s = useFreshnessStore()
    s.record({ cache_age_seconds: 3 })
    expect(s.ttlSeconds).toBe(DEFAULT_TTL_SECONDS)
    s.record({ cache_age_seconds: 3, ttl_seconds: 0 })
    expect(s.ttlSeconds).toBe(DEFAULT_TTL_SECONDS)
  })

  it('负 age / 缺字段安全兜底为 0 与空串', () => {
    const s = useFreshnessStore()
    s.record({ cache_age_seconds: -5 })
    expect(s.ageSeconds).toBe(0)
    expect(s.source).toBe('')
  })

  it('reset 清空为初始空态', () => {
    const s = useFreshnessStore()
    s.record({ cache_age_seconds: 9, ttl_seconds: 30, source: 'device' })
    s.reset()
    expect(s.hasData).toBe(false)
    expect(s.ageSeconds).toBe(0)
    expect(s.recordedAt).toBe(0)
    expect(s.source).toBe('')
  })
})
