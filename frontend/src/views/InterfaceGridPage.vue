<template>
  <GridRenderer
    v-if="schema"
    v-model="values"
    :schema="schema"
    :loading="loading"
    :submitting="submitting"
    :errors="fieldErrors"
    :page-error="pageError"
    @refresh="loadSchema"
    @submit="submit"
  />
  <div v-else class="interface-grid-loading">
    <el-icon class="is-loading"><Loading /></el-icon>
    <span>加载接口配置中...</span>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Loading } from '@element-plus/icons-vue'
import GridRenderer from '../components/grid/GridRenderer.vue'
import { applyInterfaceGridConfig, getInterfaceGridSchema } from '../api'
import type { GridSchema } from '../types/grid-schema'

const props = withDefaults(defineProps<{ deviceIp?: string }>(), {
  deviceIp: '192.168.1.1'
})

const schema = ref<GridSchema | null>(null)
const values = ref<Record<string, unknown>>({})
const fieldErrors = ref<Record<string, string[]>>({})
const pageError = ref('')
const loading = ref(false)
const submitting = ref(false)

async function loadSchema() {
  loading.value = true
  pageError.value = ''
  try {
    const res = await getInterfaceGridSchema(props.deviceIp)
    if (res.data.success && res.data.data) {
      schema.value = res.data.data
      values.value = res.data.data.values || {}
      fieldErrors.value = {}
    } else {
      pageError.value = res.data.message || '加载接口 UI schema 失败'
    }
  } catch (error: any) {
    pageError.value = error.message || '加载接口 UI schema 失败'
  } finally {
    loading.value = false
  }
}

async function submit() {
  if (!schema.value) return
  submitting.value = true
  fieldErrors.value = {}
  pageError.value = ''
  try {
    const res = await applyInterfaceGridConfig(props.deviceIp, {
      schemaVersion: schema.value.schemaVersion,
      values: values.value
    })
    if (res.data.success) {
      ElMessage.success('配置下发成功')
      if (res.data.data?.values) {
        values.value = res.data.data.values
      }
    } else {
      const data = res.data as any
      fieldErrors.value = data.fieldErrors || {}
      pageError.value = res.data.message || '配置下发失败'
    }
  } catch (error: any) {
    pageError.value = error.message || '配置下发失败'
  } finally {
    submitting.value = false
  }
}

onMounted(loadSchema)
</script>

<style scoped>
.interface-grid-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  min-height: 240px;
  color: #909399;
}
</style>
