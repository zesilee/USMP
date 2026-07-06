import { describe, it, expect } from 'vitest'
import * as sass from 'sass'
import { fileURLToPath } from 'node:url'

// 设计令牌契约测试：编译 theme.scss，断言其 :root 导出的 CSS 自定义属性。
// 目的——锁定「浅色 iMaster NCE」令牌契约，防止误改回深色科技感主题，
// 并保证新外壳组件依赖的原型变量名（--paper/--surface/--ink/--st-* 等）始终存在。
// 无数据库、无运行时依赖：纯编译期快照式断言。

const themePath = fileURLToPath(new URL('../../src/styles/theme.scss', import.meta.url))

/** 编译 theme.scss，抽取所有 `--name: value;` 自定义属性为 map（后写覆盖先写，贴合 CSS 级联）。 */
function compileTokens(): Record<string, string> {
  const { css } = sass.compile(themePath)
  const map: Record<string, string> = {}
  // 仅匹配「真正的自定义属性声明」：值不含 { } ;，且以 ; 结束。
  // 关键——排除选择器里的伪类片段（如 `.el-button--primary:active {…}`），
  // 否则 --primary 会被选择器覆盖成 "active {…}"。
  const re = /(--[\w-]+)\s*:\s*([^{};]+);/g
  let m: RegExpExecArray | null
  while ((m = re.exec(css)) !== null) {
    map[m[1]] = m[2].trim()
  }
  return map
}

describe('设计令牌契约 · 浅色 iMaster', () => {
  const tokens = compileTokens()

  it('导出原型中性/表面令牌（浅色）', () => {
    expect(tokens['--paper']).toBe('#EAEEF3')
    expect(tokens['--surface']).toBe('#FFFFFF')
    expect(tokens['--sunken']).toBe('#F4F6F9')
    expect(tokens['--ink']).toBe('#16222E')
    expect(tokens['--ink-2']).toBe('#57697A')
    expect(tokens['--ink-3']).toBe('#93A2B1')
    expect(tokens['--line']).toBe('#DBE2EA')
    expect(tokens['--line-strong']).toBe('#C4CDD7')
  })

  it('导出深钢蓝主交互色（非霓虹蓝 #165DFF）', () => {
    expect(tokens['--primary']).toBe('#0C5EA6')
    expect(tokens['--primary-ink']).toBe('#094B84')
    expect(tokens['--primary-weak']).toBe('#E7F0F8')
    expect(tokens['--primary']).not.toBe('#165DFF')
  })

  it('导出收敛四态语义色 + 背景', () => {
    expect(tokens['--st-conv']).toBe('#10814A')
    expect(tokens['--st-conv-bg']).toBe('#E3F3EB')
    expect(tokens['--st-recon']).toBe('#0C5EA6')
    expect(tokens['--st-drift']).toBe('#B26A00')
    expect(tokens['--st-off']).toBe('#C7000B')
    expect(tokens['--brand']).toBe('#C7000B')
  })

  it('字体为 IBM Plex，非 Inter/Roboto/Plus Jakarta（R11）', () => {
    expect(tokens['--f-sans']).toContain('IBM Plex Sans')
    expect(tokens['--f-mono']).toContain('IBM Plex Mono')
    expect(tokens['--f-sans']).not.toContain('Inter')
    expect(tokens['--f-sans']).not.toContain('Roboto')
    expect(tokens['--f-sans']).not.toContain('Jakarta')
  })

  it('外壳几何令牌存在', () => {
    expect(tokens['--sidebar-w']).toBe('224px')
    expect(tokens['--topbar-h']).toBe('56px')
    expect(tokens['--r-card']).toBe('10px')
    expect(tokens['--r-ctl']).toBe('7px')
  })

  it('Element Plus 主色映射到浅色主交互色（不再是深色主题）', () => {
    expect(tokens['--el-color-primary']).toBe('#0C5EA6')
    // 页面底色为浅色，杜绝残留的深色 #0F172A
    expect(tokens['--el-bg-color-page']).not.toBe('#0F172A')
  })

  it('无深色主题残留：中性表面全部为浅色（亮度高位）', () => {
    // --paper 应为浅色（首字节 E/F），若回退深色 (#0F/#1E) 立即失败
    expect(tokens['--paper']!.toUpperCase()).toMatch(/^#[E-F]/)
    expect(tokens['--surface']!.toUpperCase()).toBe('#FFFFFF')
  })
})
