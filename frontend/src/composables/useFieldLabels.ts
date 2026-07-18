// UI-03：YANG 字段标签本地化——按「语言 + 源模块」懒加载 snd res 副本
// （make sync-snd-i18n 入库），以 YANG 数据路径查表替换 FieldDef.label；
// 任一环节缺失回退原 label（YANG 节点名），界面永不空标签（R08）。
import type { LeftTreeNode } from '../stores/menu'

interface ResEntry {
  name?: string
}

interface LocalizableField {
  path?: string
  label?: string
  fields?: LocalizableField[]
  listCols?: LocalizableField[]
  cases?: { name: string; label?: string; fields?: LocalizableField[] }[]
}

// vite 懒加载 glob：构建期切片，运行时按需拉单模块单语言 json（KB 级）。
const resFiles = import.meta.glob('../assets/snd-i18n/*/*-res.json') as Record<
  string,
  () => Promise<{ default: Record<string, ResEntry> }>
>

const cache = new Map<string, Record<string, ResEntry> | null>()

// loadFieldRes 加载（并缓存）某语言某源模块的 res 映射；缺文件返回 null。
export async function loadFieldRes(locale: string, sourceModule: string): Promise<Record<string, ResEntry> | null> {
  const key = `${locale}/${sourceModule}`
  if (cache.has(key)) return cache.get(key)!
  const path = `../assets/snd-i18n/${locale}/${sourceModule}-res.json`
  const loader = resFiles[path]
  let res: Record<string, ResEntry> | null = null
  if (loader) {
    try {
      res = (await loader()).default
    } catch {
      res = null // 加载异常按缺文件降级（R08）
    }
  }
  cache.set(key, res)
  return res
}

// sourceModuleFor 由根容器名反查源模块名：左树叶（③期载荷 sourceModule/module）
// 命中优先；否则按华为命名约定 huawei-<root> 回退（现有全部生成模块实测吻合）。
export function sourceModuleFor(root: string, leftTree: LeftTreeNode[]): string {
  const found = findByModule(leftTree, root)
  return found || `huawei-${root}`
}

function findByModule(nodes: LeftTreeNode[], root: string): string | '' {
  for (const n of nodes) {
    if (n.sourceModule && n.module === root) return n.sourceModule
    if (n.children) {
      const f = findByModule(n.children, root)
      if (f) return f
    }
  }
  return ''
}

// resKeyFor：FieldDef 扁平路径（/vlan/vlans/vlan/id）→ res 键
// （/huawei-vlan:vlan/vlans/vlan/id）。
export function resKeyFor(sourceModule: string, path: string): string {
  return `/${sourceModule}:${path.replace(/^\//, '')}`
}

// localizeFields 返回替换 label 后的新字段树（不改入参；查不到保留原 label）。
export async function localizeFields<T extends LocalizableField>(
  fields: T[],
  root: string,
  locale: string,
  leftTree: LeftTreeNode[],
): Promise<T[]> {
  const sourceModule = sourceModuleFor(root, leftTree)
  const res = await loadFieldRes(locale, sourceModule)
  if (!res) return fields
  const relabel = <F extends LocalizableField>(f: F): F => {
    const out: F = { ...f }
    if (f.path) {
      const hit = res[resKeyFor(sourceModule, f.path)]
      if (hit?.name) out.label = hit.name
    }
    if (f.fields) out.fields = f.fields.map(relabel)
    if (f.listCols) out.listCols = f.listCols.map(relabel)
    if (f.cases) out.cases = f.cases.map((c) => ({ ...c, fields: c.fields?.map(relabel) }))
    return out
  }
  return fields.map(relabel)
}
