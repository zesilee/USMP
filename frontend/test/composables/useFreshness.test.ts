import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import {
  computeFreshness,
  liveAgeSeconds,
  useLiveFreshness,
  EXPIRING_THRESHOLD,
  RING_CIRCUMFERENCE,
} from '../../src/composables/useFreshness'
import { useFreshnessStore } from '../../src/stores/freshness'

describe('computeFreshness · 纯函数边界', () => {
  it('age=0 全新：fraction=0、剩满 TTL、不接近过期、offset=0', () => {
    const s = computeFreshness({ ageSeconds: 0, ttlSeconds: 30 })
    expect(s.fraction).toBe(0)
    expect(s.remainingSeconds).toBe(30)
    expect(s.expiringSoon).toBe(false)
    expect(s.expired).toBe(false)
    expect(s.dashOffset).toBe(0)
  })

  it('age=ttl 恰过期：fraction=1、剩 0、expired、offset=周长', () => {
    const s = computeFreshness({ ageSeconds: 30, ttlSeconds: 30 })
    expect(s.fraction).toBe(1)
    expect(s.remainingSeconds).toBe(0)
    expect(s.expired).toBe(true)
    expect(s.expiringSoon).toBe(true)
    expect(s.dashOffset).toBeCloseTo(RING_CIRCUMFERENCE, 5)
  })

  it('age 超过 ttl：fraction 夹取到 1、剩 0、expired', () => {
    const s = computeFreshness({ ageSeconds: 40, ttlSeconds: 30 })
    expect(s.fraction).toBe(1)
    expect(s.remainingSeconds).toBe(0)
    expect(s.expired).toBe(true)
  })

  it('颜色阈值：fraction≥0.8 才算接近过期', () => {
    expect(computeFreshness({ ageSeconds: 24, ttlSeconds: 30 }).expiringSoon).toBe(true) // 0.80
    expect(computeFreshness({ ageSeconds: 23, ttlSeconds: 30 }).expiringSoon).toBe(false) // 0.766
    expect(EXPIRING_THRESHOLD).toBe(0.8)
  })

  it('中段：fraction=0.5、剩 15、offset=半周长', () => {
    const s = computeFreshness({ ageSeconds: 15, ttlSeconds: 30 })
    expect(s.fraction).toBeCloseTo(0.5, 5)
    expect(s.remainingSeconds).toBe(15)
    expect(s.dashOffset).toBeCloseTo(RING_CIRCUMFERENCE / 2, 5)
  })

  it('TTL 非法(≤0)：视为已过期，环填满、剩 0', () => {
    const s = computeFreshness({ ageSeconds: 5, ttlSeconds: 0 })
    expect(s.expired).toBe(true)
    expect(s.remainingSeconds).toBe(0)
    expect(s.dashOffset).toBeCloseTo(RING_CIRCUMFERENCE, 5)
  })

  it('负 age 归零', () => {
    const s = computeFreshness({ ageSeconds: -10, ttlSeconds: 30 })
    expect(s.fraction).toBe(0)
    expect(s.remainingSeconds).toBe(30)
  })
})

describe('liveAgeSeconds · 纯函数', () => {
  it('同一时刻：等于基准年龄', () => {
    expect(liveAgeSeconds(5, 1000, 1000)).toBe(5)
  })
  it('经过 3 秒：基准 + 3', () => {
    expect(liveAgeSeconds(5, 1000, 4000)).toBe(8)
  })
  it('now 早于记录时刻：elapsed 夹取为 0', () => {
    expect(liveAgeSeconds(5, 5000, 1000)).toBe(5)
  })
  it('负基准年龄归零', () => {
    expect(liveAgeSeconds(-3, 1000, 1000)).toBe(0)
  })
})

describe('useLiveFreshness · 每秒时钟推进（真数据 + 本地 tick）', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    setActivePinia(createPinia())
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  const Harness = defineComponent({
    setup() {
      return useLiveFreshness()
    },
    template: '<div>{{ ageSeconds }}|{{ state.remainingSeconds }}|{{ hasData }}</div>',
  })

  it('挂载后随时间推进缓存年龄，剩余秒同步递减', async () => {
    const store = useFreshnessStore()
    store.record({ cache_age_seconds: 5, ttl_seconds: 30, source: 'device' })

    const wrapper = mount(Harness)
    expect(wrapper.text()).toContain('5|25|true')

    await vi.advanceTimersByTimeAsync(3000)
    await nextTick()
    expect(wrapper.text()).toContain('8|22|true')

    wrapper.unmount()
  })

  it('无缓存数据时年龄为 0、hasData=false', () => {
    const wrapper = mount(Harness)
    expect(wrapper.text()).toContain('0|30|false')
    wrapper.unmount()
  })

  it('卸载后清理定时器，不再更新（无泄漏）', async () => {
    const clearSpy = vi.spyOn(globalThis, 'clearInterval')
    const store = useFreshnessStore()
    store.record({ cache_age_seconds: 0, ttl_seconds: 30 })
    const wrapper = mount(Harness)
    wrapper.unmount()
    expect(clearSpy).toHaveBeenCalled()
    // 卸载后再推进时间不应抛错
    await vi.advanceTimersByTimeAsync(5000)
    clearSpy.mockRestore()
  })
})
