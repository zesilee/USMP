<template>
  <el-form-item
    :label="label"
    :prop="node.name"
    :error="errorMessage"
    class="yang-field-item"
  >
    <!-- 必填标记 -->
    <template #label v-if="node.mandatory">
      <span class="required-mark">*</span>
      {{ label }}
    </template>

    <YangSwitch
      v-if="node.type === 'boolean'"
      v-model="fieldValue"
      :disabled="!node.config"
    />

    <YangSelect
      v-else-if="node.type === 'enum'"
      v-model="fieldValue"
      :node="node"
      :disabled="!node.config"
    />

    <YangInput
      v-else-if="['string', 'int', 'uint'].includes(node.type)"
      v-model="fieldValue"
      :node="node"
      :disabled="!node.config"
      @change="handleValueChange"
    />

    <YangListEditor
      v-else-if="node.type === 'list'"
      v-model="fieldValue"
      :node="node"
      :disabled="!node.config"
    />

    <div v-else class="unsupported-type">
      <el-tag type="info" size="small">不支持的类型: {{ node.type }}</el-tag>
    </div>
  </el-form-item>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import YangSwitch from './YangSwitch.vue'
import YangSelect from './YangSelect.vue'
import YangInput from './YangInput.vue'
import YangListEditor from './YangListEditor.vue'
import { validateField, type YangNode, type FieldValue, type ValidationResult } from '../../types/yang-schema'

interface Props {
  node: YangNode
  modelValue: FieldValue
  errorMessage?: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: FieldValue]
  'validate': [result: ValidationResult]
}>()

const localError = ref('')

const label = computed(() => props.node.description || props.node.name)

const fieldValue = computed({
  get: () => props.modelValue,
  set: (val) => {
    emit('update:modelValue', val)
    validate(val)
  }
})

const validate = (value: FieldValue) => {
  const result = validateField(props.node, value)
  localError.value = result.errors[0]?.message || ''
  emit('validate', result)
}

const handleValueChange = () => {
  validate(props.modelValue)
}

// 初始化时验证一次
watch(
  () => props.modelValue,
  (newVal) => {
    if (newVal !== undefined) {
      validate(newVal)
    }
  },
  { immediate: true }
)
</script>

<style scoped>
.unsupported-type {
  display: flex;
  align-items: center;
}

:deep(.el-form-item__label) {
  white-space: nowrap;
}

.required-mark {
  color: var(--el-color-danger);
  margin-right: 4px;
}

.yang-field-item {
  margin-bottom: 16px;
}
</style>
