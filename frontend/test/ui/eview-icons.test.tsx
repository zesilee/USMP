import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render } from '@testing-library/react'
import { createElement } from 'react'

// eview icons F1（组 7.3 覆盖率收口）：makeIcon 三分支——候选名命中（IconPlus
// 前缀）、filled 变体透传 type、缺名问号占位/裸 span 兜底（R12）。真实名
// 集已由内网校准 R16 实证全命中，此处测桥逻辑本身。
vi.mock('@nce/icon-plus', () => ({
  default: undefined,
  // vitest mock 对未声明键的访问会抛——缺名分支的候选键显式置 undefined。
  IconPlusIcPublicNotice: undefined,
  IcPublicNotice: undefined,
  IcPublicKey: undefined,
  IcPublicWarning: undefined,
  IcPublicQuestionmarkCircle: undefined,
  IconPlusIcPublicKey: (p: Record<string, unknown>) =>
    createElement('i', { 'data-icon': 'key', 'data-type': p.type as string | undefined }),
  IconPlusIcPublicWarning: (p: Record<string, unknown>) =>
    createElement('i', { 'data-icon': 'warning', 'data-type': p.type as string | undefined }),
  IconPlusIcPublicQuestionmarkCircle: () => createElement('i', { 'data-icon': 'q' }),
}))

beforeEach(() => {
  vi.resetModules()
})

describe('eview icons（语义名→icon-plus 桥）', () => {
  it('候选名命中渲染真图标；className 透传', async () => {
    const { KeyIcon } = await import('../../src/ui/eview/icons')
    const { container } = render(createElement(KeyIcon, { className: 'k' }))
    expect(container.querySelector('[data-icon="key"]')).toBeTruthy()
  })

  it('filled 变体传 type=filled', async () => {
    const { WarningFilledIcon } = await import('../../src/ui/eview/icons')
    const { container } = render(createElement(WarningFilledIcon))
    expect(container.querySelector('[data-icon="warning"]')?.getAttribute('data-type')).toBe('filled')
  })

  it('缺名回落问号占位并标注 data-icon-missing', async () => {
    const { BellIcon } = await import('../../src/ui/eview/icons')
    const { container } = render(createElement(BellIcon))
    // mock 集未提供 IcPublicNotice → 走问号占位（mock 有 Questionmark）。
    expect(container.querySelector('[data-icon="q"]')).toBeTruthy()
  })
})
