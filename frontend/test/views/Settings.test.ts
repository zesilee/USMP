import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import Settings from '../../src/views/Settings.vue'

describe('Settings View · 只读架构事实', () => {
  const w = () => mount(Settings)

  it('展示协议连接事实（NETCONF 830 / gNMI 端口）', () => {
    const t = w().text()
    expect(t).toContain('协议连接')
    expect(t).toContain('830')
    expect(t).toContain('9339 / 9340')
  })

  it('展示缓存策略事实（TTL 30s / LRU 4096 / R03 禁用）', () => {
    const t = w().text()
    expect(t).toContain('缓存策略')
    expect(t).toContain('30s')
    expect(t).toContain('4096')
    expect(t).toContain('禁用')
    expect(t).toContain('R03')
  })

  it('两张事实卡各 4 行', () => {
    const rows = w().findAll('.set-row')
    expect(rows).toHaveLength(8)
  })

  it('诚实脚注：架构固定策略非运行时可改', () => {
    expect(w().find('.footnote').text()).toContain('非运行时可改')
  })
})
