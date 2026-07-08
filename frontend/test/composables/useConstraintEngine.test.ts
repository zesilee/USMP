import { describe, it, expect } from 'vitest'
import { ref } from 'vue'
import { useConstraintEngine } from '../../src/composables/useConstraintEngine'
import type { Field } from '../../src/utils/crdSchemaParser'

const fields: Field[] = [
  { path: '/i/interface/name', type: 'string', label: 'name' },
  { path: '/i/interface/class', type: 'enum', label: 'class' },
  { path: '/i/interface/parent-name', type: 'string', label: 'parent', when: "../class='sub-interface'" },
  { path: '/i/interface/bad', type: 'string', label: 'bad', when: '../x = = 1' },
]

describe('useConstraintEngine · when 驱动的响应式显隐（数据驱动，无硬编码）', () => {
  it('无 when 字段恒可见；when 字段随被引用值响应式变化', () => {
    const form = ref<Record<string, any>>({ class: 'main-interface' })
    const { visibleMap, isVisible } = useConstraintEngine(fields, form)

    // 无 when → 恒可见
    expect(visibleMap.value['/i/interface/name']).toBe(true)
    // class=main-interface → parent-name 隐藏
    expect(visibleMap.value['/i/interface/parent-name']).toBe(false)

    // 改被引用叶 → 响应式重算
    form.value.class = 'sub-interface'
    expect(visibleMap.value['/i/interface/parent-name']).toBe(true)
    expect(isVisible(fields[2])).toBe(true)

    form.value.class = 'main-interface'
    expect(visibleMap.value['/i/interface/parent-name']).toBe(false)
  })

  it('when 表达式解析失败 → 降级为可见并记录告警（R08）', () => {
    const form = ref<Record<string, any>>({})
    const { visibleMap, warnings } = useConstraintEngine(fields, form)
    expect(visibleMap.value['/i/interface/bad']).toBe(true)
    expect(warnings.value.some((w) => w.includes('/i/interface/bad'))).toBe(true)
  })

  it('接受 ref 形式的 fields', () => {
    const form = ref<Record<string, any>>({ class: 'sub-interface' })
    const { visibleMap } = useConstraintEngine(ref(fields), form)
    expect(visibleMap.value['/i/interface/parent-name']).toBe(true)
  })
})
