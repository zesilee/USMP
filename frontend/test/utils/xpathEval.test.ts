import { describe, it, expect } from 'vitest'
import { parseXPath, evaluateXPath, evalPredicate, XPathParseError } from '../../src/utils/xpathEval'

// 求值器只需覆盖 YANG when/must 的 XPath 子集，语料取自真实 huawei-ifm.yang。
describe('xpathEval · YANG XPath 子集求值器（数据驱动，无硬编码）', () => {
  describe('相对路径 + 相等（when 显隐主场景）', () => {
    it("../class='sub-interface' 命中兄弟叶值", () => {
      const ast = parseXPath("../class='sub-interface'")
      expect(evaluateXPath(ast, { class: 'sub-interface' })).toBe(true)
      expect(evaluateXPath(ast, { class: 'main-interface' })).toBe(false)
    })

    it('被引用叶缺失 → 视为空串，相等为假', () => {
      const ast = parseXPath("../class='sub-interface'")
      expect(evaluateXPath(ast, {})).toBe(false)
    })

    it('!= 不等比较', () => {
      const ast = parseXPath("../class!='sub-interface'")
      expect(evaluateXPath(ast, { class: 'main-interface' })).toBe(true)
      expect(evaluateXPath(ast, { class: 'sub-interface' })).toBe(false)
    })
  })

  describe('布尔与逻辑组合', () => {
    it("复合 and：../type='Eth-Trunk' and ../class='main-interface'", () => {
      const ast = parseXPath("../type='Eth-Trunk' and ../class='main-interface'")
      expect(evaluateXPath(ast, { type: 'Eth-Trunk', class: 'main-interface' })).toBe(true)
      expect(evaluateXPath(ast, { type: 'Eth-Trunk', class: 'sub-interface' })).toBe(false)
      expect(evaluateXPath(ast, { type: 'GigabitEthernet', class: 'main-interface' })).toBe(false)
    })

    it("or：../a='x' or ../b='y'", () => {
      const ast = parseXPath("../a='x' or ../b='y'")
      expect(evaluateXPath(ast, { a: 'x', b: '' })).toBe(true)
      expect(evaluateXPath(ast, { a: '', b: 'y' })).toBe(true)
      expect(evaluateXPath(ast, { a: '', b: '' })).toBe(false)
    })

    it("布尔叶按字符串比较：../l2-mode-enable = 'true'", () => {
      const ast = parseXPath("../l2-mode-enable = 'true'")
      expect(evaluateXPath(ast, { 'l2-mode-enable': true })).toBe(true)
      expect(evaluateXPath(ast, { 'l2-mode-enable': false })).toBe(false)
    })

    it('not(...) 取反', () => {
      const ast = parseXPath("not(../class='sub-interface')")
      expect(evaluateXPath(ast, { class: 'main-interface' })).toBe(true)
      expect(evaluateXPath(ast, { class: 'sub-interface' })).toBe(false)
    })
  })

  describe('数值关系与 mod（must 校验场景）', () => {
    it('(../down-delay-time) mod 100 = 0', () => {
      const ast = parseXPath('(../down-delay-time) mod 100 = 0')
      expect(evaluateXPath(ast, { 'down-delay-time': 200 })).toBe(true)
      expect(evaluateXPath(ast, { 'down-delay-time': 150 })).toBe(false)
    })

    it('(../suppress>../reuse) 兄弟叶数值比较', () => {
      const ast = parseXPath('(../suppress>../reuse)')
      expect(evaluateXPath(ast, { suppress: 2000, reuse: 750 })).toBe(true)
      expect(evaluateXPath(ast, { suppress: 500, reuse: 750 })).toBe(false)
    })

    it("negated 复合：not(../a and ../b and ../a<../b)", () => {
      const ast = parseXPath('not(../a and ../b and ../a<../b)')
      // a<b 且都存在 → 内部为真 → not 为假
      expect(evaluateXPath(ast, { a: 10, b: 20 })).toBe(false)
      expect(evaluateXPath(ast, { a: 30, b: 20 })).toBe(true)
    })
  })

  describe('evalPredicate 便捷降级（R08）', () => {
    it('合法表达式返回 { value }', () => {
      expect(evalPredicate("../class='sub-interface'", { class: 'sub-interface' })).toEqual({ value: true })
    })

    it('语法错误返回 { error } 且不抛', () => {
      const r = evalPredicate("../class = = 'x'", { class: 'x' })
      expect(r.error).toBeTruthy()
      expect(r.value).toBeUndefined()
    })
  })

  describe('parse 错误', () => {
    it('未闭合括号抛 XPathParseError', () => {
      expect(() => parseXPath('(../a > 1')).toThrow(XPathParseError)
    })
    it('空表达式抛 XPathParseError', () => {
      expect(() => parseXPath('   ')).toThrow(XPathParseError)
    })
  })
})
