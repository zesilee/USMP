import { ref } from 'vue'
import { getConfig, setConfig, getYangSchema } from '../api'
import type { Field } from '../utils/crdSchemaParser'
import { useFreshnessStore } from '../stores/freshness'

// 通用设备配置流（Stack B 直连）。任意华为模块（vlan/ifm/...）只需提供下述参数即可
// 用同一套「schema 动态渲染 + 列表 + 下发」界面。schema/list/下发均走 api 客户端。
export interface DeviceConfigOptions {
  module: string // schema 模块键，如 'vlan' / 'ifm'
  configPath: string // 配置 API 路径（须含 vlan:/ifm:ifm 等以路由到后端转换器）
  itemListSuffix: string // 目标 list 的 path 后缀，如 '/vlan' / '/interface'
  listKey: string // POST body 包裹 list 的键，如 'vlans' / 'interface'
  keyField: string // 单条记录主键叶子名，如 'id' / 'name'
}

// DFS 找到 path 以 suffix 结尾的 list 字段节点（供取字段集与 path 复用）。
function findItemList(fields: Field[], suffix: string): Field | null {
  for (const f of fields ?? []) {
    if (f.type === 'list' && f.path.endsWith(suffix)) return f
    if (f.fields) {
      const r = findItemList(f.fields, suffix)
      if (r) return r
    }
  }
  return null
}

// DFS 找到 path 以 suffix 结尾的 list 字段，返回其子字段（单条记录的字段集）。
export function extractItemFields(schema: any, suffix: string): Field[] {
  return findItemList(schema?.fields ?? [], suffix)?.fields ?? []
}

// DFS 找到目标 list 的完整 path（供架构树的数量 pill 定位到该 list 节点）。
export function findItemListPath(schema: any, suffix: string): string {
  return findItemList(schema?.fields ?? [], suffix)?.path ?? ''
}

// 从运行配置归一化出行数组（兼容 {listKey:[...]}、数组、以主键为键的 map）。
export function extractRows(data: any, listKey: string, keyField: string): Record<string, any>[] {
  const payload = data?.data ?? data
  const rows = payload?.[listKey] ?? payload
  if (Array.isArray(rows)) return rows
  if (rows && typeof rows === 'object') {
    return Object.entries(rows).map(([k, v]) =>
      typeof v === 'object' && v !== null
        ? { [keyField]: isNaN(Number(k)) ? k : Number(k), ...(v as object) }
        : { [keyField]: k },
    )
  }
  return []
}

export function useDeviceConfig(opts: DeviceConfigOptions) {
  const fields = ref<Field[]>([])
  const schemaFields = ref<Field[]>([]) // 完整嵌套 schema 树（供架构树 SchemaTree 渲染）
  const itemListPath = ref('') // 目标 list 节点的完整 path（供架构树数量 pill 定位）
  const items = ref<Record<string, any>[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function loadSchema() {
    // 走 api 客户端（绝对 baseURL）——staging 的 nginx 不代理 /api，裸相对 fetch 会拿到 index.html。
    const res = await getYangSchema(opts.module, 'nested')
    const data = res.data?.data
    schemaFields.value = data?.fields ?? []
    fields.value = extractItemFields(data, opts.itemListSuffix)
    itemListPath.value = findItemListPath(data, opts.itemListSuffix)
  }

  async function loadItems(ip: string) {
    loading.value = true
    error.value = null
    try {
      const res = await getConfig(ip, opts.configPath)
      const payload = res.data?.data
      // 生产者：把后端返回的缓存新鲜度（PR-B2 真数据）写入 store，供顶栏新鲜度环消费。
      // cache_age_seconds/ttl_seconds/source 与配置负载同为 ConfigGetData 的兄弟字段。
      useFreshnessStore().record({
        cache_age_seconds: payload?.cache_age_seconds,
        ttl_seconds: payload?.ttl_seconds,
        source: payload?.source,
      })
      items.value = extractRows(payload, opts.listKey, opts.keyField)
    } catch (e: any) {
      error.value = e?.response?.data?.message || e?.message || '读取失败'
      items.value = []
    } finally {
      loading.value = false
    }
  }

  async function saveItem(ip: string, item: Record<string, any>) {
    await setConfig(ip, opts.configPath, { [opts.listKey]: [item] })
  }

  return { fields, schemaFields, itemListPath, items, loading, error, loadSchema, loadItems, saveItem }
}
