<template>
  <div class="dynamic-form">
    <el-alert
      v-if="loading"
      title="加载中..."
      type="info"
      :closable="false"
      show-icon
    />

    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="false"
    />

    <el-form
      v-else-if="configData"
      :model="formData"
      label-width="120px"
      class="config-form"
    >
      <!-- Simple display of raw XML/config for now - proper parsing will come after backend has structured JSON -->
      <el-form-item label="配置数据">
        <el-input
          v-model="formDataJson"
          type="textarea"
          :rows="15"
          disabled
        />
      </el-form-item>

      <el-form-item>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          提交配置
        </el-button>
        <el-button @click="handleRefresh">
          刷新
        </el-button>
      </el-form-item>

      <div v-if="lastResult" class="result">
        <el-alert
          :title="lastResult.message"
          :type="lastResult.success ? 'success' : 'error'"
        />
      </div>
    </el-form>

    <el-empty v-else description="暂无配置数据，请点击刷新获取"></el-empty>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { getConfig, setConfig } from '../api'

interface Props {
  deviceIp: string
  yangPath: string
}

const props = withDefaults(defineProps<Props>(), {})

const loading = ref(true)
const error = ref('')
const configData = ref<any>(null)
const formData = ref<any>({})
const submitting = ref(false)
const lastResult = ref<{ success: boolean; message: string } | null>(null)

const formDataJson = computed(() => {
  return JSON.stringify(formData.value, null, 2)
})

const loadConfig = async (forceRefresh = false) => {
  loading.value = true
  error.value = ''
  lastResult.value = null
  try {
    const res = await getConfig(props.deviceIp, props.yangPath, forceRefresh)
    if (res.data.success) {
      configData.value = res.data.data?.data
      formData.value = res.data.data?.data
    } else {
      error.value = res.data.message
    }
  } catch (err: any) {
    error.value = err.message || '加载配置失败'
  } finally {
    loading.value = false
  }
}

const handleRefresh = () => {
  loadConfig(true)
}

const handleSubmit = async () => {
  submitting.value = true
  lastResult.value = null
  try {
    const res = await setConfig(props.deviceIp, props.yangPath, formData.value)
    lastResult.value = {
      success: res.data.success,
      message: res.data.message,
    }
    if (res.data.success) {
      // Reload after successful commit
      await loadConfig(true)
    }
  } catch (err: any) {
    lastResult.value = {
      success: false,
      message: err.message || '提交失败',
    }
  } finally {
    submitting.value = false
  }
}

watch(() => props.deviceIp + props.yangPath, () => {
  loadConfig()
})

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.dynamic-form {
  margin-top: 20px;
}

.config-form {
  margin-top: 20px;
}

.result {
  margin-top: 20px;
}
</style>
