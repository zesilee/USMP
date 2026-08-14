import { fieldValidation } from '../../src/form/antdRules'
import { computeVisibleMap, computeWarnings } from '../../src/form/constraintEngine'
import type { Field } from '../../src/utils/crdSchemaParser'

// 闸门校验探针：对模块全字段树递归求值 fieldValidation（空表单与示例值两态）+
// when/must 表达式解析扫描——任何抛异常即闸门失败（R08：真实 schema 的全部
// pattern/range/must/when 形态必须被稳妥消化，解析失败只允许降级不允许崩）。
export function rulesForGateProbe(fields: Field[]): void {
  const flat: Field[] = []
  const walk = (fs: Field[]) => {
    for (const f of fs) {
      flat.push(f)
      if (f.fields) walk(f.fields)
      for (const c of f.cases || []) walk(c.fields || [])
    }
  }
  walk(fields)

  const sampleValue = (f: Field): unknown => {
    switch (f.type) {
      case 'number':
        return f.minimum ?? 1
      case 'boolean':
        return true
      case 'enum':
        return f.options?.[0]?.value ?? 'x'
      default:
        return 'probe'
    }
  }

  const empty: Record<string, unknown> = {}
  const filled: Record<string, unknown> = {}
  for (const f of flat) {
    const key = f.path.split('/').filter(Boolean).pop() || f.path
    filled[key] = sampleValue(f)
  }

  computeVisibleMap(flat, empty)
  computeVisibleMap(flat, filled)
  computeWarnings(flat, filled) // 解析失败仅落告警，不抛（R08）
  for (const f of flat) {
    fieldValidation(f, flat, empty)
    fieldValidation(f, flat, filled)
  }
}
