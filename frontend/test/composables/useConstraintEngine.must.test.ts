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

  // 真机回归（T07，CE6866 创建接口死循环）：statistic-mode 挂自引用 must
  // 「../statistic-mode='interface-based' or (../l2-mode-enable='true' and
  // ../statistic-enable='true')」。RFC7950 §7.5.3：must 只约束存在的节点——
  // 叶子未赋值=节点不存在，must 不适用。旧行为把未填也判违例 → 门禁强迫用户
  // 选值 → 设备按接口类型裁剪该叶 → rpc-error unknown-element，两头堵死。
  describe('叶子未赋值 → 其 must 不适用（节点不存在语义）', () => {
    const smFields: Field[] = [
      { path: '/ifm/interfaces/interface/l2-mode-enable', type: 'boolean', label: 'l2-mode-enable' },
      { path: '/ifm/interfaces/interface/statistic-enable', type: 'boolean', label: 'statistic-enable' },
      {
        path: '/ifm/interfaces/interface/statistic-mode',
        type: 'string',
        label: 'statistic-mode',
        must: [{ expr: "../statistic-mode = 'interface-based' or (../l2-mode-enable = 'true' and ../statistic-enable = 'true')" }],
      },
    ]

    // 真机二次回归（T07）：statistic-mode 的真实 FieldDef 是 type='enum'（带
    // options 下拉），首修的叶判定漏了 enum——测试用 string 建模绕开了真实形态，
    // 真机复测仍被门禁拦。此例按后端 buildYangSchemaNested 实际输出逐字建模。
    it('enum 叶（真实 FieldDef 形态）未选 → 无违例', () => {
      const realShape: Field[] = [
        { path: '/ifm/interfaces/interface/l2-mode-enable', type: 'boolean', label: 'l2-mode-enable' },
        { path: '/ifm/interfaces/interface/statistic-enable', type: 'boolean', label: 'statistic-enable' },
        {
          path: '/ifm/interfaces/interface/statistic-mode',
          type: 'enum',
          label: 'statistic-mode',
          options: [
            { label: 'interface-based', value: 'interface-based' },
            { label: 'vlan-group-based', value: 'vlan-group-based' },
          ],
          must: [{ expr: "../statistic-mode = 'interface-based' or (../l2-mode-enable = 'true' and ../statistic-enable = 'true')" }],
        } as Field,
      ]
      const form = ref<Record<string, any>>({})
      const { mustViolations } = useConstraintEngine(realShape, form)
      expect(mustViolations.value).toEqual([])
      form.value['statistic-mode'] = 'vlan-group-based' // 选了违反值 → 正常违例
      expect(mustViolations.value.map((v) => v.path)).toContain('/ifm/interfaces/interface/statistic-mode')
      form.value['statistic-mode'] = '' // 清空回未选 → 违例消失
      expect(mustViolations.value).toEqual([])
    })

    it('未填（undefined/空串）→ 无违例，不再强迫用户选值', () => {
      const form = ref<Record<string, any>>({})
      const { mustViolations } = useConstraintEngine(smFields, form)
      expect(mustViolations.value).toEqual([])
      form.value['statistic-mode'] = ''
      expect(mustViolations.value).toEqual([])
    })

    it('填了值 → must 正常评估：满足无违例、违反有违例', () => {
      const form = ref<Record<string, any>>({ 'statistic-mode': 'interface-based' })
      const { mustViolations } = useConstraintEngine(smFields, form)
      expect(mustViolations.value).toEqual([])
      form.value['statistic-mode'] = 'vlan-based' // l2/statistic-enable 未开 → 违反
      expect(mustViolations.value.map((v) => v.path)).toContain('/ifm/interfaces/interface/statistic-mode')
    })

    it('引用他叶的 must：载体叶未填同样跳过（reuse 未填不评估 suppress>reuse）', () => {
      const form = ref<Record<string, any>>({ suppress: 2000 })
      const { mustViolations } = useConstraintEngine(mustFields, form)
      expect(mustViolations.value.filter((v) => v.path === '/d/reuse')).toEqual([])
    })

    it('布尔 false 与数值 0 是有效值，must 照常评估', () => {
      const f: Field[] = [
        { path: '/d/n', type: 'number', label: 'n', must: [{ expr: '../n > 0' }] },
      ]
      const form = ref<Record<string, any>>({ n: 0 })
      const { mustViolations } = useConstraintEngine(f, form)
      expect(mustViolations.value.map((v) => v.path)).toContain('/d/n')
    })
  })

  it('must 表达式解析失败 → 不阻断 + 记录告警（R08）', () => {
    const f: Field[] = [{ path: '/d/bad', type: 'number', label: 'bad', must: [{ expr: '../a = = 1' }] }]
    const form = ref<Record<string, any>>({})
    const { mustViolations, warnings } = useConstraintEngine(f, form)
    expect(mustViolations.value).toEqual([])
    expect(warnings.value.some((w) => w.includes('/d/bad'))).toBe(true)
  })
})
