<template>
  <el-form
    ref="formRef"
    :model="formData"
    :rules="rules"
    label-width="120px"
    class="dynamic-form"
  >
    <!-- Grouped fields with el-collapse -->
    <template v-if="groupedFields.size > 1">
      <el-collapse v-model="activeGroups">
        <el-collapse-item
          v-for="[groupName, groupFields] in groupedFields"
          :key="groupName"
          :name="groupName"
        >
          <template #title>{{ groupName }}</template>
          <div class="group-fields">
            <el-form-item
              v-for="field in groupFields"
              :key="field.path"
              :label="field.label"
              :prop="field.path"
              :required="field.required"
            >
              <FieldRenderer
                :field="field"
                :model-value="formData[field.path]"
                @update:model-value="updateField(field.path, $event)"
              />
            </el-form-item>
          </div>
        </el-collapse-item>
      </el-collapse>
    </template>

    <!-- Ungrouped fields (single group or no grouping) -->
    <template v-else>
      <el-form-item
        v-for="field in fields"
        :key="field.path"
        :label="field.label"
        :prop="field.path"
        :required="field.required"
      >
        <FieldRenderer
          :field="field"
          :model-value="formData[field.path]"
          @update:model-value="updateField(field.path, $event)"
        />
      </el-form-item>
    </template>
  </el-form>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import FieldRenderer from './FieldRenderer.vue'
import type { Field } from '../../utils/crdSchemaParser'

const props = defineProps<{
  fields: Field[]
  modelValue: Record<string, any>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, any>]
}>()

const formRef = ref<FormInstance>()
const formData = ref<Record<string, any>>({ ...props.modelValue })

// Group fields by group
const groupedFields = computed(() => {
  const groups = new Map<string, Field[]>()
  for (const field of props.fields) {
    const group = field.group || '其他'
    if (!groups.has(group)) {
      groups.set(group, [])
    }
    groups.get(group)!.push(field)
  }
  return groups
})

// Default: all groups active
const activeGroups = ref<string[]>([])
onMounted(() => {
  activeGroups.value = Array.from(groupedFields.value.keys())
})

const rules = computed<FormRules>(() => {
  const result: FormRules = {}
  props.fields.forEach(field => {
    const fieldRules: any[] = []
    if (field.required) {
      fieldRules.push({ required: true, message: `请输入${field.label}`, trigger: 'blur' })
    }
    if (field.pattern) {
      fieldRules.push({ pattern: new RegExp(field.pattern), message: `${field.label}格式不正确`, trigger: 'blur' })
    }
    if (field.minimum !== undefined) {
      fieldRules.push({ min: field.minimum, message: `${field.label}最小值为${field.minimum}`, trigger: 'blur' })
    }
    if (field.maximum !== undefined) {
      fieldRules.push({ max: field.maximum, message: `${field.label}最大值为${field.maximum}`, trigger: 'blur' })
    }
    if (fieldRules.length > 0) {
      result[field.path] = fieldRules
    }
  })
  return result
})

function updateField(path: string, value: any) {
  formData.value[path] = value
  emit('update:modelValue', { ...formData.value })
}

watch(() => props.modelValue, (newVal) => {
  formData.value = { ...newVal }
}, { deep: true })

function validate() {
  return formRef.value?.validate()
}

function resetFields() {
  formRef.value?.resetFields()
}

function getFormData() {
  return { ...formData.value }
}

defineExpose({
  validate,
  resetFields,
  getFormData
})
</script>

<style scoped>
.dynamic-form {
  padding: 16px 0;
}

.group-fields {
  padding: 8px 0;
}

:deep(.el-collapse-item__header) {
  font-weight: 600;
}
</style>
