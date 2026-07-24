// 派生结论提取器（GD-01/GD-02）。
//
// 对一个模块的 schema fixture（fields: Field[]，即后端 /yang/schema?form=nested 的
// data.fields，前端零转换直接消费）运行既有控制台派生纯函数，产出「模块 → 控制台
// 形态」的结论对象。此对象即黄金快照的内容。
//
// GD-02 边界：只记派生**结论**，不含 schema 原文副本、不含 i18n 本地化标签。
// - tabs 只取 name+kind（__basic__ 的 label 走 i18n，故丢弃 label；其余 tab name
//   为 leafName，raw YANG 名）。
// - columns 取 leafName+widget（widget=Field.type，即控件类别的判据），isKey/when 的
//   影响体现在列的**取舍与顺序**里（deriveColumns 分层），无需单列 raw 标志。
// - tree 取 deriveSchemaTree 的结构结论（kind/isConfig/isReadonly/dataType/required），
//   这些都是对 schema 的**变换**而非副本。name(label) 丢弃，path 已是稳定标识。
import {
  deriveTabs,
  deriveKeyField,
  deriveColumns,
  filterableFields,
  leafName,
} from '../../src/utils/moduleConsole'
import { deriveSchemaTree } from '../../src/utils/schemaTree'
import type { Field } from '../../src/utils/crdSchemaParser'

export interface ListConclusion {
  keyField: string
  columns: { name: string; widget: Field['type'] }[]
  filterable: string[]
}

export interface TreeNodeConclusion {
  path: string
  kind: 'container' | 'list' | 'leaf'
  depth: number
  dataType?: string
  isConfig: boolean
  isReadonly: boolean
  required: boolean
}

export interface Conclusions {
  module: string
  tabs: { name: string; kind: 'list' | 'form' }[]
  lists: Record<string, ListConclusion>
  tree: TreeNodeConclusion[]
}

// conclusionsFor runs every console-derivation pure function over a module's
// fields and collects their outputs into a stable, i18n-free, diff-ordered
// object. Key order is fixed; arrays preserve derivation order — so JSON.stringify
// is deterministic.
export function conclusionsFor(module: string, fields: Field[]): Conclusions {
  const tabs = deriveTabs(fields)

  const lists: Record<string, ListConclusion> = {}
  for (const t of tabs) {
    if (t.kind === 'list' && t.listField) {
      lists[t.name] = {
        keyField: deriveKeyField(t.listField),
        columns: deriveColumns(t.listField).map((f) => ({ name: leafName(f), widget: f.type })),
        filterable: filterableFields(t.listField).map((f) => leafName(f)),
      }
    }
  }

  const tree: TreeNodeConclusion[] = deriveSchemaTree(fields).map((n) => ({
    path: n.path,
    kind: n.kind,
    depth: n.depth,
    dataType: n.dataType,
    isConfig: n.isConfig,
    isReadonly: n.isReadonly,
    required: n.required,
  }))

  return {
    module,
    tabs: tabs.map((t) => ({ name: t.name, kind: t.kind })),
    lists,
    tree,
  }
}

// serialize renders conclusions to the canonical golden text form (2-space
// indent + trailing newline) — readable line-level diffs on change.
export function serialize(c: Conclusions): string {
  return JSON.stringify(c, null, 2) + '\n'
}
