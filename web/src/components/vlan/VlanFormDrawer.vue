<template>
  <el-drawer
    :model-value="modelValue"
    :title="mode === 'create' ? '新建 VLAN' : '编辑 VLAN'"
    size="560px"
    @update:model-value="handleVisibleChange"
  >
    <el-form
      ref="formRef"
      :model="localForm"
      :rules="formRules"
      label-width="100px"
      class="vlan-form"
    >
      <el-form-item label="VLAN ID" prop="id">
        <el-input-number
          v-model="localForm.id"
          :min="1"
          :max="4094"
          :disabled="mode === 'edit'"
          placeholder="1-4094"
          style="width: 100%"
        />
        <div class="form-tip">VLAN 标识，范围 1-4094，创建后不可修改</div>
      </el-form-item>

      <el-form-item label="名称" prop="name">
        <el-input
          v-model="localForm.name"
          placeholder="输入 VLAN 名称"
          maxlength="32"
          show-word-limit
        />
      </el-form-item>

      <el-form-item label="管理状态" prop="adminStatus">
        <el-radio-group v-model="localForm.adminStatus">
          <el-radio value="UP" border>启用</el-radio>
          <el-radio value="DOWN" border>禁用</el-radio>
        </el-radio-group>
        <div class="form-tip">禁用后该 VLAN 下的端口将无法通信</div>
      </el-form-item>

      <el-divider content-position="left">端口关联</el-divider>

      <el-form-item label="Tagged 端口">
        <PortSelector
          v-model="localForm.taggedPorts"
          :device-ip="deviceIp"
          placeholder="选择打标端口"
        />
        <div class="form-tip">打标端口可承载多个 VLAN 流量，通常用于交换机级联</div>
      </el-form-item>

      <el-form-item label="Untagged 端口">
        <PortSelector
          v-model="localForm.untaggedPorts"
          :device-ip="deviceIp"
          placeholder="选择非打标端口"
        />
        <div class="form-tip">非打标端口通常用于终端设备接入</div>
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="drawer-footer">
        <el-button @click="handleVisibleChange(false)">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ mode === 'create' ? '创建' : '保存' }}
        </el-button>
      </div>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import PortSelector from './PortSelector.vue'
import type { VlanFormData } from '../../types/vlan'

interface Props {
  modelValue: boolean
  mode: 'create' | 'edit'
  formData: VlanFormData | null
  deviceIp: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  'submit': [data: VlanFormData]
}>()

const formRef = ref<FormInstance>()
const submitting = ref(false)

const localForm = reactive<VlanFormData>({
  id: null,
  name: '',
  adminStatus: 'UP',
  taggedPorts: [],
  untaggedPorts: []
})

const formRules: FormRules = {
  id: [
    { required: true, message: '请输入 VLAN ID', trigger: 'blur' },
    { type: 'number', min: 1, max: 4094, message: 'VLAN ID 范围为 1-4094', trigger: 'blur' }
  ],
  name: [
    { required: true, message: '请输入 VLAN 名称', trigger: 'blur' },
    { min: 1, max: 32, message: '名称长度 1-32 字符', trigger: 'blur' }
  ]
}

watch(() => props.formData, (data) => {
  if (data) {
    Object.assign(localForm, data)
  }
}, { immediate: true })

watch(() => props.modelValue, (visible) => {
  if (!visible) {
    formRef.value?.resetFields()
  }
})

const handleVisibleChange = (value: boolean) => {
  emit('update:modelValue', value)
}

const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (valid) {
      submitting.value = true
      try {
        emit('submit', { ...localForm })
      } finally {
        submitting.value = false
      }
    } else {
      ElMessage.error('请检查表单填写')
    }
  })
}
</script>

<style lang="scss" scoped>
@import '../../styles/variables.scss';

.vlan-form {
  .form-tip {
    font-size: $font-size-xs;
    color: $text-tertiary;
    margin-top: $spacing-xs;
    line-height: 1.4;
  }
}

.drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: $spacing-md;
  padding-top: $spacing-md;
  border-top: 1px solid $border-color;
}

:deep(.el-divider) {
  --el-border-color: #{$border-color};

  .el-divider__text {
    background-color: $bg-card;
    color: $text-secondary;
    font-size: $font-size-sm;
    font-weight: $font-weight-medium;
  }
}

:deep(.el-radio) {
  --el-radio-text-color: #{$text-primary};
}

:deep(.el-radio-button__inner) {
  background-color: $bg-elevated;
  border-color: $border-color;
  color: $text-secondary;

  &:hover {
    color: $color-primary;
  }
}

:deep(.is-active .el-radio-button__inner) {
  background-color: $color-primary;
  border-color: $color-primary;
  color: white;
}
</style>
