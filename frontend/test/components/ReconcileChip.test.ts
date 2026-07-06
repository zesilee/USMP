import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ReconcileChip from '../../src/components/dashboard/ReconcileChip.vue'

describe('ReconcileChip · 对账四态', () => {
  const cases: Array<[any, string, string]> = [
    ['conv', '已收敛', 'conv'],
    ['recon', '收敛中', 'recon'],
    ['drift', '已漂移', 'drift'],
    ['error', '下发失败', 'error'],
    ['off', '离线', 'off'],
    ['unknown', '未对账', 'unknown'],
  ]

  it.each(cases)('state=%s → 文案与类名正确', (state, label, cls) => {
    const w = mount(ReconcileChip, { props: { state } })
    expect(w.text()).toContain(label)
    expect(w.find('.chip').classes()).toContain(cls)
  })

  it('未知 state 兜底为未对账', () => {
    const w = mount(ReconcileChip, { props: { state: 'garbage' as any } })
    expect(w.text()).toContain('未对账')
    expect(w.find('.chip').classes()).toContain('unknown')
  })
})
