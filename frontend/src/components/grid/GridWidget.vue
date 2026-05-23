<template>
  <div :style="gridStyle">
    <template v-if="widget.type === 'table'">
      <div class="grid-table-header">
        <h3>{{ widget.label }}</h3>
      </div>
      <el-table :data="rows" :row-key="widget.rowKey">
        <el-table-column
          v-for="column in widget.columns"
          :key="column.id"
          :prop="column.id"
          :label="column.label"
        />
        <el-table-column label="操作" width="100">
          <template #default="scope">
            <el-button data-test="grid-edit-row" link type="primary" @click="openEditor(scope.$index)">
              编辑
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-drawer v-model="editorVisible" :title="`编辑${widget.label}`" size="420px">
        <el-form :model="editingRow" label-width="100px">
          <el-form-item
            v-for="column in editableColumns"
            :key="column.id"
            :label="column.label"
          >
            <el-input
              v-if="column.type === 'text' || column.type === 'textarea'"
              v-model="editingRow[column.id]"
              :type="column.type === 'textarea' ? 'textarea' : 'text'"
              :data-test="`grid-field-${column.id}`"
            />
            <el-input-number
              v-else-if="column.type === 'number'"
              v-model="editingRow[column.id]"
              :min="column.validation?.min"
              :max="column.validation?.max"
              :data-test="`grid-field-${column.id}`"
            />
            <el-select
              v-else-if="column.type === 'select'"
              v-model="editingRow[column.id]"
              :data-test="`grid-field-${column.id}`"
            >
              <el-option
                v-for="option in column.options || []"
                :key="String(option.value)"
                :label="option.label"
                :value="option.value"
              />
            </el-select>
            <el-switch
              v-else-if="column.type === 'switch'"
              v-model="editingRow[column.id]"
              :data-test="`grid-field-${column.id}`"
            />
            <el-input v-else v-model="editingRow[column.id]" :data-test="`grid-field-${column.id}`" />
          </el-form-item>
        </el-form>
        <template #footer>
          <el-button @click="editorVisible = false">取消</el-button>
          <el-button data-test="grid-save-row" type="primary" @click="saveEditor">保存</el-button>
        </template>
      </el-drawer>
    </template>
    <el-empty v-else description="Unsupported widget type" />
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import type { GridColumn, GridWidget } from '../../types/grid-schema'

interface Props {
  widget: GridWidget
  modelValue: Record<string, unknown>
  errors?: Record<string, string[]>
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, unknown>]
}>()

const editorVisible = ref(false)
const editingIndex = ref(-1)
const editingRow = reactive<Record<string, unknown>>({})

const gridStyle = computed(() => ({
  gridColumn: `span ${props.widget.grid.span || 12}`
}))

const rows = computed(() => {
  const value = props.modelValue[props.widget.id]
  return Array.isArray(value) ? value : []
})

const editableColumns = computed(() => {
  return (props.widget.columns || []).filter(column => !column.readonly)
})

function openEditor(index: number) {
  editingIndex.value = index
  Object.keys(editingRow).forEach(key => delete editingRow[key])
  Object.assign(editingRow, rows.value[index] as Record<string, unknown>)
  editorVisible.value = true
}

function saveEditor() {
  if (editingIndex.value < 0) return

  const nextRows = rows.value.map((row, index) => {
    if (index !== editingIndex.value) return row
    return { ...(row as Record<string, unknown>), ...editingRow }
  })

  emit('update:modelValue', {
    ...props.modelValue,
    [props.widget.id]: nextRows
  })
  editorVisible.value = false
}
</script>

<style scoped>
.grid-table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.grid-table-header h3 {
  margin: 0 0 12px;
  font-size: 16px;
}
</style>
