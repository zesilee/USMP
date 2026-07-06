import type { Field } from './crdSchemaParser'

// YANG 架构树的单个节点（由 /yang/schema?form=nested 的 Field 树派生）。
// 展平成 DFS 前序列表 + depth，供 SchemaTree 以缩进渲染（对齐设计原型的 .ynode）。
export interface SchemaTreeNode {
  path: string
  name: string
  kind: 'container' | 'list' | 'leaf'
  depth: number
  dataType?: string // 叶子的 YANG 数据类型（string/number/boolean/enum）
  isKey: boolean // list 主键叶子
  isConfig: boolean // 可配置叶子（绿色）
  isReadonly: boolean // 只读叶子（灰色）
  required: boolean
}

export interface DeriveSchemaTreeOptions {
  keyField?: string // list 主键叶子名（如 'id' / 'name'），来自 DeviceConfigOptions
}

function segOf(f: Field): string {
  return f.path.split('/').filter(Boolean).pop() || f.path
}

function nameOf(f: Field): string {
  return f.label || segOf(f)
}

// 把嵌套 Field 树 DFS 前序展平为带 depth 的节点列表。
// group→container、list→list、其余→leaf；keyField 仅命中 list 直接子叶子中的同名者。
export function deriveSchemaTree(fields: Field[], opts: DeriveSchemaTreeOptions = {}): SchemaTreeNode[] {
  const out: SchemaTreeNode[] = []
  const walk = (list: Field[], depth: number, parentKind: SchemaTreeNode['kind'] | null): void => {
    for (const f of list ?? []) {
      const kind: SchemaTreeNode['kind'] = f.type === 'group' ? 'container' : f.type === 'list' ? 'list' : 'leaf'
      const isReadonly = !!f.readonly
      out.push({
        path: f.path,
        name: nameOf(f),
        kind,
        depth,
        dataType: kind === 'leaf' ? f.type : undefined,
        // key 匹配走 path 末段（与 DeviceConfigPage.keyOf/校验规则同源），
        // 而非展示用的 label——后端若将来本地化 label，key 徽标仍稳。
        isKey: kind === 'leaf' && parentKind === 'list' && !!opts.keyField && segOf(f) === opts.keyField,
        isConfig: kind === 'leaf' && !isReadonly,
        isReadonly,
        required: !!f.required,
      })
      if (f.fields?.length) walk(f.fields, depth + 1, kind)
    }
  }
  walk(fields, 0, null)
  return out
}
