<template>
  <svg v-if="geom" class="spark" :viewBox="`0 0 ${W} ${H}`" role="img" :aria-label="ariaText">
    <polygon class="fillarea" :points="geom.area" />
    <polyline :points="geom.line" />
  </svg>
  <span v-else class="spark-empty" :title="emptyText">—</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  points: number[] | null | undefined
  ariaLabel?: string
  emptyTitle?: string
}>()

const { t } = useI18n()

const ariaText = computed(() => props.ariaLabel ?? t('devices.loadTrend'))
const emptyText = computed(() => props.emptyTitle ?? t('devices.noLoadTelemetry'))

const W = 80
const H = 26

// 由数值序列算出折线 + 填充区坐标（移植原型 spark()）。少于 2 点无法成线 → 空态。
const geom = computed(() => {
  const pts = props.points
  if (!pts || pts.length < 2) return null
  const max = Math.max(...pts)
  const min = Math.min(...pts)
  const nx = (i: number) => (i / (pts.length - 1)) * W
  const ny = (v: number) => H - 2 - ((v - min) / (max - min || 1)) * (H - 6)
  const line = pts.map((v, i) => `${nx(i).toFixed(1)},${ny(v).toFixed(1)}`).join(' ')
  const area = `0,${H} ${line} ${W},${H}`
  return { line, area }
})
</script>

<style scoped>
.spark {
  width: 80px;
  height: 26px;
  display: block;
}

.spark polyline {
  fill: none;
  stroke: var(--primary, #2266cc);
  stroke-width: 1.6;
  stroke-linejoin: round;
  stroke-linecap: round;
}

.spark .fillarea {
  fill: var(--primary-weak, #e6effb);
  opacity: 0.6;
  stroke: none;
}

.spark-empty {
  color: var(--ink-3, #93a2b1);
}
</style>
