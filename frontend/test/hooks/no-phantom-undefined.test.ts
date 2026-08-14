import { describe, it, expect } from 'vitest'
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

// design D6 静态护栏（tasks 6.4，ESLint 规则的守护测试形态）：拦截「伪删键」
// 写法 `{ ...prev, [k]: undefined }` ——键仍在（in/Object.keys 可见），FE-27 的
// 「键不存在=节点不存在」语义被静默破坏。删键 SHALL 用解构：
//   const { [k]: _drop, ...rest } = prev
// 行为层防线见 useConfigForm.presence.test（双防线）。
const SRC = resolve(process.cwd(), 'src')

// spread 后跟「计算键或字面键: undefined」——D6 明确的反模式形态。
const PHANTOM = /\.\.\.[A-Za-z_$][\w$]*\s*,[^}]*(\[[^\]]+\]|[\w$'"-]+)\s*:\s*undefined/s

function walk(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) return p === join(SRC, 'assets') ? [] : walk(p)
    return /\.(ts|tsx)$/.test(name) && !name.endsWith('.d.ts') ? [p] : []
  })
}

// 剥注释（行注释 + 块注释）后再匹配——JSDoc 里引用反例示范不算违规。
function stripComments(code: string): string {
  return code.replace(/\/\*[\s\S]*?\*\//g, '').replace(/\/\/[^\n]*/g, '')
}

describe('D6 伪删键守护（键存在性语义）', () => {
  it('src 无 `{...prev, [k]: undefined}` 反模式', () => {
    const violations = walk(SRC).filter((f) => PHANTOM.test(stripComments(readFileSync(f, 'utf8'))))
    expect(violations.map((f) => relative(SRC, f))).toEqual([])
  })

  it('自测：反模式样例可被识别（防守护空转）', () => {
    expect(PHANTOM.test('setForm(prev => ({ ...prev, [k]: undefined }))')).toBe(true)
    expect(PHANTOM.test("setForm(prev => ({ ...prev, name: undefined }))")).toBe(true)
    expect(PHANTOM.test('const { [k]: _drop, ...rest } = prev')).toBe(false)
  })
})
