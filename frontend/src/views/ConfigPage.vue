<template>
  <div class="config-page">
    <div class="page-header">
      <div class="title-section">
        <h2 class="page-title">{{ moduleTitle }}</h2>
        <StatusBadge v-if="configHook" :phase="configHook.currentPhase.value" />
      </div>
      <el-select
        v-model="selectedDevice"
        placeholder="选择设备"
        style="width: 200px"
        @change="handleDeviceChange"
      >
        <el-option v-for="device in devices" :key="device.id" :label="device.name" :value="device.id" />
      </el-select>
    </div>

    <el-alert
      v-if="configHook?.error?.value"
      :title="configHook.error.value"
      type="error"
      :closable="false"
      style="margin-bottom: 20px"
    />

    <div v-if="configHook?.isLoading.value" class="loading-container">
      <el-icon class="is-loading" size="40">
        <Loading />
      </el-icon>
      <p>加载中...</p>
    </div>

    <template v-else>
      <DynamicTable
        v-if="configHook?.schema.value"
        :columns="configHook.schema.value.listFields"
        :data="configList"
        @add="handleAdd"
        @edit="handleEdit"
        @delete="handleDelete"
      />
    </template>

    <DetailDrawer
      v-model="drawerVisible"
      :title="isEditing ? '编辑配置' : '新增配置'"
      :loading="submitLoading"
      @confirm="handleSubmit"
    >
      <DynamicForm
        ref="formRef"
        v-if="configHook?.schema.value"
        :fields="configHook.schema.value.fields"
        v-model="formData"
      />
    </DetailDrawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useDeviceStore } from '../stores/device'
import { useDeviceConfig } from '../composables/useDeviceConfig'
import StatusBadge from '../components/common/StatusBadge.vue'
import DetailDrawer from '../components/common/DetailDrawer.vue'
import DynamicTable from '../components/config/DynamicTable.vue'
import DynamicForm from '../components/config/DynamicForm.vue'
import { Loading } from '@element-plus/icons-vue'
import type { FormInstance } from 'element-plus'

const props = defineProps<{
  module?: string
}>()

const deviceStore = useDeviceStore()
const devices = computed(() => deviceStore.devices)
const selectedDevice = ref('')
const configHook = ref<ReturnType<typeof useDeviceConfig> | null>(null)

const drawerVisible = ref(false)
const isEditing = ref(false)
const submitLoading = ref(false)
const formRef = ref<FormInstance>()
const formData = ref<Record<string, any>>({})
const editingIndex = ref(-1)

const moduleTitle = computed(() => {
  const titles: Record<string, string> = {
    interface: '接口配置',
    vlan: 'VLAN 配置',
    route: '路由配置',
    native: '原生配置'
  }
  return titles[props.module || ''] || '配置管理'
})

const configList = computed(() => {
  const spec = configHook.value?.configCR.value?.spec
  if (!spec) return []
  return Array.isArray(spec) ? spec : [spec]
})

function initConfig(deviceId: string) {
  if (deviceId && props.module) {
    configHook.value = useDeviceConfig(deviceId, props.module)
  }
}

function handleDeviceChange() {
  initConfig(selectedDevice.value)
}

function handleAdd() {
  isEditing.value = false
  editingIndex.value = -1
  formData.value = {}
  drawerVisible.value = true
}

function handleEdit(row: Record<string, any>, index: number) {
  isEditing.value = true
  editingIndex.value = index
  formData.value = { ...row }
  drawerVisible.value = true
}

async function handleDelete(row: Record<string, any>, index: number) {
  if (!configHook.value) return
  try {
    await configHook.value.remove()
  } catch (e) {
    console.error('Delete failed:', e)
  }
}

async function handleSubmit() {
  if (!formRef.value || !configHook.value) return

  try {
    await formRef.value.validate()
    submitLoading.value = true

    await configHook.value.save(formData.value)

    drawerVisible.value = false
    formData.value = {}
  } catch (e) {
    console.error('Submit failed:', e)
  } finally {
    submitLoading.value = false
  }
}

watch(() => props.module, () => {
  if (selectedDevice.value) {
    initConfig(selectedDevice.value)
  }
})
</script>

<style scoped>
.config-page {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.title-section {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  gap: 16px;
  color: #909399;
}

.loading-container p {
  margin: 0;
}
</style>
