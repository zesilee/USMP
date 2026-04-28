<template>
  <div class="vlan-manager">
    <!-- 操作栏 -->
    <div class="toolbar">
      <div class="toolbar-right">
        <el-button @click="handleRefresh" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          <el-icon><Check /></el-icon>
          下发配置
        </el-button>
      </div>
    </div>

    <!-- 动态渲染 YANG 表单 -->
    <div class="card-container">
      <div class="card-body">
        <YangRenderer
          ref="rendererRef"
          :yang-path="yangPath"
          :device-ip="deviceIp"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Refresh, Check } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import YangRenderer from '../yang/YangRenderer.vue'
import { setConfig } from '../../api'

interface Props {
  deviceIp: string
}

const props = defineProps<Props>()

const yangPath = '/vlans'
const loading = ref(false)
const submitting = ref(false)
const rendererRef = ref()

const handleRefresh = () => {
  rendererRef.value?.loadData?.()
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    const formData = rendererRef.value?.formData || {}
    const res = await setConfig(props.deviceIp, yangPath, formData)

    if (res.data.success) {
      ElMessage.success('配置下发成功')
      await handleRefresh()
    } else {
      ElMessage.error(res.data.message || '下发失败')
    }
  } catch (err: any) {
    ElMessage.error(err.message || '下发失败')
  } finally {
    submitting.value = false
  }
}
</script>

<style lang="scss" scoped>
.vlan-manager {
  width: 100%;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  margin-bottom: var(--spacing-xl);

  .toolbar-right {
    display: flex;
    gap: var(--spacing-md);
  }
}

.card-body {
  padding: var(--spacing-xl);
}
</style>
