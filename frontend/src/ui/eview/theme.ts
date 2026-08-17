// EviewUI 主题接入（FA-04 / 组 3.3）：design-token（hdesign 1.1）是 CSS 变量
// 体系——基础色板 :root{--brand-05..90} + 语义层引用。定制方案（调研定案）：
// 由 USMP 主色生成 10 档 brand 色阶，在 design-token 样式之后加载本模块产出的
// 覆盖段——纯 CSS、零 less 重编译、零组件侵入。功能色/密度同一通道。
import { colorPrimary, colorSuccess, colorWarning, colorError } from '../tokens'

/** hex → [r,g,b]（仅收 #RRGGBB，非法输入回退主色蓝分量安全值）。 */
function rgb(hex: string): [number, number, number] {
  const m = /^#([0-9a-fA-F]{6})$/.exec(hex.trim())
  if (!m) return [12, 94, 166]
  const n = parseInt(m[1], 16)
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255]
}

const toHex = (v: number) => Math.round(Math.min(255, Math.max(0, v))).toString(16).padStart(2, '0')

/** 与白（ratio<0）或黑（ratio>0）线性混合。 */
function mix(base: [number, number, number], ratio: number): string {
  const target = ratio < 0 ? 255 : 0
  const t = Math.abs(ratio)
  const [r, g, b] = base.map((c) => c + (target - c) * t) as [number, number, number]
  return `#${toHex(r)}${toHex(g)}${toHex(b)}`.toUpperCase()
}

/**
 * 由主色生成 hdesign 十档色阶（50=主色本身；05..40 向白渐亮；60..90 向黑渐暗）。
 * 混合系数对齐 hdesign 默认色板的明度梯度目测比例。
 */
export function generateRamp(hex: string): Record<string, string> {
  const base = rgb(hex)
  const steps: Array<[string, number]> = [
    ['05', -0.9], ['10', -0.72], ['20', -0.54], ['30', -0.36], ['40', -0.18],
    ['50', 0], ['60', 0.2], ['70', 0.4], ['80', 0.58], ['90', 0.74],
  ]
  return Object.fromEntries(steps.map(([k, r]) => [k, mix(base, r)]))
}

/** 产出 design-token 覆盖 CSS 文本（:root 级，须在 design-token 样式之后加载）。 */
export function buildTokenOverrideCss(): string {
  const families: Array<[string, string]> = [
    ['brand', colorPrimary],
    ['mint', colorSuccess], // hdesign 语义层 --color-success: var(--mint-50)
    ['yellow', colorWarning],
    ['red', colorError],
  ]
  const lines: string[] = [':root {']
  for (const [fam, hex] of families) {
    const ramp = generateRamp(hex)
    for (const [step, val] of Object.entries(ramp)) lines.push(`  --${fam}-${step}: ${val};`)
  }
  lines.push('}')
  return lines.join('\n')
}

/** 把覆盖段注入 document（幂等；EviewUI 后端启用时由 UiProvider 调用）。 */
export function injectTokenOverride(doc: Document = document): void {
  const ID = 'usmp-token-override'
  if (doc.getElementById(ID)) return
  const style = doc.createElement('style')
  style.id = ID
  style.textContent = buildTokenOverrideCss()
  doc.head.appendChild(style)
}
