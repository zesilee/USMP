import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import PlaceholderPage from '../../src/views/PlaceholderPage'
import { UiProvider } from '../../src/ui'

// 占位页（tasks 10 组前的路由完整性载体）：渲染并透出重建提示。
describe('PlaceholderPage', () => {
  it('渲染占位空态与页面名', () => {
    render(
      <UiProvider>
        <PlaceholderPage nameKey="nav.devices" />
      </UiProvider>,
    )
    expect(document.querySelector('[data-test="page-placeholder"]')).toBeTruthy()
  })
})
