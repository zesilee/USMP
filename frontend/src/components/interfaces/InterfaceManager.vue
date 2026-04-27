<template>
  <div class="interface-manager">
    <!-- 操作栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <h2 class="page-title">接口配置管理</h2>
        <el-tag size="small" type="info">{{ deviceIp }}</el-tag>
      </div>
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
    <YangRenderer
      ref="rendererRef"
      :yang-path="yangPath"
      :device-ip="deviceIp"
    />
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

const yangPath = '/interfaces'
const loading = ref(false)
const submitting = ref(false)
const rendererRef = ref()

const handleRefresh = () => {
  rendererRef.value?.loadData?.()
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    // 获取当前表单数据
    const formData = rendererRef.value?.formData || {}

    // 调用后端下发配置
    const res = await setConfig(props.deviceIp, yangPath, formData)

    if (res.data.success) {
      ElMessage.success('配置下发成功')
      // 刷新数据
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
@import '../../styles/variables.scss';

.interface-manager {
  width: 100%;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 0;
  margin-bottom: 16px;
  border-bottom: 1px solid $border-color;

  .toolbar-left {
    display: flex;
    align-items: center;
    gap: 12px;

    .page-title {
      font-size: 18px;
      font-weight: 600;
      color: $text-primary;
      margin: 0;
    }
  }

  .toolbar-right {
    display: flex;
    gap: 8px;
  }
}
</style>
