<template>
  <el-card>
    <template #header>
      <div class="card-header">
        <span>{{ section.title }}</span>
        <span v-if="section.description" class="description">{{ section.description }}</span>
      </div>
    </template>
    <div class="grid-section-body">
      <GridWidget
        v-for="widget in sectionWidgets"
        :key="widget.id"
        :widget="widget"
        :model-value="modelValue"
        :errors="errors"
        @update:model-value="$emit('update:modelValue', $event)"
      />
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import GridWidget from './GridWidget.vue'
import type { GridSection as IGridSection, GridWidget as IGridWidget } from '../../types/grid-schema'

interface Props {
  section: IGridSection
  widgets: IGridWidget[]
  modelValue: Record<string, unknown>
  errors?: Record<string, string[]>
}

const props = defineProps<Props>()

defineEmits<{
  'update:modelValue': [value: Record<string, unknown>]
}>()

const sectionWidgets = computed(() => {
  return props.widgets.filter(w => props.section.widgets.includes(w.id))
})
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.description {
  color: var(--el-text-color-secondary);
  font-size: 14px;
}
.grid-section-body {
  display: grid;
  grid-template-columns: repeat(24, 1fr);
  gap: 16px;
}
</style>
