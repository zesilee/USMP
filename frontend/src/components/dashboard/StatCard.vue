<template>
  <div class="stat-card">
    <div class="icon-area" :style="{ backgroundColor: iconBg }">
      <component :is="icon" :size="24" :color="iconColor" />
    </div>
    <div class="content-area">
      <div class="stat-value">{{ value }}</div>
      <div class="stat-title">{{ title }}</div>
      <div v-if="trend !== undefined" class="stat-trend" :class="trendClass">
        <span>{{ trendArrow }}</span>
        <span class="trend-value">{{ trend > 0 ? `+${trend}` : trend }}</span>
        <span v-if="trendLabel" class="trend-label">{{ trendLabel }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, type Component } from 'vue'

interface Props {
  title: string
  value: number | string
  icon?: Component
  iconColor?: string
  iconBg?: string
  trend?: number
  trendLabel?: string
}

const props = withDefaults(defineProps<Props>(), {
  iconColor: '#409eff',
  iconBg: '#ecf5ff'
})

const trendClass = computed(() => {
  if (props.trend === undefined) return ''
  return props.trend >= 0 ? 'trend-positive' : 'trend-negative'
})

const trendArrow = computed(() => {
  if (props.trend === undefined) return ''
  return props.trend >= 0 ? '↑' : '↓'
})
</script>

<style scoped>
.stat-card {
  display: flex;
  align-items: center;
  padding: 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.08);
}

.icon-area {
  width: 64px;
  height: 64px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 20px;
  flex-shrink: 0;
}

.content-area {
  flex: 1;
  min-width: 0;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #303133;
  line-height: 1.2;
  margin-bottom: 4px;
}

.stat-title {
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}

.stat-trend {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
}

.trend-positive {
  color: #67c23a;
}

.trend-negative {
  color: #f56c6c;
}

.trend-label {
  color: #909399;
  margin-left: 4px;
}
</style>
