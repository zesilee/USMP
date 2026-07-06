<template>
  <div
    class="fresh"
    :class="stateClass"
    role="img"
    :aria-label="ariaLabel"
    :title="ariaLabel"
  >
    <svg class="fresh-ring" viewBox="0 0 24 24" aria-hidden="true">
      <circle class="track" cx="12" cy="12" :r="radius" />
      <circle
        class="val"
        cx="12"
        cy="12"
        :r="radius"
        :stroke-dasharray="circumference.toFixed(1)"
        :stroke-dashoffset="dashOffset.toFixed(1)"
      />
    </svg>
    <div class="fresh-txt">
      缓存新鲜度
      <b v-if="hasData" class="mono">{{ remainingLabel }}</b>
      <b v-else class="mono">—</b>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  RING_RADIUS,
  RING_CIRCUMFERENCE,
  computeFreshness,
} from '../../composables/useFreshness'

// 顶栏「缓存新鲜度环」——纯展示组件，全部输入经 props，便于确定性单测。
// 数据来自 PR-B2 的 cache_age_seconds/ttl_seconds（真数据）；无缓存时显示空态「—」。
const props = withDefaults(
  defineProps<{
    ageSeconds?: number
    ttlSeconds?: number
    hasData?: boolean
    source?: string
  }>(),
  {
    ageSeconds: 0,
    ttlSeconds: 30,
    hasData: true,
    source: '',
  },
)

const radius = RING_RADIUS
const circumference = RING_CIRCUMFERENCE

const fresh = computed(() =>
  computeFreshness({ ageSeconds: props.ageSeconds, ttlSeconds: props.ttlSeconds }),
)

// 无数据时不填充环（offset=满周长，弧为空）。
const dashOffset = computed(() =>
  props.hasData ? fresh.value.dashOffset : circumference,
)

const stateClass = computed(() => {
  if (!props.hasData) return 'is-idle'
  if (fresh.value.expired) return 'is-expired'
  if (fresh.value.expiringSoon) return 'is-soon'
  return 'is-fresh'
})

const remainingLabel = computed(
  () => `${fresh.value.remainingSeconds}s / TTL ${props.ttlSeconds}s`,
)

const ariaLabel = computed(() => {
  if (!props.hasData) return '缓存新鲜度：暂无活跃缓存'
  const src = props.source === 'cache' ? '（命中缓存）' : props.source === 'device' ? '（刚回源）' : ''
  const tail = fresh.value.expired ? '，已过期将自动重拉' : `，距过期 ${fresh.value.remainingSeconds} 秒`
  return `缓存新鲜度${src}：TTL ${props.ttlSeconds} 秒${tail}。此为配置缓存年龄，非设备在线状态`
})
</script>

<style scoped>
.fresh {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 5px 11px;
  border-radius: 999px;
  background: var(--sunken);
  border: 1px solid var(--line);
}
.fresh-ring {
  width: 22px;
  height: 22px;
  transform: rotate(-90deg);
}
.fresh-ring .track {
  stroke: var(--line-strong);
  fill: none;
  stroke-width: 2.6;
}
.fresh-ring .val {
  fill: none;
  stroke-width: 2.6;
  stroke-linecap: round;
  transition: stroke-dashoffset 1s linear, stroke 0.4s;
  stroke: var(--st-conv);
}
.fresh.is-soon .fresh-ring .val,
.fresh.is-expired .fresh-ring .val {
  stroke: var(--st-drift);
}
.fresh.is-idle .fresh-ring .val {
  stroke: var(--line-strong);
}
.fresh-txt {
  font-size: 11.5px;
  color: var(--ink-2);
  line-height: 1.2;
}
.fresh-txt b {
  display: block;
  color: var(--ink);
  font-weight: 600;
  font-size: 12px;
}
</style>
