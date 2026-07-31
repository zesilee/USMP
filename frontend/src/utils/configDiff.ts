import type { Field } from './crdSchemaParser'

// 差异操作类型（对齐 NCE 图例：增加/修改/删除，FE-22）。
export type DiffOp = 'add' | 'modify' | 'remove'

// 「下发预览/变更内容」的单条差异（表单期望值 ↔ 已回填的实际值）。
export interface DiffEntry {
  key: string // 数据键（path 末段，与表单 keyOf/formData 同源）
  label: string
  was: any // 原实际值（新增时为空）
  now: any // 新期望值（删除时为空）
  isNew: boolean // 向后兼容 = op==='add'
  op: DiffOp
}

// computeDiff 行为开关。
export interface DiffOptions {
  /**
   * removals=true 时启用删除表达（FE-22 二期语义）：基线有值而表单被清
   * （空串或字段被 clearField 删除）→ remove 条目；基线无值清空仍忽略。
   * 缺省 false = 一期语义字节不变（清空不算改动），供未切换调用方过渡。
   */
  removals?: boolean
}

function segOf(f: Field): string {
  return f.path.split('/').filter(Boolean).pop() || f.path
}

const norm = (v: any): string => (v ?? '').toString().trim()

// 计算表单相对已回填实际态的改动集：「新值非空且与原值不同」→ add/modify；
// removals 开启时「基线有值而新值空」→ remove（FE-22 二期删除表达）。
// 保持 fields 声明顺序。缺省不开 removals = 一期语义字节不变（清空不算改动）。
export function computeDiff(
  formData: Record<string, any> | null | undefined,
  original: Record<string, any> | null | undefined,
  fields: Field[],
  opts?: DiffOptions,
): DiffEntry[] {
  const form = formData ?? {}
  const orig = original ?? {}
  const out: DiffEntry[] = []
  for (const f of fields ?? []) {
    const key = segOf(f)
    const now = norm(form[key])
    const was = norm(orig[key])
    if (now !== '' && now !== was) {
      const op: DiffOp = was === '' ? 'add' : 'modify'
      out.push({ key, label: f.label, was: orig[key], now: form[key], isNew: op === 'add', op })
      continue
    }
    if (opts?.removals && now === '' && was !== '') {
      out.push({ key, label: f.label, was: orig[key], now: undefined, isNew: false, op: 'remove' })
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
