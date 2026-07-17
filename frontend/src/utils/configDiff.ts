import type { Field } from './crdSchemaParser'

// 抽屉「下发预览」的单条差异（表单期望值 ↔ 已回填的实际值）。
export interface DiffEntry {
  key: string // 数据键（path 末段，与表单 keyOf/formData 同源）
  label: string
  was: any // 原实际值（新增时为空）
  now: any // 新期望值
  isNew: boolean // 原值为空 → 新增条目/字段
}

function segOf(f: Field): string {
  return f.path.split('/').filter(Boolean).pop() || f.path
}

const norm = (v: any): string => (v ?? '').toString().trim()

// 计算表单相对已回填实际态的改动集：仅列出「新值非空且与原值不同」的字段，保持 fields 声明顺序。
// 与设计原型 renderPreview 一致——清空字段不作为改动下发（避免误删）。
export function computeDiff(
  formData: Record<string, any> | null | undefined,
  original: Record<string, any> | null | undefined,
  fields: Field[],
): DiffEntry[] {
  const form = formData ?? {}
  const orig = original ?? {}
  const out: DiffEntry[] = []
  for (const f of fields ?? []) {
    const key = segOf(f)
    const now = norm(form[key])
    const was = norm(orig[key])
    if (now !== '' && now !== was) {
      out.push({ key, label: f.label, was: orig[key], now: form[key], isNew: was === '' })
    }
  }
  return out
}

// 必填未填的字段 label 列表（keyField 恒视为必填）。下发按钮 = 有改动 && 无缺失必填。
// dynamicDefault 叶豁免必填（FE-15）：空值=系统自动分配，非缺配置；keyField 例外恒必填。
export function missingRequired(
  fields: Field[],
  formData: Record<string, any> | null | undefined,
  keyField: string,
): string[] {
  const form = formData ?? {}
  const out: string[] = []
  for (const f of fields ?? []) {
    const key = segOf(f)
    const req = (f.required && !f.dynamicDefault) || key === keyField
    if (req && norm(form[key]) === '') out.push(f.label)
  }
  return out
}
