<template>
  <el-switch
    :model-value="modelValue"
    @update:model-value="emit('update:modelValue', $event)"
    :disabled="disabled"
    size="default"
  />
</template>

<script setup lang="ts">
interface Props {
  modelValue: boolean
  disabled?: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()
</script>
