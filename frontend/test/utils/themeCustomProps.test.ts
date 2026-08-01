import { describe, it, expect } from 'vitest'
import { resolve } from 'node:path'
import * as sass from 'sass'

// 回归防线（T07）：SCSS 自定义属性（--x）的值是**纯 CSS**，Sass 不解析其中的
// `$var`——写 `--el-dialog-bg-color: $bg-card` 会原样输出字符串 `$bg-card`，
// 该值在 computed-value 阶段非法，令 `background: var(--el-dialog-bg-color)`
// 回落到初始值 transparent（弹窗/表格/卡片"透明融进背景"的根因）。
// 必须写 `#{$bg-card}` 插值。本测试编译真实 theme.scss 并断言无漏网。
const stylesDir = resolve(process.cwd(), 'src/styles')

const compiled = sass.compile(resolve(stylesDir, 'theme.scss'), { loadPaths: [stylesDir] }).css

/** 提取所有自定义属性声明：`--name: value;` */
function customPropDecls(css: string): { name: string; value: string }[] {
  const out: { name: string; value: string }[] = []
  const re = /(--[A-Za-z0-9-]+)\s*:\s*([^;{}]+);/g
  let m: RegExpExecArray | null
  while ((m = re.exec(css))) out.push({ name: m[1], value: m[2].trim() })
  return out
}

describe('theme.scss · 自定义属性不得残留未插值 SCSS 变量（T07 回归）', () => {
  it('编译产物中无 `--x: $var` 形态（否则该属性在浏览器里非法→回落透明）', () => {
    const offenders = customPropDecls(compiled).filter((d) => d.value.includes('$'))
    expect(
      offenders.map((d) => `${d.name}: ${d.value}`),
      '自定义属性值含未插值 SCSS 变量，请改用 #{$var}',
    ).toEqual([])
  })

  it('弹窗/表格/卡片/抽屉的关键背景变量编译为真实颜色值', () => {
    const decls = new Map(customPropDecls(compiled).map((d) => [d.name, d.value]))
    for (const name of [
      '--el-dialog-bg-color',
      '--el-table-bg-color',
      '--el-card-bg-color',
      '--el-drawer-bg-color',
      '--el-select-dropdown-bg-color',
    ]) {
      const v = decls.get(name)
      expect(v, `${name} 未声明`).toBeTruthy()
      // 真实颜色：# 十六进制 / rgb() / var() 引用，绝不是裸 $ 变量名
      expect(v, `${name} = ${v} 不是合法颜色值`).toMatch(/^(#[0-9A-Fa-f]{3,8}|rgba?\(|var\()/)
    }
  })
})
