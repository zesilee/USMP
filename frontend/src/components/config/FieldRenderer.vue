<template>
  <div class="field-renderer">
    <!-- String -->
    <el-input
      v-if="field.type === 'string'"
      :model-value="modelValue"
      @update:model-value="$emit('update:modelValue', $event)"
      :placeholder="field.placeholder"
      :disabled="field.readonly"
    />

    <!-- Number -->
    <el-input-number
      v-else-if="field.type === 'number'"
      :model-value="modelValue"
      @update:model-value="$emit('update:modelValue', $event)"
      :disabled="field.readonly"
      :min="field.minimum"
      :max="field.maximum"
      controls-position="right"
      style="width: 100%"
    />

    <!-- Boolean -->
    <el-switch
      v-else-if="field.type === 'boolean'"
      :model-value="modelValue"
      @update:model-value="$emit('update:modelValue', $event)"
      :disabled="field.readonly"
    />

    <!-- Enum -->
    <el-select
      v-else-if="field.type === 'enum'"
      :model-value="modelValue"
      @update:model-value="$emit('update:modelValue', $event)"
      :placeholder="field.placeholder"
      :disabled="field.readonly"
      clearable
      style="width: 100%"
    >
      <el-option
        v-for="option in field.options"
        :key="String(option.value)"
        :label="option.label"
        :value="option.value"
      />
    </el-select>

    <!-- Group (single nested object) -->
    <div v-else-if="field.type === 'group'" class="field-group">
      <div class="group-fields">
        <div v-for="subField in childFields" :key="subField.path" class="sub-field">
          <label class="field-label">{{ subField.label }}</label>
          <FieldRenderer
            :field="subField"
            :model-value="(modelValue || {})[keyOf(subField)]"
            @update:model-value="updateSubField(keyOf(subField), $event)"
          />
        </div>
      </div>
    </div>

    <!-- List (repeatable rows of a nested sub-form) -->
    <div v-else-if="field.type === 'list'" class="field-list">
      <div v-for="(row, idx) in rows" :key="idx" class="list-row">
        <div class="list-row-fields">
          <div v-for="subField in childFields" :key="subField.path" class="sub-field">
            <label class="field-label">{{ subField.label }}</label>
            <FieldRenderer
              :field="subField"
              :model-value="(row || {})[keyOf(subField)]"
              @update:model-value="updateRow(idx, keyOf(subField), $event)"
            />
          </div>
        </div>
        <el-button
          type="danger"
          size="small"
          link
          :icon="Delete"
          @click="removeRow(idx)"
        >删除</el-button>
      </div>
      <el-button type="primary" size="small" plain :icon="Plus" @click="addRow">
        添加{{ field.label }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Plus, Delete } from '@element-plus/icons-vue'
import type { Field } from '../../utils/crdSchemaParser'

const props = defineProps<{
  field: Field
  modelValue: any
}>()

const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

// 数据以 YANG 叶子名（path 末段）为键，对齐后端转换（非 full path）。
function keyOf(f: Field): string {
  const seg = f.path.split('/').filter(Boolean).pop()
  return seg || f.path
}

// 子字段：group/list 的直接子项。read-only 的 oper 叶子（如 member-port/state）不参与配置。
const childFields = computed<Field[]>(() =>
  (props.field.fields || []).filter(f => !f.readonly)
)

const rows = computed<Record<string, any>[]>(() =>
  Array.isArray(props.modelValue) ? props.modelValue : []
)

function updateSubField(key: string, value: any) {
  const current = (props.modelValue as Record<string, any>) || {}
  emit('update:modelValue', { ...current, [key]: value })
}

function updateRow(idx: number, key: string, value: any) {
  const next = rows.value.map((r, i) => (i === idx ? { ...r, [key]: value } : r))
  emit('update:modelValue', next)
}

function addRow() {
  emit('update:modelValue', [...rows.value, {}])
}

function removeRow(idx: number) {
  emit('update:modelValue', rows.value.filter((_, i) => i !== idx))
}
</script>

<style scoped>
.field-renderer {
  width: 100%;
}

.field-group {
  width: 100%;
  padding: 12px;
  background-color: #f9fafb;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
}

.group-fields {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.sub-field {
  display: flex;
  align-items: center;
  gap: 12px;
}

.field-label {
  min-width: 96px;
  font-size: 14px;
  color: #606266;
  margin: 0;
}

.field-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 100%;
}

.list-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  background-color: #f9fafb;
  border-radius: 8px;
  border: 1px solid #e5e7eb;
}

.list-row-fields {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
</style>
