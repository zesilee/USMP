import { describe, it, expect } from 'vitest'
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

// UI-02 判据固化（tasks 4.5）：前端源码（排除 locales/assets/声明文件）SHALL NOT
// 残留界面中文——**字符串字面量与 JSX 文本**都拦（评审 #1：JSX text child 是组件
// 波次最高频的泄漏通道），注释中文合法（工程叙事语言），词表中文归 locales。
// 旧时代为临时 grep 口径，现固化为守护测试随 pre-commit/CI 常跑。
const SRC = resolve(process.cwd(), 'src')
const CHINESE = /[一-鿿]/

// 文件级豁免：xpathEval 的解析器 throw 诊断消息（不进 DOM），且属 D4 逐字节
// 沿用件禁顺手重构。新文件/新界面中文不在豁免内，仍会命中。
const DIAGNOSTIC_ALLOWLIST = new Set(['utils/xpathEval.ts'])

function walk(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) {
      return p === join(SRC, 'assets') || p === join(SRC, 'locales') ? [] : walk(p)
    }
    return /\.(ts|tsx)$/.test(name) && !name.endsWith('.d.ts') ? [p] : []
  })
}

export interface ScanResult {
  /** 字符串字面量内的中文（含模板字面量）。 */
  inStrings: string[]
  /** 剥注释与字符串后残余代码里的中文——即 JSX 文本节点（或中文标识符）。 */
  inCode: string[]
}

// 逐字符小扫描器：状态机区分 注释/字符串/正则/代码。
// - 正则字面量按前导上下文识别（`=(,:;!&|?{}[` 与行首/return 后的 `/` 视为正则
//   起点，跳到未转义收尾 `/`）——否则含奇数引号的正则会让扫描器错位失明（评审 #2）。
// - 已知限制：模板字面量嵌套 ${`…`} 会提前终止外层（当前 src 无此写法）。
export function scanChinese(code: string): ScanResult {
  const inStrings: string[] = []
  let stripped = ''
  let i = 0
  let line = 1
  let lastSignificant = ''
  const n = code.length
  const push = (buf: string) => {
    if (CHINESE.test(buf)) inStrings.push(`L${line}: ${buf.slice(0, 40)}`)
  }
  while (i < n) {
    const c = code[i]
    const next = code[i + 1]
    if (c === '\n') { line++; stripped += '\n'; i++; continue }
    if (c === '/' && next === '/') { while (i < n && code[i] !== '\n') i++; continue }
    if (c === '/' && next === '*') {
      i += 2
      while (i < n && !(code[i] === '*' && code[i + 1] === '/')) { if (code[i] === '\n') line++; i++ }
      i += 2
      continue
    }
    if (c === '/' && /[=(,:;!&|?{[\n]|^$|return$/.test(lastSignificant)) {
      // 正则字面量：跳过至未转义 '/'（字符类内的 '/' 无需转义，粗跳可接受——
      // 字符类内含 '/' 的正则极罕见，出现时扫描恢复点仍在同行）。
      i++
      while (i < n && code[i] !== '/') {
        if (code[i] === '\\') i++
        if (code[i] === '\n') { line++; break }
        i++
      }
      i++
      lastSignificant = ')'
      continue
    }
    if (c === '"' || c === "'" || c === '`') {
      const quote = c
      i++
      let buf = ''
      while (i < n && code[i] !== quote) {
        if (code[i] === '\\') { buf += code[i] + (code[i + 1] ?? ''); i += 2; continue }
        if (code[i] === '\n') line++
        buf += code[i]
        i++
      }
      i++
      push(buf)
      lastSignificant = ')'
      continue
    }
    stripped += c
    if (!/\s/.test(c)) {
      // 追踪「上一个有效 token 尾」：单字符即够（return 特判取词尾）。
      lastSignificant = /[a-zA-Z]/.test(c)
        ? (lastSignificant + c).slice(-6).match(/[a-zA-Z]+$/)?.[0] ?? c
        : c
    }
    i++
  }
  const inCode = stripped
    .split('\n')
    .map((l, idx) => (CHINESE.test(l) ? `L${idx + 1}(jsx/code): ${l.trim().slice(0, 40)}` : null))
    .filter((x): x is string => x !== null)
  return { inStrings, inCode }
}

describe('UI-02 零硬编码界面中文（守护）', () => {
  it('src 字符串字面量与 JSX 文本无中文（locales/assets 除外）', () => {
    const violations: string[] = []
    for (const f of walk(SRC)) {
      const rel = relative(SRC, f)
      if (DIAGNOSTIC_ALLOWLIST.has(rel)) continue
      const r = scanChinese(readFileSync(f, 'utf8'))
      for (const hit of [...r.inStrings, ...r.inCode]) violations.push(`${rel} ${hit}`)
    }
    expect(violations).toEqual([])
  })

  // 扫描器自测（评审 #2 防线）：奇数引号正则不致错位失明；JSX 文本中文可拦。
  it('自测：正则字面量含引号不错位，后续中文字符串仍被捕获', () => {
    const r = scanChinese(`const a = x.split(/'/)\nconst b = '中文文案'\n`)
    expect(r.inStrings).toHaveLength(1)
    expect(r.inStrings[0]).toContain('中文文案')
  })

  it('自测：JSX 文本节点中文被拦（组件波次主通道）', () => {
    const r = scanChinese(`export function A() {\n  return <div>中文标题</div>\n}\n`)
    expect(r.inCode).toHaveLength(1)
    expect(r.inCode[0]).toContain('中文标题')
  })

  it('自测：注释中文合法、词表 t() 调用合法', () => {
    const r = scanChinese(`// 这是中文注释\nconst x = t('console.basicTab')\n`)
    expect(r.inStrings).toHaveLength(0)
    expect(r.inCode).toHaveLength(0)
  })
})
