import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import type { Field } from '../utils/crdSchemaParser'
import { evalPredicate } from '../utils/xpathEval'
import { i18n } from '../i18n'

// 叶子类字段（标量/leaf-list）：未赋值即节点不存在；容器/列表/choice 的存在性
// 由上层 presence 语义处理，不在此判定。
function isLeafKind(t: Field['type']): boolean {
  return t === 'string' || t === 'number' || t === 'boolean' || t === 'leaf-list'
}

function leafSeg(f: Field): string {
  return f.path.split('/').filter(Boolean).pop() || f.path
}

// 未赋值判定：undefined/null/空串（表单清空）/空 leaf-list。false 与 0 是有效值。
function isUnsetLeaf(v: any): boolean {
  return v === undefined || v === null || v === '' || (Array.isArray(v) && v.length === 0)
}

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

  function isVisible(f: Field): boolean {
    return visibleMap.value[f.path] ?? true
  }

  // must 违例：仅对当前可见字段（when=false 的节点视为不存在，其 must 不适用）逐条求值。
  // 违反 → 收集 { path, label, message }（message 兜底：优先 YANG 提示，否则生成含标签的通用提示）。
  const mustViolations = computed<{ path: string; label: string; message: string }[]>(() => {
    const ctx = toValue(formData) ?? {}
    const vmap = visibleMap.value
    const out: { path: string; label: string; message: string }[] = []
    for (const f of toValue(fields) ?? []) {
      if (!f.must?.length) continue
      if (!(vmap[f.path] ?? true)) continue // 隐藏字段跳过
      // RFC7950 §7.5.3：must 只约束存在的节点。叶子未赋值=节点不存在，其 must
      // 不适用（与 when=false 同理；presence 容器由上层过滤）。否则自引用 must
      //（如 ifm statistic-mode）会强迫用户为设备按类型裁剪的叶选值，真机拒收。
      if (isLeafKind(f.type) && isUnsetLeaf(ctx[leafSeg(f)])) continue
      for (const rule of f.must) {
        const r = evalPredicate(rule.expr, ctx)
        if ('value' in r && r.value === false) {
          out.push({ path: f.path, label: f.label, message: rule.message || i18n.global.t('console.validation.must', { label: f.label, expr: rule.expr }) })
        }
      }
    }
    return out
  })

  // 告警汇总：when 与 must 表达式解析失败（降级、不阻断）。
  const warnings = computed<string[]>(() => {
    const ctx = toValue(formData) ?? {}
    const w: string[] = []
    for (const f of toValue(fields) ?? []) {
      if (f.when) {
        const r = evalPredicate(f.when, ctx)
        if ('error' in r && r.error) w.push(`[${f.path}] when 解析失败：${r.error}`)
      }
      for (const rule of f.must ?? []) {
        const r = evalPredicate(rule.expr, ctx)
        if ('error' in r && r.error) w.push(`[${f.path}] must 解析失败：${r.error}`)
      }
    }
    return w
  })

  return { visibleMap, warnings, mustViolations, isVisible }
}
