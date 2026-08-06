import type { Field } from './crdSchemaParser'
import { evalPredicate } from './xpathEval'
import { i18n } from '../i18n'

// 通用模块控制台的纯逻辑层（FE-10/FE-11）：Tab 派生、列派生、search-filter、
// 行级 when 单元格、运行时配置路径派生。全部由 schema 元数据驱动，零模块硬编码。

export interface ConsoleTab {
  name: string
  label: string
  kind: 'list' | 'form' | 'rpc'
  /** Tab 对应的模块根子节点（列表 Tab 时为包裹容器或裸 list，configPath 取其 path）。 */
  field: Field
  /** kind==='list' 时的目标 list 节点。 */
  listField?: Field
  /** 整棵子树为 state 数据（config false，FE-14）：降级只读视图，无编辑/下发入口。 */
  readonly?: boolean
  /** kind==='rpc' 时的 rpc 定义（执行面板据此渲染输入与执行，FE-19）。 */
  rpc?: RpcDef
}

/** 一个 rpc 的前端呈现契约（FE-19，对齐后端 RPCSchema）：input 复用 Field 渲染。 */
export interface RpcDef {
  name: string
  label: string
  highRisk?: boolean
  input: Field[]
}

// rpc 与模块顶层 container 平级呈现（FE-19）：每个 rpc 派生一个 kind==='rpc' 的 Tab，
// 携带 rpc 定义；field 为合成占位（Tab 契约要求非空），执行面板只读 tab.rpc。
export function deriveRpcTabs(rpcs: RpcDef[] | undefined): ConsoleTab[] {
  return (rpcs || []).map((r) => ({
    name: '__rpc__' + r.name,
    label: r.label || r.name,
    kind: 'rpc' as const,
    field: { path: '', type: 'group', label: r.label || r.name, fields: r.input },
    rpc: r,
  }))
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
// readonly 子树（config false state 数据）照常派生但整 Tab 标只读（FE-14）——
// 降级为可查看视图而非隐藏，state 数据仍有查看价值。
export function deriveTabs(fields: Field[] | undefined): ConsoleTab[] {
  const tabs: ConsoleTab[] = []
  const looseLeaves: Field[] = []
  for (const f of fields || []) {
    const ro = !!f.readonly
    if (SCALAR_TYPES.has(f.type) || f.type === 'leaf-list') {
      looseLeaves.push(f)
      continue
    }
    if (f.type === 'list') {
      tabs.push({ name: leafName(f), label: f.label || leafName(f), kind: 'list', field: f, listField: f, readonly: ro })
      continue
    }
    if (f.type === 'group') {
      // 只读 group 整棵同源只读，list 包裹判定无须再按 readonly 过滤子节点。
      const kids = (f.fields || []).filter((c) => ro || !c.readonly)
      if (kids.length === 1 && kids[0].type === 'list') {
        tabs.push({ name: leafName(f), label: f.label || leafName(f), kind: 'list', field: f, listField: kids[0], readonly: ro })
        continue
      }
    }
    tabs.push({ name: leafName(f), label: f.label || leafName(f), kind: 'form', field: f, readonly: ro })
  }
  if (looseLeaves.length) {
    const basicLabel = i18n.global.t('console.basicTab')
    tabs.unshift({
      name: '__basic__',
      label: basicLabel,
      kind: 'form',
      field: { path: '', type: 'group', label: basicLabel, fields: looseLeaves },
      readonly: looseLeaves.every((f) => !!f.readonly),
    })
  }
  return tabs
}

// 详情二级 Tab（FE-21）：list 条目的非容器子节点（标量/leaf-list/choice）聚合为
// 首个主表单 Tab；嵌套 group→子表单 Tab、嵌套 list（含 group 包裹单 list）→子表格
// Tab，schema 序。无嵌套时退化为单主 Tab；fields 缺失降级空表单不崩（R08）。
export function deriveDetailTabs(listField: Field): ConsoleTab[] {
  const mainFields: Field[] = []
  const subTabs: ConsoleTab[] = []
  for (const f of listField.fields || []) {
    if (f.type !== 'group' && f.type !== 'list') {
      mainFields.push(f)
      continue
    }
    const ro = !!f.readonly
    if (f.type === 'list') {
      subTabs.push({ name: leafName(f), label: f.label || leafName(f), kind: 'list', field: f, listField: f, readonly: ro })
      continue
    }
    const kids = (f.fields || []).filter((c) => ro || !c.readonly)
    if (kids.length === 1 && kids[0].type === 'list') {
      subTabs.push({ name: leafName(f), label: f.label || leafName(f), kind: 'list', field: f, listField: kids[0], readonly: ro })
      continue
    }
    subTabs.push({ name: leafName(f), label: f.label || leafName(f), kind: 'form', field: f, readonly: ro })
  }
  const label = listField.label || leafName(listField)
  const main: ConsoleTab = {
    name: '__main__',
    label,
    kind: 'form',
    field: { path: listField.path, type: 'group', label, fields: mainFields },
    readonly: !!listField.readonly,
  }
  return [main, ...subTabs]
}

/** 可用列全集（FE-11 列设置）：同分层启发式、不封顶——默认集恒为其前缀。 */
export function deriveAllColumns(listField: Field): Field[] {
  return deriveColumns(listField, Infinity)
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

// 服务端过滤参数（FE-25）：把高级搜索条件下推为 BR-13 filter 语法，
// 语义与 filterRows 一一对应——enum 全等（==）、其余包含（~=），空条件跳过。
export function buildServerFilters(criteria: Record<string, any>, fields: Field[]): string[] {
  const typeOf = new Map(fields.map((f) => [leafName(f), f.type]))
  return Object.entries(criteria)
    .filter(([, v]) => v !== '' && v != null)
    .map(([k, v]) => (typeOf.get(k) === 'enum' ? `${k}==${v}` : `${k}~=${v}`))
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
