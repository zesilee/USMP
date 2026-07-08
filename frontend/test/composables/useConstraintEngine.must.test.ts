import { describe, it, expect } from 'vitest'
import { ref } from 'vue'
import { useConstraintEngine } from '../../src/composables/useConstraintEngine'
import type { Field } from '../../src/utils/crdSchemaParser'

// must 语料取自真实 IFM：suppress>reuse（阻尼抑制阈值）、interval mod 10=0（统计周期）。
describe('useConstraintEngine · must 跨字段校验（数据驱动，无硬编码）', () => {
  const mustFields: Field[] = [
    { path: '/d/suppress', type: 'number', label: 'suppress' },
    { path: '/d/reuse', type: 'number', label: 'reuse', must: [{ expr: '(../suppress>../reuse)', message: 'reuse 必须小于 suppress' }] },
    { path: '/d/interval', type: 'number', label: 'interval', must: [{ expr: '(../interval) mod 10 = 0' }] },
  ]

  it('满足约束 → 无违例；违反 → 返回带消息的 violation', () => {
    const form = ref<Record<string, any>>({ suppress: 2000, reuse: 750, interval: 20 })
    const { mustViolations } = useConstraintEngine(mustFields, form)
    expect(mustViolations.value).toEqual([])

    form.value.reuse = 3000 // reuse>suppress → 违反
    expect(mustViolations.value.map((v) => v.path)).toContain('/d/reuse')
    expect(mustViolations.value.find((v) => v.path === '/d/reuse')!.message).toBe('reuse 必须小于 suppress')
  })

  it('无 message 的 must → 生成含字段标签的通用提示', () => {
    const form = ref<Record<string, any>>({ suppress: 2000, reuse: 750, interval: 15 })
    const { mustViolations } = useConstraintEngine(mustFields, form)
    const v = mustViolations.value.find((x) => x.path === '/d/interval')
    expect(v).toBeTruthy()
    expect(v!.message).toContain('interval')
  })

  it('隐藏字段(when=false)的 must 不触发（YANG 语义：节点不存在）', () => {
    const f: Field[] = [
      { path: '/d/mode', type: 'string', label: 'mode' },
      { path: '/d/x', type: 'number', label: 'x', when: "../mode='on'", must: [{ expr: '../x>10' }] },
    ]
    const form = ref<Record<string, any>>({ mode: 'off', x: 5 }) // x 隐藏 → must 跳过
    const { mustViolations } = useConstraintEngine(f, form)
    expect(mustViolations.value).toEqual([])
  })

  it('must 表达式解析失败 → 不阻断 + 记录告警（R08）', () => {
    const f: Field[] = [{ path: '/d/bad', type: 'number', label: 'bad', must: [{ expr: '../a = = 1' }] }]
    const form = ref<Record<string, any>>({})
    const { mustViolations, warnings } = useConstraintEngine(f, form)
    expect(mustViolations.value).toEqual([])
    expect(warnings.value.some((w) => w.includes('/d/bad'))).toBe(true)
  })
})
