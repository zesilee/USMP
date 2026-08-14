import { describe, it, expect } from 'vitest'
import { computeVisibleMap, computeMustViolations, computeWarnings } from '../../src/form/constraintEngine'
import type { Field } from '../../src/utils/crdSchemaParser'

const fields: Field[] = [
  { path: '/i/interface/name', type: 'string', label: 'name' },
  { path: '/i/interface/class', type: 'enum', label: 'class' },
  { path: '/i/interface/parent-name', type: 'string', label: 'parent', when: "../class='sub-interface'" },
  { path: '/i/interface/bad', type: 'string', label: 'bad', when: '../x = = 1' },
]

describe('useConstraintEngine · when 驱动的响应式显隐（数据驱动，无硬编码）', () => {
  it('无 when 字段恒可见；when 字段随被引用值响应式变化', () => {
    const form: Record<string, any> = { class: 'main-interface' }
    const visibleMap = () => computeVisibleMap(fields, form)
    const isVisible = (f: any) => visibleMap()[f.path] ?? true

    // 无 when → 恒可见
    expect(visibleMap()['/i/interface/name']).toBe(true)
    // class=main-interface → parent-name 隐藏
    expect(visibleMap()['/i/interface/parent-name']).toBe(false)

    // 改被引用叶 → 响应式重算
    form.class = 'sub-interface'
    expect(visibleMap()['/i/interface/parent-name']).toBe(true)
    expect(isVisible(fields[2])).toBe(true)

    form.class = 'main-interface'
    expect(visibleMap()['/i/interface/parent-name']).toBe(false)
  })

  it('when 表达式解析失败 → 降级为可见并记录告警（R08）', () => {
    const form: Record<string, any> = {}
    const visibleMap = () => computeVisibleMap(fields, form)
    const warnings = () => computeWarnings(fields, form)
    expect(visibleMap()['/i/interface/bad']).toBe(true)
    expect(warnings().some((w) => w.includes('/i/interface/bad'))).toBe(true)
  })

  it('接受 ref 形式的 fields', () => {
    const form: Record<string, any> = { class: 'sub-interface' }
    const visibleMap = () => computeVisibleMap(fields, form)
    expect(visibleMap()['/i/interface/parent-name']).toBe(true)
  })
})
