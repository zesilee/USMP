<template>
  <el-select
    v-model="localValue"
    :disabled="disabled"
    :placeholder="placeholder"
    class="w-full"
    clearable
    @change="handleChange"
  >
    <el-option
      v-for="opt in node.enumOptions"
      :key="opt.value"
      :label="opt.name"
      :value="opt.value"
    />
  </el-select>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { YangNode } from '../../types/yang-schema'

interface Props {
  modelValue: string | number | undefined
  node: YangNode
  disabled?: boolean
  placeholder?: string
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: '请选择'
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number | undefined]
}>()

const localValue = ref(props.modelValue)

watch(() => props.modelValue, (newVal) => {
  localValue.value = newVal
})

const handleChange = () => {
  emit('update:modelValue', localValue.value)
}
</script>
