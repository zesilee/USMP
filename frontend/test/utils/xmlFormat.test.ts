import { describe, it, expect } from 'vitest'
import { formatXml, type XmlLine } from '../../src/utils/xmlFormat'

/** 测试可读性辅助：把结构化行还原为 "缩进+文本" 字符串数组。 */
const render = (lines: XmlLine[]): string[] =>
  lines.map((l) => '  '.repeat(l.indent) + l.tokens.map((t) => t.text).join(''))

describe('formatXml · 报文格式化（FE-23 试运行可读性）', () => {
  it('嵌套元素逐层缩进，每个标签独占一行', () => {
    const raw = '<ifm xmlns="urn:huawei:yang:huawei-ifm"><interfaces><interface></interface></interfaces></ifm>'
    expect(render(formatXml(raw))).toEqual([
      '<ifm xmlns="urn:huawei:yang:huawei-ifm">',
      '  <interfaces>',
      '    <interface>',
      '    </interface>',
      '  </interfaces>',
      '</ifm>',
    ])
  })

  it('纯文本叶合并为单行（<id>10</id> 不拆三行）', () => {
    const raw = '<vlans><vlan><id>10</id><description>mgmt vlan</description></vlan></vlans>'
    expect(render(formatXml(raw))).toEqual([
      '<vlans>',
      '  <vlan>',
      '    <id>10</id>',
      '    <description>mgmt vlan</description>',
      '  </vlan>',
      '</vlans>',
    ])
  })

  it('自闭合标签与属性原样保留（叶级删除报文形态）', () => {
    const raw = '<vlan><id>5</id><description nc:operation="delete" xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0"/></vlan>'
    expect(render(formatXml(raw))).toEqual([
      '<vlan>',
      '  <id>5</id>',
      '  <description nc:operation="delete" xmlns:nc="urn:ietf:params:xml:ns:netconf:base:1.0"/>',
      '</vlan>',
    ])
  })

  it('属性值内含 > 不误切标签（tokenizer 尊重引号）', () => {
    const raw = '<a b="x>y"><c>1</c></a>'
    expect(render(formatXml(raw))).toEqual(['<a b="x>y">', '  <c>1</c>', '</a>'])
  })

  it('XML 声明与注释各自成行且不影响缩进层级', () => {
    const raw = '<?xml version="1.0"?><!-- note --><a><b>1</b></a>'
    expect(render(formatXml(raw))).toEqual(['<?xml version="1.0"?>', '<!-- note -->', '<a>', '  <b>1</b>', '</a>'])
  })

  it('已带换行/缩进的输入被归一（不叠加原有空白）', () => {
    const raw = '<a>\n   <b>1</b>\n</a>'
    expect(render(formatXml(raw))).toEqual(['<a>', '  <b>1</b>', '</a>'])
  })

  it('空/非 XML 输入安全降级：原样单行，不抛错（R08）', () => {
    expect(formatXml('')).toEqual([])
    expect(render(formatXml('   '))).toEqual([])
    expect(render(formatXml('not xml at all'))).toEqual(['not xml at all'])
  })

  it('残缺 XML（闭合缺失/多余闭合）不抛错且缩进不为负', () => {
    expect(() => formatXml('<a><b></a>')).not.toThrow()
    const lines = formatXml('</a></b><c>')
    expect(lines.every((l) => l.indent >= 0)).toBe(true)
  })

  it('token 着色分类：标签名/属性名/属性值/文本可区分（供无 v-html 安全渲染）', () => {
    const [line] = formatXml('<vlan id="10">x</vlan>')
    const kinds = new Set(line.tokens.map((t) => t.kind))
    expect(kinds.has('tag')).toBe(true)
    expect(kinds.has('attr-name')).toBe(true)
    expect(kinds.has('attr-value')).toBe(true)
    expect(kinds.has('text')).toBe(true)
    // token 文本拼接必须无损还原该行（不吞字符）
    expect(line.tokens.map((t) => t.text).join('')).toBe('<vlan id="10">x</vlan>')
  })

  it('大报文性能可接受：2000 元素 < 200ms（弹窗不卡）', () => {
    const raw = '<root>' + '<i><n>1</n></i>'.repeat(2000) + '</root>'
    const t0 = Date.now()
    const lines = formatXml(raw)
    expect(Date.now() - t0).toBeLessThan(200)
    expect(lines.length).toBeGreaterThan(2000)
  })
})
