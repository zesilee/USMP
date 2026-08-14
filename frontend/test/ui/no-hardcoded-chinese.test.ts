import { describe, it, expect } from 'vitest'
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

// UI-02 判据固化（tasks 4.5）：前端源码（排除 locales/assets/声明文件）SHALL NOT
// 在**字符串字面量**中残留界面中文——注释中文合法（工程叙事语言），词表中文归
// locales。旧时代为临时 grep 口径，现固化为守护测试随 pre-commit/CI 常跑。
const SRC = resolve(process.cwd(), 'src')
const CHINESE = /[一-鿿]/

// 文件级豁免：存量**开发者诊断字符串**（console 日志 / 解析器 throw 消息），
// 不进 DOM、非界面 chrome 文案（UI-02 排除口径），且 xpathEval 属 D4 逐字节
// 沿用件禁顺手重构。新文件/新界面中文不在豁免内，仍会命中。
const DIAGNOSTIC_ALLOWLIST = new Set(['utils/xpathEval.ts', 'stores/menu.ts'])

function walk(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) {
      return p === join(SRC, 'assets') || p === join(SRC, 'locales') ? [] : walk(p)
    }
    return /\.(ts|tsx)$/.test(name) && !name.endsWith('.d.ts') ? [p] : []
  })
}

// 剥注释后仅在字符串字面量（'…' "…" `…`）里找中文。逐字符小扫描器：
// 状态机区分 注释/字符串/代码，避免正则对嵌套引号的误判。
function chineseInStrings(code: string): string[] {
  const hits: string[] = []
  let i = 0
  let line = 1
  const n = code.length
  while (i < n) {
    const c = code[i]
    const next = code[i + 1]
    if (c === '\n') { line++; i++; continue }
    if (c === '/' && next === '/') { while (i < n && code[i] !== '\n') i++; continue }
    if (c === '/' && next === '*') {
      i += 2
      while (i < n && !(code[i] === '*' && code[i + 1] === '/')) { if (code[i] === '\n') line++; i++ }
      i += 2
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
      if (CHINESE.test(buf)) hits.push(`L${line}: ${buf.slice(0, 40)}`)
      continue
    }
    i++
  }
  return hits
}

describe('UI-02 零硬编码界面中文（守护）', () => {
  it('src 字符串字面量无中文（locales/assets 除外）', () => {
    const violations: string[] = []
    for (const f of walk(SRC)) {
      const rel = relative(SRC, f)
      if (DIAGNOSTIC_ALLOWLIST.has(rel)) continue
      for (const hit of chineseInStrings(readFileSync(f, 'utf8'))) {
        violations.push(`${rel} ${hit}`)
      }
    }
    expect(violations).toEqual([])
  })
})
