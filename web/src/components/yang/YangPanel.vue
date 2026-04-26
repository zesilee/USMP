<template>
  <el-card class="yang-panel" shadow="never">
    <template #header>
      <div class="panel-header">
        <span class="panel-title">{{ title }}</span>
        <el-tag v-if="!node.config" size="small" type="info">只读</el-tag>
      </div>
    </template>

    <div class="panel-content">
      <slot />
    </div>
  </el-card>
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
  margin-bottom: 16px;

  .panel-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    font-weight: 500;
    color: var(--text-color-primary);
  }

  .panel-title {
    font-size: 15px;
  }

  :deep(.el-card__body) {
    padding: 16px 20px;
  }
}
</style>
