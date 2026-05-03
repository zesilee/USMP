<template>
  <el-drawer
    :model-value="modelValue"
    :title="title"
    direction="rtl"
    size="560px"
    @close="handleClose"
    @update:model-value="handleUpdate"
  >
    <slot />
    <template #footer v-if="showFooter">
      <div class="drawer-footer">
        <el-button @click="handleCancel">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          {{ submitText }}
        </el-button>
      </div>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
interface Props {
  modelValue: boolean
  title: string
  showFooter?: boolean
  submitText?: string
  submitting?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  showFooter: false,
  submitText: '提交',
  submitting: false
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'close': []
  'cancel': []
  'submit': []
}>()

function handleClose() {
  emit('close')
}

function handleUpdate(value: boolean) {
  emit('update:modelValue', value)
}

function handleCancel() {
  emit('cancel')
  emit('update:modelValue', false)
}

function handleSubmit() {
  emit('submit')
}
</script>

<style scoped>
.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid #e5e7eb;
}
</style>
