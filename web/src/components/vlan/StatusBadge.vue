<template>
  <span :class="badgeClass">
    <!-- 圆点样式 - 用于运行状态 -->
    <span v-if="variant === 'dot'" class="status-dot" :class="dotClass"></span>
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AdminStatus, OperStatus } from '../../types/vlan'

interface Props {
  type: 'admin' | 'oper'  // 管理状态 / 运行状态
  value: AdminStatus | OperStatus
}

const props = defineProps<Props>()

const statusConfig = computed(() => {
  if (props.type === 'admin') {
    return {
      UP: { label: '启用', variant: 'tag', color: 'success' },
      DOWN: { label: '禁用', variant: 'tag', color: 'info' }
    }
  } else {
    return {
      ACTIVE: { label: '运行中', variant: 'dot', color: 'success' },
      INACTIVE: { label: '未激活', variant: 'dot', color: 'warning' },
      SUSPENDED: { label: '已暂停', variant: 'dot', color: 'error' }
    }
  }
})

const config = computed(() => {
  return statusConfig.value[props.value as keyof typeof statusConfig.value] || {
    label: props.value,
    variant: 'tag',
    color: 'info'
  }
})

const variant = computed(() => config.value.variant)

const badgeClass = computed(() => {
  const base = 'status-badge'
  const variantClass = `status-badge--${config.value.color}`
  return [base, variantClass]
})

const dotClass = computed(() => `status-dot--${config.value.color}`)

const label = computed(() => config.value.label)
</script>

<style lang="scss" scoped>
@import '../../styles/variables.scss';

.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  font-size: $font-size-xs;
  font-weight: $font-weight-medium;
  line-height: 1.4;

  // 不同形状 - 去 AI 味的关键
  &--success {
    color: $color-success;
    background-color: $color-success-bg;
    border-radius: $radius-sm;  // 小方形
  }

  &--warning {
    color: $color-warning;
    background-color: $color-warning-bg;
    border-radius: $radius-full; // 胶囊形
  }

  &--error {
    color: $color-error;
    background-color: $color-error-bg;
    border-radius: 2px; // 更小的方形
  }

  &--info {
    color: $color-info;
    background-color: $color-info-bg;
    border-radius: $radius-md; // 正常圆角
  }

  .status-dot {
    width: 6px;
    height: 6px;
    flex-shrink: 0;

    &--success { background-color: $color-success; border-radius: 50%; }
    &--warning { background-color: $color-warning; border-radius: 1px; } // 菱形感
    &--error { background-color: $color-error; border-radius: 50%; }
    &--info { background-color: $color-info; border-radius: 50%; }
  }
}
</style>
