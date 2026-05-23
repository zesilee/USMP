<template>
  <div class="grid-renderer">
    <div class="grid-toolbar">
      <h2>{{ toolbarTitle }}</h2>
      <span class="capability-source">{{ schema.capabilitySource }}</span>
      <div class="toolbar-buttons">
        <el-button
          data-test="grid-refresh"
          :loading="loading"
          @click="$emit('refresh')"
        >
          刷新
        </el-button>
        <el-button
          type="primary"
          data-test="grid-submit"
          :loading="submitting"
          @click="$emit('submit')"
        >
          提交
        </el-button>
      </div>
    </div>

    <el-alert
      v-if="pageError"
      type="error"
      :title="pageError"
      show-icon
      class="page-error"
    />

    <div class="grid-sections">
      <GridSection
        v-for="section in schema.sections"
        :key="section.id"
        :section="section"
        :widgets="widgetsBySection(section.widgets)"
        :model-value="modelValue"
        :errors="errors"
        @update:model-value="$emit('update:modelValue', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import GridSection from './GridSection.vue'
import type { GridSchema, GridWidget } from '../../types/grid-schema'

interface Props {
  schema: GridSchema
  modelValue: Record<string, unknown>
  errors?: Record<string, string[]>
  loading?: boolean
  submitting?: boolean
  pageError?: string
}

const props = defineProps<Props>()

defineEmits<{
  'update:modelValue': [value: Record<string, unknown>]
  'submit': []
  'refresh': []
}>()

const toolbarTitle = computed(() => {
  return props.schema.sections[0]?.title || '配置管理'
})

const widgetsBySection = (sectionWidgetIds: string[]): GridWidget[] => {
  return props.schema.widgets.filter(w => sectionWidgetIds.includes(w.id))
}
</script>

<style scoped>
.grid-renderer {
  padding: 16px;
}
.grid-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.capability-source {
  color: var(--el-text-color-secondary);
}
.toolbar-buttons {
  display: flex;
  gap: 8px;
}
.page-error {
  margin-bottom: 16px;
}
.grid-sections {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
</style>
