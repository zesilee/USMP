<template>
  <div class="settings">
    <div class="page-header">
      <h2>系统设置</h2>
    </div>

    <el-card class="setting-card">
      <template #header>
        <div class="card-header">
          <span>全局设置</span>
        </div>
      </template>

      <el-form :model="globalSettings" label-width="120px">
        <el-form-item label="缓存有效期">
          <el-select v-model="globalSettings.cacheTTL" style="width: 200px">
            <el-option label="30秒" :value="30" />
            <el-option label="60秒" :value="60" />
            <el-option label="120秒" :value="120" />
          </el-select>
        </el-form-item>

        <el-form-item label="同步间隔">
          <el-select v-model="globalSettings.syncInterval" style="width: 200px">
            <el-option label="30秒" :value="30" />
            <el-option label="60秒" :value="60" />
            <el-option label="120秒" :value="120" />
          </el-select>
        </el-form-item>

        <el-form-item label="请求超时">
          <el-input-number
            v-model="globalSettings.requestTimeout"
            :min="5"
            :max="120"
            :step="5"
            style="width: 200px"
          />
          <span class="unit">秒</span>
        </el-form-item>

        <el-form-item label="重试次数">
          <el-input-number
            v-model="globalSettings.retryCount"
            :min="0"
            :max="5"
            :step="1"
            style="width: 200px"
          />
          <span class="unit">次</span>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="setting-card">
      <template #header>
        <div class="card-header">
          <span>主题设置</span>
        </div>
      </template>

      <el-form :model="themeSettings" label-width="120px">
        <el-form-item label="配色方案">
          <el-radio-group v-model="themeSettings.theme">
            <el-radio-button value="light">浅色模式</el-radio-button>
            <el-radio-button value="dark">深色模式</el-radio-button>
            <el-radio-button value="system">跟随系统</el-radio-button>
          </el-radio-group>
        </el-form-item>
      </el-form>
    </el-card>

    <div class="action-bar">
      <el-button @click="handleReset">重置</el-button>
      <el-button type="primary" @click="handleSave">保存设置</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { ElMessage } from 'element-plus'

const defaultGlobalSettings = {
  cacheTTL: 60,
  syncInterval: 60,
  requestTimeout: 30,
  retryCount: 3
}

const defaultThemeSettings = {
  theme: 'light'
}

const globalSettings = reactive({ ...defaultGlobalSettings })
const themeSettings = reactive({ ...defaultThemeSettings })

function handleSave() {
  ElMessage.success('设置已保存')
}

function handleReset() {
  Object.assign(globalSettings, defaultGlobalSettings)
  Object.assign(themeSettings, defaultThemeSettings)
  ElMessage.info('已恢复默认设置')
}
</script>

<style scoped>
.settings {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;
  max-width: 800px;
}

.page-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.setting-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.08);
}

.card-header {
  font-weight: 600;
  color: #303133;
}

.unit {
  margin-left: 12px;
  color: #909399;
  font-size: 14px;
}

.action-bar {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 12px;
}
</style>
