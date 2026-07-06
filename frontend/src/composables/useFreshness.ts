import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useFreshnessStore } from '../stores/freshness'

// 缓存新鲜度（freshness）——反映「运行配置读缓存」距 TTL 过期还剩多久。
// 数据源：后端 PR-B2 的 GET /config 响应字段 cache_age_seconds / ttl_seconds / source。
//
// 语义边界（重要）：freshness ≠ liveness。
//   新鲜度只描述「本地缓存有多旧」；设备在线/离线判定走 /devices/:ip/status，
//   不由此环兼任（命中缓存 30s 内即使设备离线，缓存仍新鲜）。

/** 接近过期阈值：已消耗 ≥80% TTL 即视为「即将过期」（原型 24/30s）。 */
export const EXPIRING_THRESHOLD = 0.8

/** 新鲜度环半径（SVG viewBox 24×24，与原型一致）。 */
export const RING_RADIUS = 9
/** 环周长 = 2πr ≈ 56.55，作为 stroke-dasharray。 */
export const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS

export interface FreshnessInput {
  ageSeconds: number
  ttlSeconds: number
}

export interface FreshnessState {
  /** 已消耗 TTL 比例，clamp 到 [0,1]。 */
  fraction: number
  /** 距过期剩余秒，max(ttl-age,0)。 */
  remainingSeconds: number
  /** 是否接近过期（fraction ≥ 阈值）。 */
  expiringSoon: boolean
  /** 是否已过期（age ≥ ttl）。 */
  expired: boolean
  /** 环的 stroke-dashoffset：随新鲜度消耗，弧从满到空。 */
  dashOffset: number
}

/**
 * 纯函数：由缓存年龄 + TTL 计算新鲜度环所需的全部派生量。
 * 无副作用、不依赖时钟，便于确定性单测（TTL 边界 0/30、颜色阈值）。
 */
export function computeFreshness({ ageSeconds, ttlSeconds }: FreshnessInput): FreshnessState {
  const ttl = Number.isFinite(ttlSeconds) && ttlSeconds > 0 ? ttlSeconds : 0
  const age = Number.isFinite(ageSeconds) && ageSeconds > 0 ? ageSeconds : 0

  // TTL 非法（≤0）：视为已过期，环填满、剩 0。
  if (ttl <= 0) {
    return {
      fraction: 1,
      remainingSeconds: 0,
      expiringSoon: true,
      expired: true,
      dashOffset: RING_CIRCUMFERENCE,
    }
  }

  const fraction = Math.min(age / ttl, 1)
  return {
    fraction,
    remainingSeconds: Math.max(ttl - age, 0),
    expiringSoon: fraction >= EXPIRING_THRESHOLD,
    expired: age >= ttl,
    dashOffset: RING_CIRCUMFERENCE * fraction,
  }
}

/**
 * 纯函数：由「记录时缓存年龄 + 记录时刻 + 当前时刻」推算实时缓存年龄（秒）。
 * 前端本地每秒递增，真实反映本地缓存新鲜度，无需后端持续推送。
 */
export function liveAgeSeconds(baseAgeSeconds: number, recordedAtMs: number, nowMs: number): number {
  const elapsed = Math.max(0, Math.floor((nowMs - recordedAtMs) / 1000))
  return Math.max(0, baseAgeSeconds) + elapsed
}

/**
 * 组合式：把新鲜度 store（真数据）+ 本地每秒时钟组合为响应式新鲜度状态。
 * 挂载时启动 1s 定时器，卸载时清理（无泄漏）。供 Header 顶栏新鲜度环消费。
 */
export function useLiveFreshness(intervalMs = 1000) {
  const store = useFreshnessStore()
  const nowMs = ref(Date.now())
  let timer: ReturnType<typeof setInterval> | null = null

  onMounted(() => {
    timer = setInterval(() => {
      nowMs.value = Date.now()
    }, intervalMs)
  })
  onBeforeUnmount(() => {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  })

  const hasData = computed(() => store.hasData)
  const ttlSeconds = computed(() => store.ttlSeconds)
  const source = computed(() => store.source)
  const ageSeconds = computed(() =>
    store.hasData ? liveAgeSeconds(store.ageSeconds, store.recordedAt, nowMs.value) : 0,
  )
  const state = computed(() =>
    computeFreshness({ ageSeconds: ageSeconds.value, ttlSeconds: ttlSeconds.value }),
  )

  return { hasData, ageSeconds, ttlSeconds, source, state }
}
