import { create } from './createStore'

/** 后端 GET /config 响应中与缓存新鲜度相关的字段（PR-B2）。 */
export interface CacheMeta {
  cache_age_seconds?: number
  ttl_seconds?: number
  source?: string
}

/** 缓存 TTL 兜底（后端 §8：运行配置缓存 30s）。 */
export const DEFAULT_TTL_SECONDS = 30

interface FreshnessState {
  ageSeconds: number
  ttlSeconds: number
  source: string
  recordedAt: number // Date.now() ms，记录时刻
  hasData: boolean
  record: (meta: CacheMeta) => void
  reset: () => void
}

/**
 * 新鲜度 store：持有「最近一次读到的运行配置缓存年龄」，供顶栏新鲜度环消费。
 * 由 getConfig 成功后写入（真数据）；本地时钟推进由 useLiveFreshness（React 重建
 * 随顶栏恢复）负责。无数据库、纯内存（R03）；跨设备/路径只保留最近一次。
 */
export const useFreshnessStore = create<FreshnessState>((set) => ({
  ageSeconds: 0,
  ttlSeconds: DEFAULT_TTL_SECONDS,
  source: '',
  recordedAt: 0,
  hasData: false,

  /** 记录一次缓存读结果（来自 getConfig 响应）。缺字段时安全兜底。 */
  record: (meta) => {
    const age = meta.cache_age_seconds
    const ttl = meta.ttl_seconds
    set({
      ageSeconds: typeof age === 'number' && age > 0 ? age : 0,
      ttlSeconds: typeof ttl === 'number' && ttl > 0 ? ttl : DEFAULT_TTL_SECONDS,
      source: meta.source ?? '',
      recordedAt: Date.now(),
      hasData: true,
    })
  },

  /** 清空（如切换设备上下文、无活跃缓存时）。 */
  reset: () =>
    set({
      ageSeconds: 0,
      ttlSeconds: DEFAULT_TTL_SECONDS,
      source: '',
      recordedAt: 0,
      hasData: false,
    }),
}))
