import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import Sparkline from '../../src/components/common/Sparkline.vue'

describe('Sparkline · 负载趋势迷你图', () => {
  it('有数据渲染折线 + 填充区 SVG', () => {
    const w = mount(Sparkline, { props: { points: [3, 5, 4, 6, 5, 7] } })
    expect(w.find('svg.spark').exists()).toBe(true)
    expect(w.find('polyline').attributes('points')).toBeTruthy()
    expect(w.find('polygon.fillarea').exists()).toBe(true)
  })

  it('折线点数与输入序列一致', () => {
    const w = mount(Sparkline, { props: { points: [1, 2, 3, 4] } })
    const pts = w.find('polyline').attributes('points')!.trim().split(/\s+/)
    expect(pts).toHaveLength(4)
  })

  it('null/空/单点 → 空态占位 —（不成线不崩，R08）', () => {
    for (const p of [null, undefined, [], [5]] as any[]) {
      const w = mount(Sparkline, { props: { points: p } })
      expect(w.find('svg').exists()).toBe(false)
      expect(w.find('.spark-empty').text()).toBe('—')
    }
  })

  it('全等值序列不除零（max===min 时安全）', () => {
    const w = mount(Sparkline, { props: { points: [4, 4, 4, 4] } })
    expect(w.find('polyline').attributes('points')).toContain(',')
    expect(w.find('polyline').attributes('points')).not.toContain('NaN')
  })
})
