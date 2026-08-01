import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import ChangesContentDialog from '../../src/components/config/ChangesContentDialog.vue'
import 'element-plus/dist/index.css'
import '../../src/styles/theme.scss'

// F3 真 Chromium（T07 视觉回归）：happy-dom 不做样式层叠计算，透明背景这类
// bug 只有真浏览器的 getComputedStyle 能验。断言弹窗面板背景**不透明**——
// 修复前 --el-dialog-bg-color 是未插值字符串 → background 回落 transparent。

beforeEach(() => {
  setActivePinia(createPinia())
  document.body.innerHTML = ''
})

/** 解析 rgb/rgba 的 alpha（无 alpha 视为 1；transparent → 0）。 */
function alphaOf(color: string): number {
  if (color === 'transparent') return 0
  const m = color.match(/rgba?\(([^)]+)\)/)
  if (!m) return 1
  const parts = m[1].split(',').map((s) => parseFloat(s.trim()))
  return parts.length >= 4 ? parts[3] : 1
}

describe('弹窗背景不透明（T07 回归：--el-dialog-bg-color 未插值→透明）', () => {
  it('变更内容弹窗面板背景为不透明实色，且与遮罩层可区分', async () => {
    const w = mount(ChangesContentDialog, {
      props: { visible: true, device: '10.0.0.1' },
      global: { plugins: [ElementPlus] },
      attachTo: document.body,
    })
    await new Promise((r) => setTimeout(r, 80))

    const panel = document.body.querySelector('.el-dialog') as HTMLElement
    expect(panel, '弹窗未渲染').toBeTruthy()
    const bg = getComputedStyle(panel).backgroundColor
    expect(alphaOf(bg), `弹窗背景 ${bg} 不是不透明实色`).toBe(1)

    // 遮罩层仍应是半透明（不是把 scrim 也改成实色）
    const overlay = document.body.querySelector('.el-overlay') as HTMLElement
    if (overlay) {
      const oa = alphaOf(getComputedStyle(overlay).backgroundColor)
      expect(oa, '遮罩层不应变成不透明').toBeLessThan(1)
    }
    w.unmount()
  })
})
