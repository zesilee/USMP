// xpathEval —— YANG `when`/`must` 表达式的轻量求值器。
//
// 只覆盖设备 YANG 约束真实使用的 XPath 子集（语料取自 huawei-ifm.yang）：
//   相对路径 `../leaf`、字符串/数字字面量、= != > < >= <=、and / or / not()、mod、括号。
// 纯函数、无副作用、无共享可变态（R09）；不引入 eval / safe-eval 等依赖（R10）。
// 语法非法时抛 XPathParseError，由上层降级（R08），绝不 new Function/eval。

export class XPathParseError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'XPathParseError'
  }
}

// ---- AST ----
type Node =
  | { kind: 'lit'; value: string | number }
  | { kind: 'path'; segment: string }
  | { kind: 'binary'; op: string; left: Node; right: Node }
  | { kind: 'logical'; op: 'and' | 'or'; left: Node; right: Node }
  | { kind: 'not'; expr: Node }

// ---- 词法 ----
type Token = { t: 'str' | 'num' | 'path' | 'op' | 'kw' | 'lparen' | 'rparen'; v: string }

const KEYWORDS = new Set(['and', 'or', 'not', 'mod'])
// 路径/名字允许的字符：字母数字与 YANG 标识符常见符号（含 ../ 的 . 与 /）。
const PATH_CHAR = /[A-Za-z0-9_\-.:/]/

function tokenize(src: string): Token[] {
  const toks: Token[] = []
  let i = 0
  while (i < src.length) {
    const c = src[i]
    if (c === ' ' || c === '\t' || c === '\n' || c === '\r') {
      i++
      continue
    }
    if (c === '(') {
      toks.push({ t: 'lparen', v: c })
      i++
      continue
    }
    if (c === ')') {
      toks.push({ t: 'rparen', v: c })
      i++
      continue
    }
    if (c === "'" || c === '"') {
      const end = src.indexOf(c, i + 1)
      if (end === -1) throw new XPathParseError(`未闭合的字符串字面量：${src.slice(i)}`)
      toks.push({ t: 'str', v: src.slice(i + 1, end) })
      i = end + 1
      continue
    }
    if (c === '!' || c === '>' || c === '<' || c === '=') {
      if (src[i + 1] === '=') {
        toks.push({ t: 'op', v: c + '=' })
        i += 2
      } else if (c === '!') {
        throw new XPathParseError(`非法字符 '!'（应为 '!='）`)
      } else {
        toks.push({ t: 'op', v: c })
        i++
      }
      continue
    }
    // 数字：以数字开头（负号不在 YANG when/must 语料中，故不特判）。
    if (c >= '0' && c <= '9') {
      let j = i
      while (j < src.length && /[0-9.]/.test(src[j])) j++
      toks.push({ t: 'num', v: src.slice(i, j) })
      i = j
      continue
    }
    // 路径 / 名字 / 关键字。
    if (PATH_CHAR.test(c)) {
      let j = i
      while (j < src.length && PATH_CHAR.test(src[j])) j++
      const word = src.slice(i, j)
      toks.push(KEYWORDS.has(word) ? { t: 'kw', v: word } : { t: 'path', v: word })
      i = j
      continue
    }
    throw new XPathParseError(`非法字符 '${c}'`)
  }
  return toks
}

// ---- 语法（递归下降，优先级低→高：or > and > 比较 > mod > 一元/primary）----
class Parser {
  private p = 0
  constructor(private toks: Token[]) {}

  parse(): Node {
    if (this.toks.length === 0) throw new XPathParseError('空表达式')
    const node = this.parseOr()
    if (this.p < this.toks.length) throw new XPathParseError(`多余的标记：${this.toks[this.p].v}`)
    return node
  }

  private peek(): Token | undefined {
    return this.toks[this.p]
  }

  private parseOr(): Node {
    let left = this.parseAnd()
    while (this.peek()?.t === 'kw' && this.peek()!.v === 'or') {
      this.p++
      left = { kind: 'logical', op: 'or', left, right: this.parseAnd() }
    }
    return left
  }

  private parseAnd(): Node {
    let left = this.parseComparison()
    while (this.peek()?.t === 'kw' && this.peek()!.v === 'and') {
      this.p++
      left = { kind: 'logical', op: 'and', left, right: this.parseComparison() }
    }
    return left
  }

  private parseComparison(): Node {
    const left = this.parseMod()
    const tok = this.peek()
    if (tok?.t === 'op') {
      this.p++
      return { kind: 'binary', op: tok.v, left, right: this.parseMod() }
    }
    return left
  }

  private parseMod(): Node {
    let left = this.parsePrimary()
    while (this.peek()?.t === 'kw' && this.peek()!.v === 'mod') {
      this.p++
      left = { kind: 'binary', op: 'mod', left, right: this.parsePrimary() }
    }
    return left
  }

  private parsePrimary(): Node {
    const tok = this.peek()
    if (!tok) throw new XPathParseError('表达式意外结束')
    if (tok.t === 'kw' && tok.v === 'not') {
      this.p++
      this.expect('lparen', "'not' 后应为 '('")
      const expr = this.parseOr()
      this.expect('rparen', "'not(...)' 缺少 ')'")
      return { kind: 'not', expr }
    }
    if (tok.t === 'lparen') {
      this.p++
      const expr = this.parseOr()
      this.expect('rparen', "缺少 ')'")
      return expr
    }
    if (tok.t === 'str') {
      this.p++
      return { kind: 'lit', value: tok.v }
    }
    if (tok.t === 'num') {
      this.p++
      return { kind: 'lit', value: Number(tok.v) }
    }
    if (tok.t === 'path') {
      this.p++
      return { kind: 'path', segment: lastSegment(tok.v) }
    }
    throw new XPathParseError(`意外的标记：${tok.v}`)
  }

  private expect(t: Token['t'], msg: string) {
    if (this.peek()?.t !== t) throw new XPathParseError(msg)
    this.p++
  }
}

// `../class` / `../foo/bar` → 末段叶子名（表单数据以 YANG 叶子名为键）。
function lastSegment(path: string): string {
  const stripped = path.replace(/^[./]+/, '')
  const segs = stripped.split('/').filter(Boolean)
  return segs.length ? segs[segs.length - 1] : stripped
}

export function parseXPath(expr: string): Node {
  return new Parser(tokenize(expr)).parse()
}

// ---- 求值 ----
type Ctx = Record<string, unknown>

function isNumericLike(v: unknown): boolean {
  if (typeof v === 'number') return !Number.isNaN(v)
  if (typeof v === 'string') return v.trim() !== '' && !Number.isNaN(Number(v))
  return false
}

function looseEq(a: unknown, b: unknown): boolean {
  if (isNumericLike(a) && isNumericLike(b)) return Number(a) === Number(b)
  return String(a ?? '') === String(b ?? '')
}

function toNum(v: unknown): number {
  return typeof v === 'number' ? v : Number(v)
}

export function toBool(v: unknown): boolean {
  if (typeof v === 'boolean') return v
  if (typeof v === 'number') return v !== 0 && !Number.isNaN(v)
  if (typeof v === 'string') return v.length > 0
  return v != null
}

export function evaluateXPath(node: Node, ctx: Ctx): string | number | boolean {
  switch (node.kind) {
    case 'lit':
      return node.value
    case 'path': {
      const v = ctx[node.segment]
      return (v ?? '') as string | number | boolean
    }
    case 'not':
      return !toBool(evaluateXPath(node.expr, ctx))
    case 'logical': {
      const l = toBool(evaluateXPath(node.left, ctx))
      if (node.op === 'and') return l && toBool(evaluateXPath(node.right, ctx))
      return l || toBool(evaluateXPath(node.right, ctx))
    }
    case 'binary': {
      const l = evaluateXPath(node.left, ctx)
      const r = evaluateXPath(node.right, ctx)
      switch (node.op) {
        case '=':
          return looseEq(l, r)
        case '!=':
          return !looseEq(l, r)
        case '>':
          return toNum(l) > toNum(r)
        case '<':
          return toNum(l) < toNum(r)
        case '>=':
          return toNum(l) >= toNum(r)
        case '<=':
          return toNum(l) <= toNum(r)
        case 'mod':
          return toNum(l) % toNum(r)
        default:
          throw new XPathParseError(`不支持的运算符：${node.op}`)
      }
    }
  }
}

// evalPredicate：parse+eval 一步到位并降级。合法 → { value }；异常 → { error }（R08）。
export function evalPredicate(
  expr: string,
  ctx: Ctx,
): { value: boolean } | { value?: undefined; error: string } {
  try {
    return { value: toBool(evaluateXPath(parseXPath(expr), ctx)) }
  } catch (e) {
    return { error: e instanceof Error ? e.message : String(e) }
  }
}
