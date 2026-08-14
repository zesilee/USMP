import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import Sparkline from '../../src/components/common/Sparkline'
import { UiProvider } from '../../src/ui'

// Sparkline F2：数值序列 → 折线+填充区；<2 点/空 → 空态「—」。
describe('Sparkline', () => {
  it('≥2 点渲染折线与填充区（含平线 max=min 除零护栏）', () => {
    const { container, rerender } = render(
      <UiProvider>
        <Sparkline points={[1, 5, 3]} />
      </UiProvider>,
    )
    expect(container.querySelector('polyline')).toBeTruthy()
    expect(container.querySelector('polygon.fillarea')).toBeTruthy()

    rerender(
      <UiProvider>
        <Sparkline points={[4, 4, 4]} />
      </UiProvider>,
    )
    expect(container.querySelector('polyline')).toBeTruthy() // 平线不除零
  })

  it('空/单点 → 空态「—」', () => {
    const { container } = render(
      <UiProvider>
        <Sparkline points={null} />
      </UiProvider>,
    )
    expect(container.querySelector('.spark-empty')).toBeTruthy()
    expect(container.querySelector('svg')).toBeNull()
  })
})
