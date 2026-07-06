import { defineStore } from 'pinia'
import { ref } from 'vue'

/** 后端 GET /config 响应中与缓存新鲜度相关的字段（PR-B2）。 */
export interface CacheMeta {
  cache_age_seconds?: number
  ttl_seconds?: number
  source?: string
}

/** 缓存 TTL 兜底（后端 §8：运行配置缓存 30s）。 */
export const DEFAULT_TTL_SECONDS = 30

/**
 * 新鲜度 store：持有「最近一次读到的运行配置缓存年龄」，供顶栏新鲜度环消费。
 * 由 getConfig 成功后写入（真数据）；本地时钟每秒推进由 useLiveFreshness 负责。
 * 无数据库、纯内存（R03）；跨设备/路径只保留最近一次，顶栏是全局新鲜度指示。
 */
export const useFreshnessStore = defineStore('freshness', () => {
  const ageSeconds = ref(0)
  const ttlSeconds = ref(DEFAULT_TTL_SECONDS)
  const source = ref('')
  const recordedAt = ref(0) // Date.now() ms，记录时刻
  const hasData = ref(false)

  /** 记录一次缓存读结果（来自 getConfig 响应）。缺字段时安全兜底。 */
  function record(meta: CacheMeta) {
    const age = meta.cache_age_seconds
    ageSeconds.value = typeof age === 'number' && age > 0 ? age : 0
    const ttl = meta.ttl_seconds
    ttlSeconds.value = typeof ttl === 'number' && ttl > 0 ? ttl : DEFAULT_TTL_SECONDS
    source.value = meta.source ?? ''
    recordedAt.value = Date.now()
    hasData.value = true
  }

  /** 清空（如切换设备上下文、无活跃缓存时）。 */
  function reset() {
    ageSeconds.value = 0
    ttlSeconds.value = DEFAULT_TTL_SECONDS
    source.value = ''
    recordedAt.value = 0
    hasData.value = false
  }

  return { ageSeconds, ttlSeconds, source, recordedAt, hasData, record, reset }
})
