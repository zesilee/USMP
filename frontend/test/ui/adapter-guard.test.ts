import { describe, it, expect } from 'vitest'
import { readFileSync, readdirSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

// FA-01/FA-02 守护测试（frontend-ui-adapter spec）：
// ① 业务代码禁直接 import 组件库——src/ 下除 src/ui/ 外任何文件出现
//    `from 'antd'` / `from '@ant-design/...'` 即失败；换库时本清单是唯一豁免区。
// ② 适配层禁整包透传——src/ui/ 内禁 `export * from '<组件库>'`（自有子模块的
//    命名空间再导出不在此列）。
// vitest cwd 恒为 frontend/（happy-dom 下 import.meta.url 非 file 协议，不可用）。
const SRC = resolve(process.cwd(), 'src')
// 匹配模块说明符字符串本身：覆盖 import-from / 动态 import() / require /
// import x = require / 副作用裸导入（'antd/dist/reset.css'）全部形态（评审 #2）。
const LIB_IMPORT = /['"](antd|antd\/[^'"]*|@ant-design\/[^'"]*)['"]/

function walk(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const p = join(dir, name)
    // 仅豁免顶层 src/assets（snd-i18n 数据），深层同名目录不放行（评审 #7）。
    if (statSync(p).isDirectory()) return p === join(SRC, 'assets') ? [] : walk(p)
    return /\.(ts|tsx)$/.test(name) && !name.endsWith('.d.ts') ? [p] : []
  })
}

describe('UI 适配层守护（FA-01/FA-02）', () => {
  const files = walk(SRC)
  // 适配层自身以外的全部源码 = 业务面。
  const business = files.filter((f) => !relative(SRC, f).startsWith('ui/'))
  const adapter = files.filter((f) => relative(SRC, f).startsWith('ui/'))

  it('业务代码零直接组件库 import（FA-01）', () => {
    const violations = business.filter((f) => LIB_IMPORT.test(readFileSync(f, 'utf8')))
    expect(violations.map((f) => relative(SRC, f))).toEqual([])
  })

  it('适配层禁整包透传组件库（FA-02）', () => {
    // 含 `export * as ns from 'antd'` 命名空间形态（评审 #3）。
    const starExport = /export\s+\*\s+(?:as\s+\w+\s+)?from\s+['"](antd|@ant-design\/[^'"]*)['"]/
    const violations = adapter.filter((f) => starExport.test(readFileSync(f, 'utf8')))
    expect(violations.map((f) => relative(SRC, f))).toEqual([])
  })

  it('守护自检：适配层确实存在且引用了组件库（防守护空转）', () => {
    expect(adapter.length).toBeGreaterThan(0)
    expect(adapter.some((f) => LIB_IMPORT.test(readFileSync(f, 'utf8')))).toBe(true)
  })
})
