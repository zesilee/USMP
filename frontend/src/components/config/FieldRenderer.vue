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
      style="width: 100%"
    >
      <el-option
        v-for="option in field.options"
        :key="option.value"
        :label="option.label"
        :value="option.value"
      />
    </el-select>

    <!-- Group (nested fields) -->
    <div v-else-if="field.type === 'group'" class="field-group">
      <div class="group-label">{{ field.label }}</div>
      <div class="group-fields">
        <div v-for="subField in field.fields" :key="subField.path" class="sub-field">
          <label class="field-label">{{ subField.label }}</label>
          <FieldRenderer
            :field="subField"
            :model-value="modelValue?.[subField.path]"
            @update:model-value="updateSubField(subField.path, $event)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Field } from '../../utils/crdSchemaParser'

const props = defineProps<{
  field: Field
  modelValue: any
}>()

const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

function updateSubField(path: string, value: any) {
  const current = (props.modelValue as Record<string, any>) || {}
  emit('update:modelValue', { ...current, [path]: value })
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

.group-label {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 12px;
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
  min-width: 80px;
  font-size: 14px;
  color: #606266;
  margin: 0;
}
</style>
