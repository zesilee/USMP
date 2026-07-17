import { reactive, ref, computed, type Ref, type ComputedRef } from 'vue'
import type { FormRules } from 'element-plus'
import { useConstraintEngine } from './useConstraintEngine'
import { computeDiff, missingRequired } from '../utils/configDiff'
import type { Field } from '../utils/crdSchemaParser'

// 模型驱动表单编排（自旧配置页收敛的通用逻辑，FE-07/08/09 语义不变）：
// 约束引擎（when 显隐/must 校验）、pattern/range/required 规则、choice 展开、
// 差异比对与可提交门禁、仅可见字段入 payload。供通用控制台的列表/表单 Tab 复用。
export function useConfigForm(fields: Ref<Field[]> | ComputedRef<Field[]>, keyField?: Ref<string> | ComputedRef<string>) {
  const formData = reactive<Record<string, any>>({})
  const original = ref<Record<string, any>>({}) // 已回填的实际态基线（新增时为空）

  function keyOf(f: Field): string {
    return f.path.split('/').filter(Boolean).pop() || f.path
  }

  // ===== choice 展开（成员扁平同级）=====
  function choiceMemberFields(field: Field): Field[] {
    return (field.cases || []).flatMap((c) => c.fields || [])
  }
  function choiceScope(field: Field): Record<string, any> {
    const o: Record<string, any> = {}
    for (const k of choiceMemberFields(field).map(keyOf)) if (k in formData) o[k] = formData[k]
    return o
  }
  function onChoiceUpdate(field: Field, next: Record<string, any>) {
    for (const k of choiceMemberFields(field).map(keyOf)) {
      if (next[k] === undefined) delete formData[k]
      else formData[k] = next[k]
    }
  }

  // 编译 YANG pattern；非法正则降级为不校验 + 告警（R08）。
  function compilePattern(pattern?: string): RegExp | null {
    if (!pattern) return null
    try {
      return new RegExp(`^(?:${pattern})$`)
    } catch {
      console.warn('[useConfigForm] 非法 YANG pattern，已跳过校验：', pattern)
      return null
    }
  }

  // 约束引擎（FE-07）：when=false 的字段不渲染、不校验、不入 payload。
  const engine = useConstraintEngine(fields, formData)
  const visibleFields = computed(() => fields.value.filter((f) => engine.isVisible(f)))

  // must 违例（presence 语义修正，FE-12）：presence 容器未开启（键不存在=节点不存在）
  // 时其 must 不适用（YANG must 仅约束存在的节点），过滤掉这类违例。
  const mustViolations = computed(() =>
    engine.mustViolations.value.filter((v) => {
      const f = fields.value.find((x) => x.path === v.path)
      // readonly state 叶的 must 不入门禁（FE-14）：设备值用户不可改，违例会永久卡死提交。
      if (f?.readonly) return false
      return !(f?.type === 'group' && f.presence && formData[keyOf(f)] === undefined)
    }),
  )
  const flatFields = computed(() =>
    visibleFields.value.flatMap((f) => (f.type === 'choice' ? choiceMemberFields(f) : [f])),
  )
  // 可编辑字段面（FE-14）：readonly（config false state）叶可见可回显，但不参与
  // 校验/diff/payload——state 数据永不进设备写路径。
  const editableFlat = computed(() => flatFields.value.filter((f) => !f.readonly))

  const patternViolations = computed(() =>
    editableFlat.value.filter((f) => {
      const re = compilePattern(f.pattern)
      if (!re) return false
      const v = formData[keyOf(f)]
      if (v == null || v === '') return false
      return !re.test(String(v))
    }),
  )

  const diff = computed(() => computeDiff(formData, original.value, editableFlat.value))
  const submittable = computed(
    () =>
      diff.value.length > 0 &&
      missingRequired(editableFlat.value, formData, keyField?.value ?? '').length === 0 &&
      mustViolations.value.length === 0 &&
      patternViolations.value.length === 0,
  )

  // 权威门禁（§9）：缺必填/must 违例/pattern 违例一律拦截（el-form validate 只管行内展示）。
  const blocked = computed(
    () =>
      missingRequired(editableFlat.value, formData, keyField?.value ?? '').length > 0 ||
      mustViolations.value.length > 0 ||
      patternViolations.value.length > 0,
  )

  // 由 schema 生成 el-form 校验规则：required/range/must/pattern（行内提示，§9）。
  const rules = computed<FormRules>(() => {
    const r: FormRules = {}
    for (const f of visibleFields.value) {
      if (f.readonly) continue // state 叶只读展示，无校验规则（FE-14）
      const key = keyOf(f)
      const list: any[] = []
      // dynamicDefault 豁免必填（FE-15）：空值=系统自动分配；keyField 恒必填。
      if ((f.required && !f.dynamicDefault) || (keyField && key === keyField.value)) {
        list.push({ required: true, message: `${f.label} 必填`, trigger: ['change', 'blur'] })
      }
      if (f.type === 'number' && (f.minimum != null || f.maximum != null)) {
        list.push({ type: 'number', min: f.minimum, max: f.maximum, message: `${f.label} 超出范围`, trigger: ['change', 'blur'] })
      }
      if (f.must?.length) {
        list.push({
          validator: (_rule: unknown, _value: unknown, cb: (e?: Error) => void) => {
            const v = mustViolations.value.find((x) => x.path === f.path)
            cb(v ? new Error(v.message) : undefined)
          },
          trigger: ['change', 'blur'],
        })
      }
      const re = compilePattern(f.pattern)
      if (re) {
        list.push({ pattern: re, message: `${f.label} 格式不符合约束`, trigger: ['change', 'blur'] })
      }
      if (list.length) r[key] = list
    }
    return r
  })

  // 下发 payload：仅当前可见字段的键（when 隐藏 = 节点不存在）；undefined 键剔除
  //（presence 关闭 = 节点不存在，FE-12）。
  function visiblePayload(): Record<string, any> {
    const keys = new Set(editableFlat.value.map(keyOf))
    // dynamicDefault 叶空值不入 payload（FE-15）：空=「设备自行决定」，下发空串会覆写设备缺省。
    const dynKeys = new Set(editableFlat.value.filter((f) => f.dynamicDefault).map(keyOf))
    const out: Record<string, any> = {}
    for (const k of Object.keys(formData)) {
      if (!keys.has(k) || formData[k] === undefined) continue
      if (dynKeys.has(k) && (formData[k] === '' || formData[k] === null)) continue
      out[k] = formData[k]
    }
    return out
  }

  function resetForm(seed: Record<string, any> = {}) {
    Object.keys(formData).forEach((k) => delete formData[k])
    Object.assign(formData, seed)
    original.value = { ...seed }
  }

  return {
    formData,
    original,
    engine,
    mustViolations,
    visibleFields,
    flatFields,
    diff,
    rules,
    patternViolations,
    submittable,
    blocked,
    keyOf,
    choiceScope,
    onChoiceUpdate,
    visiblePayload,
    resetForm,
  }
}
