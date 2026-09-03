import { describe, it, expect } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

// 左树搜索框事故守护（2026-09-03，[[eviewui-switch-implementation]] 「前缀通配
// 误伤」第二起）：侧栏曾用 `[class*='ev_textField']` 把 eview TextField 全部层
// 拉到 100%，输入后出现的清除钮（挂 eview suffix 槽）一并被拉满——搜索框变长
// 溢出侧栏、输入区被挤没。规则：
// ① theme.scss 禁再出现 ev_textField 前缀通配选择器（拉宽一律点名类/结构锚）；
// ② 桥自绘装饰（affix/prefix/clear）必须有样式声明——曾经三个类零样式裸奔。
const THEME = resolve(process.cwd(), 'src/styles/theme.scss')

describe('eview 输入框装饰样式守护', () => {
  // 剥注释后再匹配（注释里允许引述事故选择器）。
  const css = readFileSync(THEME, 'utf8').replace(/\/\*[\s\S]*?\*\//g, '').replace(/^\s*\/\/.*$/gm, '')

  it('禁 ev_textField 前缀通配选择器（拉宽点名类/结构锚，勿误伤装饰元素）', () => {
    expect(css).not.toMatch(/\[class\*=['"]ev_textField['"]\]/)
  })

  it('桥自绘装饰三件套均有样式声明', () => {
    for (const cls of ['.ub-input-affix', '.ub-input-prefix', '.ub-input-clear']) {
      expect(css, `${cls} 缺样式`).toContain(cls)
    }
  })

  it('affix 容器在侧栏满宽、装饰绝对定位覆于输入之上', () => {
    expect(css).toMatch(/\.sidebar \.ub-input-affix[^{]*\{[^}]*width:\s*100%/)
    expect(css).toMatch(/\.ub-input-prefix[^{]*\{[^}]*position:\s*absolute/)
    expect(css).toMatch(/\.ub-input-clear[^{]*\{[^}]*position:\s*absolute/)
  })
})
