import { reactive, ref, computed, type Ref, type ComputedRef } from 'vue'
import type { FormRules } from 'element-plus'
import { i18n } from '../i18n'
import { useConstraintEngine } from './useConstraintEngine'
import { computeDiff, missingRequired } from '../utils/configDiff'
import type { Field } from '../utils/crdSchemaParser'

// 模型驱动表单编排（自旧配置页收敛的通用逻辑，FE-07/08/09 语义不变）：
// 约束引擎（when 显隐/must 校验）、pattern/range/required 规则、choice 展开、
// 差异比对与可提交门禁、仅可见字段入 payload。供通用控制台的列表/表单 Tab 复用。
export interface UseConfigFormOptions {
  /** 启用 diff 删除表达（FE-22 二期）：基线有值被清 → remove 条目入 diff（攒批链路开启）。 */
  removals?: boolean
}

// 基线深快照：seed 的嵌套对象（group/嵌套 list 行）与 formData 共享引用，浅拷贝
// 会让原位编辑同时改掉基线、changedPayload 永远判「未变」。JSON 深拷贝（数据均为
// 纯 JSON 回读值）切断共享；失败降级浅拷贝（R08，行为等同旧版）。
export function snapshotBaseline(seed: Record<string, any>): Record<string, any> {
  try {
    return JSON.parse(JSON.stringify(seed))
  } catch {
    return { ...seed }
  }
}

export function useConfigForm(
  fields: Ref<Field[]> | ComputedRef<Field[]>,
  keyField?: Ref<string> | ComputedRef<string>,
  opts?: UseConfigFormOptions,
) {
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

  const diff = computed(() => computeDiff(formData, original.value, editableFlat.value, { removals: opts?.removals }))
  // 删除意图叶（FE-22 二期）：基线有值被清除的键，随条目入变更集（cleared）。
  const clearedKeys = computed(() => diff.value.filter((d) => d.op === 'remove').map((d) => d.key))
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
        list.push({ required: true, message: i18n.global.t('console.validation.required', { label: f.label }), trigger: ['change', 'blur'] })
      }
      if (f.type === 'number' && (f.minimum != null || f.maximum != null)) {
        list.push({ type: 'number', min: f.minimum, max: f.maximum, message: i18n.global.t('console.validation.outOfRange', { label: f.label }), trigger: ['change', 'blur'] })
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
        list.push({ pattern: re, message: i18n.global.t('console.validation.pattern', { label: f.label }), trigger: ['change', 'blur'] })
      }
      if (list.length) r[key] = list
    }
    return r
  })

  // 深层排除 readonly 后代（FE-14 补全，NS-08/BR-01）：读路径带回 config=false
  // 状态后，可写 group/嵌套 list 的行对象里会携带 readonly 子叶的设备值；Encode
  // 是 populated-means-pushed，state 叶随 payload 下发真机会被拒绝，须按 schema
  // 递归剥除。schema 未描述的键原样保留（宽进严出，R08）。
  function stripReadonlyDeep(field: Field, value: any): any {
    if (!field.fields?.length || value == null || typeof value !== 'object') return value
    const kids = field.fields
    const stripObj = (obj: Record<string, any>): Record<string, any> => {
      const r: Record<string, any> = {}
      for (const key of Object.keys(obj)) {
        const child = kids.find((c) => keyOf(c) === key)
        if (child?.readonly) continue
        r[key] = child ? stripReadonlyDeep(child, obj[key]) : obj[key]
      }
      return r
    }
    if (Array.isArray(value)) return value.map((v) => (v && typeof v === 'object' && !Array.isArray(v) ? stripObj(v) : v))
    return stripObj(value)
  }

  // 下发 payload：仅当前可见字段的键（when 隐藏 = 节点不存在）；undefined 键剔除
  //（presence 关闭 = 节点不存在，FE-12）；group/list 值按 schema 深层剥除 readonly。
  function visiblePayload(): Record<string, any> {
    const editable = new Map(editableFlat.value.map((f) => [keyOf(f), f]))
    // dynamicDefault 叶空值不入 payload（FE-15）：空=「设备自行决定」，下发空串会覆写设备缺省。
    const dynKeys = new Set(editableFlat.value.filter((f) => f.dynamicDefault).map(keyOf))
    const out: Record<string, any> = {}
    for (const k of Object.keys(formData)) {
      const f = editable.get(k)
      if (!f || formData[k] === undefined) continue
      if (dynKeys.has(k) && (formData[k] === '' || formData[k] === null)) continue
      out[k] = stripReadonlyDeep(f, formData[k])
    }
    return out
  }

  // 值级变更判定：标量沿用 computeDiff 的 norm 口径；对象/数组（group/嵌套 list
  // 原位编辑）按 JSON 序列化比较——norm 会把对象折叠成 "[object Object]" 而失真。
  function valueChanged(now: any, was: any): boolean {
    const isObj = (v: any) => v !== null && typeof v === 'object'
    if (isObj(now) || isObj(was)) {
      try {
        return JSON.stringify(now ?? null) !== JSON.stringify(was ?? null)
      } catch {
        return true // 无法序列化（循环引用等）宁可多发不静默丢改动（R08）
      }
    }
    return (now ?? '').toString().trim() !== (was ?? '').toString().trim()
  }

  // 下发载荷 = 主键 + 相对基线改过的字段（真机 unknown-element 回归）：设备按
  // 接口/节点类型裁剪叶能力（statistic-mode 类），把回读整行原样回推会被拒。
  // 未改动字段不代表用户意图，不入载荷；NETCONF merge 是稀疏语义，只发改动即正确。
  function changedPayload(): Record<string, any> {
    const full = visiblePayload()
    const kf = keyField?.value ?? ''
    const out: Record<string, any> = {}
    for (const k of Object.keys(full)) {
      if ((kf && k === kf) || valueChanged(formData[k], original.value[k])) out[k] = full[k]
    }
    return out
  }

  function resetForm(seed: Record<string, any> = {}) {
    Object.keys(formData).forEach((k) => delete formData[k])
    Object.assign(formData, seed)
    original.value = snapshotBaseline(seed)
  }

  return {
    formData,
    original,
    clearedKeys,
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
    changedPayload,
    resetForm,
  }
}
