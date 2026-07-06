import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import FreshnessRing from '../../src/components/layout/FreshnessRing.vue'
import { RING_CIRCUMFERENCE } from '../../src/composables/useFreshness'

const CIRC = RING_CIRCUMFERENCE.toFixed(1)

describe('FreshnessRing 组件 · 由 props 渲染', () => {
  it('全新缓存(age=0)：is-fresh、剩满 TTL、环 offset=0', () => {
    const w = mount(FreshnessRing, { props: { ageSeconds: 0, ttlSeconds: 30 } })
    expect(w.find('.fresh').classes()).toContain('is-fresh')
    expect(w.text()).toContain('30s / TTL 30s')
    expect(w.find('.val').attributes('stroke-dashoffset')).toBe('0.0')
    expect(w.find('.val').attributes('stroke-dasharray')).toBe(CIRC)
  })

  it('接近过期(age=24/30)：is-soon，剩 6s', () => {
    const w = mount(FreshnessRing, { props: { ageSeconds: 24, ttlSeconds: 30 } })
    expect(w.find('.fresh').classes()).toContain('is-soon')
    expect(w.text()).toContain('6s / TTL 30s')
  })

  it('已过期(age=30/30)：is-expired，剩 0s，环填满', () => {
    const w = mount(FreshnessRing, { props: { ageSeconds: 30, ttlSeconds: 30 } })
    expect(w.find('.fresh').classes()).toContain('is-expired')
    expect(w.text()).toContain('0s / TTL 30s')
    expect(w.find('.val').attributes('stroke-dashoffset')).toBe(CIRC)
  })

  it('空态(hasData=false)：is-idle，显示「—」，环不填充', () => {
    const w = mount(FreshnessRing, { props: { hasData: false } })
    expect(w.find('.fresh').classes()).toContain('is-idle')
    expect(w.text()).toContain('—')
    expect(w.find('.val').attributes('stroke-dashoffset')).toBe(CIRC)
  })

  it('无障碍：aria-label 声明 freshness≠liveness（非设备在线状态）', () => {
    const w = mount(FreshnessRing, {
      props: { ageSeconds: 5, ttlSeconds: 30, source: 'cache' },
    })
    const label = w.find('.fresh').attributes('aria-label') || ''
    expect(label).toContain('命中缓存')
    expect(label).toContain('非设备在线状态')
  })

  it('空态 aria-label 说明暂无活跃缓存', () => {
    const w = mount(FreshnessRing, { props: { hasData: false } })
    expect(w.find('.fresh').attributes('aria-label')).toContain('暂无活跃缓存')
  })
})
