import { describe, it, expect } from 'vitest'
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

// RT-02 守护（frontend-runtime spec，change frontend-eviewui-inula-switch）：
// openInula 不提供 React 18+ API——业务代码出现其调用/导入即失败。
// 只匹配调用与 import 形态，注释中的提及不拦（i18n 薄层注释有说明文字）。
const SRC = resolve(process.cwd(), 'src')
const BANNED = ['useSyncExternalStore', 'useTransition', 'useDeferredValue', 'useId']
const CALL = new RegExp(`\\b(${BANNED.join('|')})\\s*\\(`)
const IMPORT = new RegExp(`import[^\\n]*\\{[^}]*\\b(${BANNED.join('|')})\\b[^}]*\\}[^\\n]*from\\s*['"]react`)

function walk(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const p = join(dir, name)
    // src/runtime = 18 级 API 垫片本体（波 C 备用），是这些 API 的唯一合法实现处。
    if (statSync(p).isDirectory()) return p === join(SRC, 'assets') || p === join(SRC, 'runtime') ? [] : walk(p)
    return /\.(ts|tsx)$/.test(name) && !name.endsWith('.d.ts') ? [p] : []
  })
}

describe('RT-02 运行时禁用 API 守护', () => {
  it('src 零 React 18+ API 调用/导入（openInula 不提供）', () => {
    const violations: string[] = []
    for (const f of walk(SRC)) {
      const body = readFileSync(f, 'utf8')
      for (const [i, line] of body.split('\n').entries()) {
        const code = line.replace(/\/\/.*$/, '').replace(/\/\*.*?\*\//g, '')
        if (CALL.test(code) || IMPORT.test(code)) violations.push(`${relative(SRC, f)}:${i + 1}`)
      }
    }
    expect(violations).toEqual([])
  })
})
