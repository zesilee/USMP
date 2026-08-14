import { useCallback, useEffect, useState } from 'react'
import { getConfig, type ListQuery } from '../api'
import { useFreshnessStore } from '../stores/freshness'
import { nodeUnsupportedFromEnvelope, nodeUnsupportedFromError } from '../utils/nodeSupport'
import { buildServerFilters, leafName } from '../utils/moduleConsole'
import type { Field } from '../utils/crdSchemaParser'
import { i18n } from '../i18n'

// 列表取数编排 hook（FE-24/FE-25，自 ModuleListTab 抽出便于单测与瘦身）：
// requestRows 统一收口（只读 Tab <get>/信封不支持学习/带参被拒回退旧读法/新鲜度
// 埋点）+ 双模式分页（首读 limit=200 探测：total>阈值转服务端，翻页/搜索/排序
// 下推 BR-13；否则客户端整树）。语义自旧 Vue 版逐行平移。
const t = (k: string, p?: Record<string, unknown>) => i18n.global.t(k, p)

export const SERVER_PAGE_THRESHOLD = 200

// 分页模式响应形状判定（BR-13 ListPage）。
export function isListPage(body: any): body is { rows: Record<string, any>[]; total: number } {
  return !!body && typeof body === 'object' && Array.isArray(body.rows) && typeof body.total === 'number'
}

export interface SortState {
  prop: string
  desc: boolean
}

export interface UseListQueryArgs {
  device: string
  configPath: string
  readonlyTab: boolean
  listField: Field
  searchFields: Field[]
  /** 整树形状 → 行数组归一化（normalizeRows，调用方注入避免循环依赖）。 */
  normalize: (subtree: any) => { rows: Record<string, any>[]; key: string }
  /** 预标记不支持（CN-05）。 */
  unsupported?: boolean
}

export function useListQuery(args: UseListQueryArgs) {
  const { device, configPath, readonlyTab, listField, searchFields, normalize, unsupported } = args
  const freshness = useFreshnessStore()

  const [items, setItems] = useState<Record<string, any>[]>([])
  const [postKey, setPostKey] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [queryAt, setQueryAt] = useState('')
  const [nodeUnsupported, setNodeUnsupported] = useState(!!unsupported)
  useEffect(() => setNodeUnsupported(!!unsupported), [unsupported])

  const [serverMode, setServerMode] = useState(false)
  const [serverTotal, setServerTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(10)
  const [sortState, setSortState] = useState<SortState | null>(null)
  const [applied, setApplied] = useState<Record<string, any>>({})

  const requestRows = useCallback(
    async (force: boolean, query?: ListQuery): Promise<any> => {
      const res = await getConfig(device, configPath, force, readonlyTab, query)
      if (nodeUnsupportedFromEnvelope(res.data)) {
        setNodeUnsupported(true)
        return undefined
      }
      setNodeUnsupported(false)
      // 带参被拒（如该路径实际非 list，信封 400）：回退旧读法一次（R08 降级）。
      if (query && res.data?.success === false) {
        return requestRows(force, undefined)
      }
      const payload = res.data?.data
      freshness.record({
        cache_age_seconds: payload?.cache_age_seconds,
        ttl_seconds: payload?.ttl_seconds,
        source: payload?.source,
      })
      return payload?.data ?? payload
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [device, configPath, readonlyTab],
  )

  const handleLoadError = useCallback((e: any, force: boolean) => {
    if (nodeUnsupportedFromError(e)) {
      setNodeUnsupported(true)
      return
    }
    setError(
      e?.response?.data?.message || e?.message || (force ? t('console.fetchSourceFailed') : t('console.readFailed')),
    )
    // §9：强制回读失败保留原列表；常规加载失败清空避免陈旧数据误导。
    if (!force) setItems([])
  }, [])

  // 服务端模式取一页：翻页/每页/搜索/排序全部下推（FE-25）。next 覆盖参用于
  // 事件回调里携带最新值（setState 异步，不可读旧闭包）。
  const pageLoad = useCallback(
    async (
      force = false,
      next?: { page?: number; pageSize?: number; sort?: SortState | null; applied?: Record<string, any> },
    ) => {
      setLoading(true)
      setError('')
      const p = next?.page ?? page
      const ps = next?.pageSize ?? pageSize
      const st = next?.sort !== undefined ? next.sort : sortState
      const ap = next?.applied ?? applied
      try {
        const body = await requestRows(force, {
          limit: ps,
          offset: (p - 1) * ps,
          filters: buildServerFilters(ap, searchFields),
          sort: st?.prop || undefined,
          sortDir: st?.desc ? 'desc' : 'asc',
        })
        if (body === undefined) return
        if (isListPage(body)) {
          setItems(body.rows)
          setServerTotal(body.total)
          setPostKey((k) => k || leafName(listField))
        }
        setQueryAt(new Date().toLocaleString())
      } catch (e: any) {
        handleLoadError(e, force)
      } finally {
        setLoading(false)
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [requestRows, page, pageSize, sortState, applied, searchFields, listField],
  )

  const load = useCallback(
    async (force = false) => {
      if (!device) {
        setItems([])
        return
      }
      if (nodeUnsupported && !force) return // FE-24：占位态零请求
      setLoading(true)
      setError('')
      try {
        const body = await requestRows(force, { limit: SERVER_PAGE_THRESHOLD, offset: 0 })
        if (body === undefined) return
        if (isListPage(body)) {
          setServerTotal(body.total)
          if (body.total > SERVER_PAGE_THRESHOLD) {
            // 大表 → 服务端模式：取当前页窗口（切自后端快照，零设备往返）。
            setServerMode(true)
            const p = force ? 1 : page
            if (force) setPage(1)
            await pageLoad(force, { page: p })
            return
          }
          setServerMode(false)
          setItems(body.rows)
          setPostKey(leafName(listField))
        } else {
          // 整树形状（回退旧读法/旧后端）：客户端路径。
          setServerMode(false)
          const { rows, key } = normalize(body)
          setItems(rows)
          setPostKey(key)
        }
        setQueryAt(new Date().toLocaleString())
      } catch (e: any) {
        handleLoadError(e, force)
      } finally {
        setLoading(false)
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [device, requestRows, pageLoad, listField, normalize, nodeUnsupported, page],
  )

  return {
    items,
    postKey,
    loading,
    error,
    queryAt,
    nodeUnsupported,
    serverMode,
    serverTotal,
    page,
    setPage,
    pageSize,
    setPageSize,
    sortState,
    setSortState,
    applied,
    setApplied,
    load,
    pageLoad,
  }
}
