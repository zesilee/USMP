<template>
  <el-form
    ref="formRef"
    :model="formData"
    :rules="rules"
    label-width="120px"
    class="dynamic-form"
  >
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
  </el-form>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { FormInstance, FormRules } from 'element-plus'
import FieldRenderer from './FieldRenderer.vue'
import type { Field } from '../../api/crd'

const props = defineProps<{
  fields: Field[]
  modelValue: Record<string, any>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: Record<string, any>]
}>()

const formRef = ref<FormInstance>()
const formData = ref<Record<string, any>>({ ...props.modelValue })

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
</style>
