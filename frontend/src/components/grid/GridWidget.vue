<template>
  <div :style="gridStyle">
    <template v-if="widget.type === 'table'">
      <h3>{{ widget.label }}</h3>
      <el-table :data="rows" :row-key="widget.rowKey">
        <el-table-column
          v-for="column in widget.columns"
          :key="column.id"
          :prop="column.id"
          :label="column.label"
        />
      </el-table>
    </template>
    <el-empty v-else description="Unsupported widget type" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { GridWidget } from '../../types/grid-schema'

interface Props {
  widget: GridWidget
  modelValue: Record<string, unknown>
  errors?: Record<string, string[]>
}

const props = defineProps<Props>()

defineEmits<{
  'update:modelValue': [value: Record<string, unknown>]
}>()

const gridStyle = computed(() => ({
  gridColumn: `span ${props.widget.grid.span || 12}`
}))

const rows = computed(() => {
  const value = props.modelValue[props.widget.id]
  return Array.isArray(value) ? value : []
})
</script>
