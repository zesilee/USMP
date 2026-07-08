import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import type { Field } from '../utils/crdSchemaParser'
import { evalPredicate } from '../utils/xpathEval'

// useConstraintEngine —— 通用约束引擎（FE-07）。把 schema 中每个字段的 YANG `when`
// XPath 表达式对当前表单数据求值为响应式 `visible` 状态。100% 数据驱动、零厂商/
// 模型/字段名硬编码；表达式非法时降级为「可见 + 告警」（R08），绝不崩。
//
// 表单数据以 YANG 叶子名为键（path 末段），与 when 的 `../leaf` 相对引用对齐。
// fields/formData 均接受 ref / getter / 普通值 / reactive（经 toValue 归一，保持响应式）。
export function useConstraintEngine(
  fields: MaybeRefOrGetter<Field[]>,
  formData: MaybeRefOrGetter<Record<string, any>>,
) {
  function fieldVisible(f: Field, ctx: Record<string, any>): boolean {
    if (!f.when) return true
    const r = evalPredicate(f.when, ctx)
    // 解析失败（无 value）→ 降级视为可见。
    return 'value' in r && r.value !== undefined ? r.value : true
  }

  const visibleMap = computed<Record<string, boolean>>(() => {
    const ctx = toValue(formData) ?? {}
    const map: Record<string, boolean> = {}
    for (const f of toValue(fields) ?? []) {
      map[f.path] = fieldVisible(f, ctx)
    }
    return map
  })

  const warnings = computed<string[]>(() => {
    const ctx = toValue(formData) ?? {}
    const w: string[] = []
    for (const f of toValue(fields) ?? []) {
      if (!f.when) continue
      const r = evalPredicate(f.when, ctx)
      if ('error' in r && r.error) w.push(`[${f.path}] when 解析失败：${r.error}`)
    }
    return w
  })

  function isVisible(f: Field): boolean {
    return visibleMap.value[f.path] ?? true
  }

  return { visibleMap, warnings, isVisible }
}
