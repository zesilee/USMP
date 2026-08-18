import { describe, it, expect } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { useState } from 'react'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { useRemountKey, anchorId, ANCHOR_SELECTOR } from '../../src/ui/bridge'
import { generateRamp, buildTokenOverrideCss, injectTokenOverride } from '../../src/ui/eview/theme'
import { ICON_MAP } from '../../src/ui/eview/iconMap'
import { colorPrimary } from '../../src/ui/tokens'

// 组 3 适配层核心 F1：桥接工具 / 主题色阶 / 图标映射一致性。

describe('useRemountKey（半受控兜底③档）', () => {
  function Probe() {
    const [dep, setDep] = useState('a')
    const key = useRemountKey(dep)
    return (
      <div>
        <span data-test="key">{`k:${key}`}</span>
        <button onClick={() => setDep('b')}>change</button>
        <button onClick={() => setDep((d) => d)}>same</button>
      </div>
    )
  }
  it('dep 变化 key 递增，dep 不变 key 稳定', () => {
    render(<Probe />)
    expect(screen.getByText('k:0')).toBeInTheDocument()
    fireEvent.click(screen.getByText('same'))
    expect(screen.getByText('k:0')).toBeInTheDocument()
    fireEvent.click(screen.getByText('change'))
    expect(screen.getByText('k:1')).toBeInTheDocument()
  })
})

describe('FA-05 锚点工具', () => {
  it('anchorId 映射与选择器互通；无锚点不产 id', () => {
    expect(anchorId('device-select')).toBe('dt-device-select')
    expect(anchorId(undefined)).toBeUndefined()
    expect(ANCHOR_SELECTOR('device-select')).toContain('[data-test="device-select"]')
    expect(ANCHOR_SELECTOR('device-select')).toContain('#dt-device-select')
  })
})

describe('主题色阶（FA-04 / design-token CSS 变量覆盖）', () => {
  it('50 档=主色本身，向 05 渐亮、向 90 渐暗（单调）', () => {
    const ramp = generateRamp(colorPrimary)
    expect(ramp['50'].toUpperCase()).toBe(colorPrimary.toUpperCase())
    const lum = (hex: string) => parseInt(hex.slice(1, 3), 16) + parseInt(hex.slice(3, 5), 16) + parseInt(hex.slice(5, 7), 16)
    const order = ['05', '10', '20', '30', '40', '50', '60', '70', '80', '90']
    for (let i = 1; i < order.length; i++) {
      expect(lum(ramp[order[i - 1]])).toBeGreaterThan(lum(ramp[order[i]]))
    }
  })

  it('覆盖 CSS 含四族色阶且 brand-50=主色；注入幂等', () => {
    const css = buildTokenOverrideCss()
    expect(css).toContain(`--brand-50: ${colorPrimary.toUpperCase()};`)
    for (const fam of ['mint', 'yellow', 'red']) expect(css).toContain(`--${fam}-50:`)
    injectTokenOverride(document)
    injectTokenOverride(document)
    expect(document.querySelectorAll('#usmp-token-override').length).toBe(1)
    document.getElementById('usmp-token-override')?.remove()
  })
})

describe('图标映射一致性（FA-04）', () => {
  it('ICON_MAP 键集合 = 两后端 icons 导出的语义名集合（不缺不多）', () => {
    // 组 5.3：icons 也进 @ui-backend 切换——antd 版挪 antd-backend/icons.ts
    // （as 别名重导出），eview 版 src/ui/eview/icons.tsx（makeIcon 逐名导出）。
    // 三方（ICON_MAP / antd 版 / eview 版）语义名集合须严格一致。
    const antdSrc = readFileSync(resolve(process.cwd(), 'src/ui/antd-backend/icons.ts'), 'utf8')
    const antdNames = Array.from(antdSrc.matchAll(/as (\w+Icon)\b/g)).map((m) => m[1])
    const evSrc = readFileSync(resolve(process.cwd(), 'src/ui/eview/icons.tsx'), 'utf8')
    const evNames = Array.from(evSrc.matchAll(/export const (\w+Icon)\b/g)).map((m) => m[1])
    expect(antdNames.length).toBeGreaterThan(0)
    expect(Object.keys(ICON_MAP).sort()).toEqual([...new Set(antdNames)].sort())
    expect(Object.keys(ICON_MAP).sort()).toEqual([...new Set(evNames)].sort())
  })

  it('映射目标全部为 IcPublic/IcIct 家族且近似项已标注', () => {
    for (const [sem, v] of Object.entries(ICON_MAP)) {
      expect(v.name, sem).toMatch(/^Ic(Public|Ict)/)
    }
    expect(ICON_MAP.BellIcon.approx).toBe(true)
    expect(ICON_MAP.MonitorIcon.approx).toBe(true)
    expect(ICON_MAP.WarningFilledIcon.filled).toBe(true)
  })
})
