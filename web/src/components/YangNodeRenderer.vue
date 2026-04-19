<template>
  <div class="yang-node">
    <el-form-item :label="label">
      <!-- boolean -> switch -->
      <template v-if="type === 'boolean'">
        <el-switch v-model="value" />
      </template>

      <!-- enum -> select -->
      <template v-else-if="type === 'enum'">
        <el-select v-model="value" :placeholder="label">
          <el-option
            v-for="opt in options"
            :key="opt.value"
            :label="opt.label"
            :value="opt.value"
          />
        </el-select>
      </template>

      <!-- integer/uint -> input number -->
      <template v-else-if="['int', 'uint', 'uint16', 'uint32'].includes(type)">
        <el-input-number v-model="value" :min="0" />
      </template>

      <!-- string -> input text -->
      <template v-else>
        <el-input v-model="value" :placeholder="label" />
      </template>
    </el-form-item>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  label: string
  type: string
  modelValue: any
  options?: Array<{ label: string; value: string }>
}

const props = withDefaults(defineProps<Props>(), {
  options: () => [],
})

const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

const value = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val),
})
</script>

<style scoped>
.yang-node {
  width: 100%;
}
</style>
