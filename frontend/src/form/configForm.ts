import type { Field } from '../utils/crdSchemaParser'
import { computeDiff, missingRequired, type DiffEntry } from '../utils/configDiff'
import { computeVisibleMap, computeMustViolations, type MustViolation } from './constraintEngine'

// 模型驱动表单编排（FE-07/08/09 语义自旧 Vue composable 逐项平移）纯函数核心：
// 约束显隐/must 过滤/pattern 校验/diff 与可提交门禁/仅可见字段入 payload/深层
// 剥除 readonly。全部为 (fields, formData, …) → 结论 的确定性函数，React 侧每
// 渲染重算（design D5）；状态与动作见 hooks/useConfigForm。

export interface ConfigFormOptions {
  /** 启用 diff 删除表达（FE-22 二期）：基线有值被清 → remove 条目入 diff。 */
  removals?: boolean
}

/** path 末段 = 数据键（YANG 叶名，对齐后端转换）。 */
export function keyOf(f: Field): string {
  return f.path.split('/').filter(Boolean).pop() || f.path
}

// ===== choice 展开（成员扁平同级）=====
export function choiceMemberFields(field: Field): Field[] {
  return (field.cases || []).flatMap((c) => c.fields || [])
}

// 编译 YANG pattern；非法正则降级为不校验 + 告警（R08）。
export function compilePattern(pattern?: string): RegExp | null {
  if (!pattern) return null
  try {
    return new RegExp(`^(?:${pattern})$`)
  } catch {
    console.warn('[configForm] invalid YANG pattern, validation skipped:', pattern)
    return null
  }
}

/** 约束引擎（FE-07）：when=false 的字段不渲染、不校验、不入 payload。 */
export function visibleFields(fields: Field[], formData: Record<string, any>): Field[] {
  const vmap = computeVisibleMap(fields, formData)
  return fields.filter((f) => vmap[f.path] ?? true)
}

/** choice 展开后的扁平字段面（可见字段口径）。 */
export function flatFields(fields: Field[], formData: Record<string, any>): Field[] {
  return visibleFields(fields, formData).flatMap((f) =>
    f.type === 'choice' ? choiceMemberFields(f) : [f],
  )
}

/** 可编辑字段面（FE-14）：readonly（config false state）叶可见可回显，但不参与
 *  校验/diff/payload——state 数据永不进设备写路径。 */
export function editableFlat(fields: Field[], formData: Record<string, any>): Field[] {
  return flatFields(fields, formData).filter((f) => !f.readonly)
}

/**
 * must 违例（presence 语义修正，FE-12）：presence 容器未开启（键不存在=节点不存在）
 * 时其 must 不适用；readonly state 叶的 must 不入门禁（FE-14：设备值用户不可改，
 * 违例会永久卡死提交）。
 */
export function mustViolations(fields: Field[], formData: Record<string, any>): MustViolation[] {
  return computeMustViolations(fields, formData).filter((v) => {
    const f = fields.find((x) => x.path === v.path)
    if (f?.readonly) return false
    return !(f?.type === 'group' && f.presence && formData[keyOf(f)] === undefined)
  })
}

export function patternViolations(fields: Field[], formData: Record<string, any>): Field[] {
  return editableFlat(fields, formData).filter((f) => {
    const re = compilePattern(f.pattern)
    if (!re) return false
    const v = formData[keyOf(f)]
    if (v == null || v === '') return false
    return !re.test(String(v))
  })
}

export function computeFormDiff(
  fields: Field[],
  formData: Record<string, any>,
  original: Record<string, any>,
  opts?: ConfigFormOptions,
): DiffEntry[] {
  return computeDiff(formData, original, editableFlat(fields, formData), {
    removals: opts?.removals,
  })
}

/** 删除意图叶（FE-22 二期）：基线有值被清除的键，随条目入变更集（cleared）。 */
export function clearedKeys(
  fields: Field[],
  formData: Record<string, any>,
  original: Record<string, any>,
  opts?: ConfigFormOptions,
): string[] {
  return computeFormDiff(fields, formData, original, opts)
    .filter((d) => d.op === 'remove')
    .map((d) => d.key)
}

/** 权威门禁（§9）：缺必填/must 违例/pattern 违例一律拦截。 */
export function isBlocked(
  fields: Field[],
  formData: Record<string, any>,
  keyField = '',
): boolean {
  return (
    missingRequired(editableFlat(fields, formData), formData, keyField).length > 0 ||
    mustViolations(fields, formData).length > 0 ||
    patternViolations(fields, formData).length > 0
  )
}

export function isSubmittable(
  fields: Field[],
  formData: Record<string, any>,
  original: Record<string, any>,
  keyField = '',
  opts?: ConfigFormOptions,
): boolean {
  return computeFormDiff(fields, formData, original, opts).length > 0 && !isBlocked(fields, formData, keyField)
}

/**
 * 深层排除 readonly 后代（FE-14 补全，NS-08/BR-01）：读路径带回 config=false
 * 状态后，可写 group/嵌套 list 的行对象里会携带 readonly 子叶的设备值；Encode
 * 是 populated-means-pushed，state 叶随 payload 下发真机会被拒绝，须按 schema
 * 递归剥除。schema 未描述的键原样保留（宽进严出，R08）。
 */
export function stripReadonlyDeep(field: Field, value: any): any {
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

/**
 * 下发 payload：仅当前可见字段的键（when 隐藏 = 节点不存在）；undefined 键剔除
 * （presence 关闭 = 节点不存在，FE-12）；group/list 值按 schema 深层剥除 readonly；
 * dynamicDefault 叶空值不入 payload（FE-15：空=「设备自行决定」，下发空串会覆写设备缺省）。
 */
export function visiblePayload(fields: Field[], formData: Record<string, any>): Record<string, any> {
  const editable = new Map(editableFlat(fields, formData).map((f) => [keyOf(f), f]))
  const dynKeys = new Set(
    editableFlat(fields, formData)
      .filter((f) => f.dynamicDefault)
      .map(keyOf),
  )
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
export function valueChanged(now: any, was: any): boolean {
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

/**
 * 下发载荷 = 主键 + 相对基线改过的字段（真机 unknown-element 回归）：设备按
 * 接口/节点类型裁剪叶能力，把回读整行原样回推会被拒。未改动字段不代表用户意图，
 * 不入载荷；NETCONF merge 是稀疏语义，只发改动即正确。
 */
export function changedPayload(
  fields: Field[],
  formData: Record<string, any>,
  original: Record<string, any>,
  keyField = '',
): Record<string, any> {
  const full = visiblePayload(fields, formData)
  const out: Record<string, any> = {}
  for (const k of Object.keys(full)) {
    if ((keyField && k === keyField) || valueChanged(formData[k], original[k])) out[k] = full[k]
  }
  return out
}

// 基线深快照：seed 的嵌套对象与 formData 共享引用会让原位编辑同时改掉基线、
// changedPayload 永远判「未变」。JSON 深拷贝（数据均为纯 JSON 回读值）切断共享；
// 失败降级浅拷贝（R08）。
export function snapshotBaseline(seed: Record<string, any>): Record<string, any> {
  try {
    return JSON.parse(JSON.stringify(seed))
  } catch {
    return { ...seed }
  }
}
