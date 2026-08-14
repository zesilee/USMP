import type { Field } from '../utils/crdSchemaParser'
import { evalPredicate } from '../utils/xpathEval'
import { i18n } from '../i18n'

// 约束引擎（FE-07）纯函数核心：把 schema 中每个字段的 YANG `when` XPath 表达式
// 对当前表单数据求值为显隐结论；must 违例与解析告警同源。100% 数据驱动、零厂商/
// 模型/字段名硬编码；表达式非法时降级为「可见 + 告警」（R08），绝不崩。
// React 侧不做 memo、每渲染重算（design D5：输入是单表单几十字段，重算成本可
// 忽略，换取零陈旧值风险）；hook 外壳见 hooks/useConfigForm。

export interface MustViolation {
  path: string
  label: string
  message: string
}

// 叶子类字段（标量/enum/leaf-list）：未赋值即节点不存在；容器/列表/choice 的
// 存在性由上层 presence 语义处理，不在此判定。enum 曾漏列（statistic-mode 真机
// 二次回归）——与 FieldRenderer 的叶类型口径一致，新增叶类型两处同步。
function isLeafKind(t: Field['type']): boolean {
  return t === 'string' || t === 'number' || t === 'boolean' || t === 'enum' || t === 'leaf-list'
}

function leafSeg(f: Field): string {
  return f.path.split('/').filter(Boolean).pop() || f.path
}

// 未赋值判定：undefined/null/空串（表单清空）/空 leaf-list。false 与 0 是有效值。
function isUnsetLeaf(v: unknown): boolean {
  return v === undefined || v === null || v === '' || (Array.isArray(v) && v.length === 0)
}

function fieldVisible(f: Field, ctx: Record<string, unknown>): boolean {
  if (!f.when) return true
  const r = evalPredicate(f.when, ctx)
  // 解析失败（无 value）→ 降级视为可见。
  return 'value' in r && r.value !== undefined ? r.value : true
}

/** 全字段显隐结论：path → visible（无 when 恒可见）。 */
export function computeVisibleMap(
  fields: Field[],
  formData: Record<string, unknown>,
): Record<string, boolean> {
  const map: Record<string, boolean> = {}
  for (const f of fields ?? []) map[f.path] = fieldVisible(f, formData ?? {})
  return map
}

/**
 * must 违例：仅对当前可见字段（when=false 的节点视为不存在，其 must 不适用）逐条求值。
 * RFC7950 §7.5.3：must 只约束存在的节点——叶子未赋值=节点不存在，其 must 不适用
 * （否则自引用 must 如 ifm statistic-mode 会强迫用户为设备按类型裁剪的叶选值，真机拒收）。
 */
export function computeMustViolations(
  fields: Field[],
  formData: Record<string, unknown>,
): MustViolation[] {
  const ctx = formData ?? {}
  const vmap = computeVisibleMap(fields, ctx)
  const out: MustViolation[] = []
  for (const f of fields ?? []) {
    if (!f.must?.length) continue
    if (!(vmap[f.path] ?? true)) continue // 隐藏字段跳过
    if (isLeafKind(f.type) && isUnsetLeaf(ctx[leafSeg(f)])) continue
    for (const rule of f.must) {
      const r = evalPredicate(rule.expr, ctx)
      if ('value' in r && r.value === false) {
        out.push({
          path: f.path,
          label: f.label,
          message:
            rule.message || i18n.global.t('console.validation.must', { label: f.label, expr: rule.expr }),
        })
      }
    }
  }
  return out
}

/** 告警汇总：when 与 must 表达式解析失败（降级、不阻断，R08）。 */
export function computeWarnings(fields: Field[], formData: Record<string, unknown>): string[] {
  const ctx = formData ?? {}
  const w: string[] = []
  for (const f of fields ?? []) {
    if (f.when) {
      const r = evalPredicate(f.when, ctx)
      if ('error' in r && r.error) w.push(`[${f.path}] when parse failed: ${r.error}`)
    }
    for (const rule of f.must ?? []) {
      const r = evalPredicate(rule.expr, ctx)
      if ('error' in r && r.error) w.push(`[${f.path}] must parse failed: ${r.error}`)
    }
  }
  return w
}
