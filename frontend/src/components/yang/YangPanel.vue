<template>
  <div class="yang-panel">
    <div class="panel-header">
      <span class="panel-title">{{ title }}</span>
      <el-tag v-if="!node.config" size="small" type="info">只读</el-tag>
    </div>
    <div class="panel-content">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { YangNode } from '../../types/yang-schema'

interface Props {
  node: YangNode
}

const props = defineProps<Props>()

const title = computed(() =>
  props.node.description || props.node.name
)
</script>

<style lang="scss" scoped>
.yang-panel {
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  background-color: var(--bg-elevated);
  margin-bottom: var(--spacing-xl);

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--spacing-lg) var(--spacing-xl);
    border-bottom: 1px solid var(--border-color);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);

    .panel-title {
      font-size: var(--font-size-base);
    }
  }

  .panel-content {
    padding: var(--spacing-xl);
  }
}
</style>
