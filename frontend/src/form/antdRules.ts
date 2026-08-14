import type { Field } from '../utils/crdSchemaParser'
import { keyOf, compilePattern, mustViolations, editableFlat } from './configForm'
import { i18n } from '../i18n'

// 运行时动态校验（FE-02/§9，R05 命门之一）：由 YANG 元数据现场判定违例并生成
// 行内提示。架构决定（切片闸门结论）：**不接 antd Form store 的 rules 引擎**——
// Form.Item name 绑定要求 Form store 做数据源，与 useConfigForm 单一数据源冲突
// （双源同步是 bug 温床）；改用 validateStatus/help 受控展示，校验权威始终在
// form/configForm（isBlocked），行内提示由本模块逐字段现场求得。能力上完全
// 满足 FE-02：required/pattern/range/must 全部运行时生成、失败不提交、行内提示。

const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

export interface FieldValidation {
  status?: 'error'
  help?: string
}

/** number 越界违例（旧 el-form type:number min/max 规则的判定面）。 */
export function rangeViolations(fields: Field[], formData: Record<string, any>): Field[] {
  return editableFlat(fields, formData).filter((f) => {
    if (f.type !== 'number') return false
    const v = formData[keyOf(f)]
    if (v == null || v === '') return false
    const n = Number(v)
    if (Number.isNaN(n)) return true
    if (f.minimum != null && n < f.minimum) return true
    if (f.maximum != null && n > f.maximum) return true
    return false
  })
}

/**
 * 单字段行内校验结论：pattern/range/must 有值即时判定；required 缺失不做行内红
 * （无 touched 语义时会满屏红），由 isBlocked 计入提交门禁（双防线的权威侧）。
 */
export function fieldValidation(
  f: Field,
  fields: Field[],
  formData: Record<string, any>,
): FieldValidation {
  if (f.readonly) return {}
  const key = keyOf(f)
  const v = formData[key]

  if (v != null && v !== '') {
    const re = compilePattern(f.pattern)
    if (re && !re.test(String(v))) {
      return { status: 'error', help: t('console.validation.pattern', { label: f.label }) }
    }
    if (f.type === 'number') {
      const n = Number(v)
      if (
        Number.isNaN(n) ||
        (f.minimum != null && n < f.minimum) ||
        (f.maximum != null && n > f.maximum)
      ) {
        return { status: 'error', help: t('console.validation.outOfRange', { label: f.label }) }
      }
    }
  }

  if (f.must?.length) {
    const hit = mustViolations(fields, formData).find((x) => x.path === f.path)
    if (hit) return { status: 'error', help: hit.message }
  }
  return {}
}
