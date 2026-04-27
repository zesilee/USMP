<template>
  <div class="yang-input">
    <el-input-number
      v-if="isNumeric"
      v-model="localValue"
      :min="min"
      :max="max"
      :disabled="disabled"
      class="w-full"
      @change="handleChange"
    />
    <el-input
      v-else
      v-model="localValue"
      :type="inputType"
      :disabled="disabled"
      :placeholder="placeholder"
      clearable
      @change="handleChange"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import type { YangNode } from '../../types/yang-schema'

interface Props {
  modelValue: string | number | undefined
  node: YangNode
  disabled?: boolean
  placeholder?: string
}

const props = withDefaults(defineProps<Props>(), {
  placeholder: ''
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number | undefined]
}>()

const localValue = ref(props.modelValue)

watch(() => props.modelValue, (newVal) => {
  localValue.value = newVal
})

const isNumeric = computed(() =>
  props.node.type === 'int' || props.node.type === 'uint'
)

const inputType = computed(() =>
  props.node.type === 'uint' || props.node.type === 'int' ? 'number' : 'text'
)

const min = computed(() => props.node.range?.min ?? -Infinity)
const max = computed(() => props.node.range?.max ?? Infinity)

const handleChange = () => {
  emit('update:modelValue', localValue.value)
}
</script>

<style scoped>
.yang-input {
  width: 100%;
}
</style>
