import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import XmlViewer from '../../src/components/config/XmlViewer.vue'
import 'element-plus/dist/index.css'
import '../../src/styles/theme.scss'

// F3 真 Chromium：缩进与着色是**计算样式**，happy-dom 不做层叠计算——
// padding-left 真实像素与 token 颜色差异只有真浏览器能验（§5.6 F3 军规）。

const RAW =
  '<ifm xmlns="urn:huawei:yang:huawei-ifm"><interfaces><interface><name>GE0/0/1</name></interface></interfaces></ifm>'

describe('XmlViewer（真浏览器）· 缩进与着色实际生效', () => {
  it('层级越深左缩进像素越大，且报文横向不换行（保持报文结构可读）', async () => {
    const w = mount(XmlViewer, {
      props: { xml: RAW },
      global: { plugins: [ElementPlus] },
      attachTo: document.body,
    })
    await new Promise((r) => setTimeout(r, 60))

    const codes = Array.from(document.body.querySelectorAll('.xml-line .code')) as HTMLElement[]
    expect(codes.length).toBe(7) // 3 开 + 1 文本叶合并 + 3 闭
    const pads = codes.map((c) => parseFloat(getComputedStyle(c).paddingLeft))
    expect(pads[0]).toBe(0) // <ifm>
    expect(pads[1]).toBeGreaterThan(pads[0]) // <interfaces>
    expect(pads[2]).toBeGreaterThan(pads[1]) // <interface>

    // 报文行不折行（white-space: pre），否则长命名空间会把结构冲乱
    expect(getComputedStyle(codes[0]).whiteSpace).toBe('pre')
    w.unmount()
  })

  it('标签名/属性名/属性值三类 token 颜色互不相同（可区分）', async () => {
    const w = mount(XmlViewer, {
      props: { xml: RAW },
      global: { plugins: [ElementPlus] },
      attachTo: document.body,
    })
    await new Promise((r) => setTimeout(r, 60))

    const colorOf = (sel: string) => {
      const el = document.body.querySelector(sel) as HTMLElement
      expect(el, `${sel} 未渲染`).toBeTruthy()
      return getComputedStyle(el).color
    }
    const tag = colorOf('.tk-tag')
    const attrName = colorOf('.tk-attr-name')
    const attrValue = colorOf('.tk-attr-value')
    expect(new Set([tag, attrName, attrValue]).size).toBe(3)
    w.unmount()
  })

  it('行号不可选中：复制报文不会带上行号', async () => {
    const w = mount(XmlViewer, {
      props: { xml: RAW },
      global: { plugins: [ElementPlus] },
      attachTo: document.body,
    })
    await new Promise((r) => setTimeout(r, 60))
    const ln = document.body.querySelector('.ln') as HTMLElement
    expect(getComputedStyle(ln).userSelect).toBe('none')
    w.unmount()
  })
})
