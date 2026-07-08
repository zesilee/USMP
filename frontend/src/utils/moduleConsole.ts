import type { Field } from './crdSchemaParser'
import { evalPredicate } from './xpathEval'

// 通用模块控制台的纯逻辑层（FE-10/FE-11）：Tab 派生、列派生、search-filter、
// 行级 when 单元格、运行时配置路径派生。全部由 schema 元数据驱动，零模块硬编码。

export interface ConsoleTab {
  name: string
  label: string
  kind: 'list' | 'form'
  /** Tab 对应的模块根子节点（列表 Tab 时为包裹容器或裸 list，configPath 取其 path）。 */
  field: Field
  /** kind==='list' 时的目标 list 节点。 */
  listField?: Field
}

const SCALAR_TYPES = new Set<Field['type']>(['string', 'number', 'boolean', 'enum'])

/** path 末段 = YANG 叶名（数据键，对齐后端转换）。 */
export function leafName(f: Field): string {
  return f.path.split('/').filter(Boolean).pop() || f.path
}

function scalarLeaves(f: Field): Field[] {
  return (f.fields || []).filter((c) => SCALAR_TYPES.has(c.type) && !c.hidden)
}

// 模块根顶层子节点 → 一级 Tab：list（含「group 包裹单 list」的常见形态）→列表页，
// group/choice→表单页；散落根叶子聚合为「基本属性」表单 Tab 排最前（FE-10）。
export function deriveTabs(fields: Field[] | undefined): ConsoleTab[] {
  const tabs: ConsoleTab[] = []
  const looseLeaves: Field[] = []
  for (const f of fields || []) {
    if (SCALAR_TYPES.has(f.type) || f.type === 'leaf-list') {
      looseLeaves.push(f)
      continue
    }
    if (f.type === 'list') {
      tabs.push({ name: leafName(f), label: f.label || leafName(f), kind: 'list', field: f, listField: f })
      continue
    }
    if (f.type === 'group') {
      const kids = (f.fields || []).filter((c) => !c.readonly)
      if (kids.length === 1 && kids[0].type === 'list') {
        tabs.push({ name: leafName(f), label: f.label || leafName(f), kind: 'list', field: f, listField: kids[0] })
        continue
      }
    }
    tabs.push({ name: leafName(f), label: f.label || leafName(f), kind: 'form', field: f })
  }
  if (looseLeaves.length) {
    tabs.unshift({
      name: '__basic__',
      label: '基本属性',
      kind: 'form',
      field: { path: '', type: 'group', label: '基本属性', fields: looseLeaves },
    })
  }
  return tabs
}

/** keyField：isKey 叶优先；缺失时回退首个标量叶（降级，R08）。 */
export function deriveKeyField(listField: Field): string {
  const leaves = scalarLeaves(listField)
  const key = leaves.find((f) => f.isKey)
  return leafName(key || leaves[0] || listField)
}

// 分层取列（层内保持 schema 顺序，跨层去重，封顶 cap）：
// key → identity（operationExclude∋update 的 create-only 标识叶）→ 带 when 的条件叶
// → enum → 其余标量。group/list/choice 子节点不入列（FE-11）。
export function deriveColumns(listField: Field, cap = 9): Field[] {
  const leaves = scalarLeaves(listField)
  const tiers: Field[][] = [
    leaves.filter((f) => f.isKey),
    leaves.filter((f) => f.operationExclude?.includes('update')),
    leaves.filter((f) => !!f.when),
    leaves.filter((f) => f.type === 'enum'),
    leaves,
  ]
  const seen = new Set<string>()
  const out: Field[] = []
  for (const tier of tiers) {
    for (const f of tier) {
      if (out.length >= cap) return out
      if (seen.has(f.path)) continue
      seen.add(f.path)
      out.push(f)
    }
  }
  return out
}

/** 高级搜索字段集：厂商 support-filter 标注的叶（FE-11）。 */
export function filterableFields(listField: Field): Field[] {
  return scalarLeaves(listField).filter((f) => f.supportFilter)
}

// 客户端过滤：空条件跳过；enum 全等；其余子串（大小写不敏感）。组合条件 AND。
export function filterRows(
  rows: Record<string, any>[],
  criteria: Record<string, any>,
  fields: Field[],
): Record<string, any>[] {
  const typeOf = new Map(fields.map((f) => [leafName(f), f.type]))
  const active = Object.entries(criteria).filter(([, v]) => v !== '' && v != null)
  if (!active.length) return rows
  return rows.filter((row) =>
    active.every(([k, v]) => {
      const cell = row[k]
      if (typeOf.get(k) === 'enum') return String(cell) === String(v)
      return String(cell ?? '').toLowerCase().includes(String(v).toLowerCase())
    }),
  )
}

// 行级 when 单元格：以该行数据为上下文求值（../x 即行内兄弟叶）。
// 无 when 恒可见；求值失败降级可见（R08）。
export function cellVisible(col: Field, row: Record<string, any>): boolean {
  if (!col.when) return true
  const r = evalPredicate(col.when, row)
  return 'error' in r && r.error !== undefined ? true : !!r.value
}

// schema path → 运行时配置路径：逐段加模块根名前缀，对齐控制器注册的规范路径
// （/ifm/interfaces → ifm:ifm/ifm:interfaces，与 main.go 的 Prefix 谓词一致）。
export function configPathFor(rootName: string, fieldPath: string): string {
  return fieldPath
    .split('/')
    .filter(Boolean)
    .map((seg) => `${rootName}:${seg}`)
    .join('/')
}

/** 值驱动状态色：up→ok、down→bad，其余无色（不做字段名语义映射）。 */
export function statusTone(v: unknown): 'ok' | 'bad' | '' {
  if (v === 'up') return 'ok'
  if (v === 'down') return 'bad'
  return ''
}
