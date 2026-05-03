<template>
  <div class="yang-renderer">
    <!-- 工具栏 - 仅顶层渲染显示 -->
    <div v-if="!rootSchema && !loading" class="toolbar">
      <div class="toolbar-left">
        <span class="toolbar-title">{{ schemaDescription }}</span>
      </div>
      <div class="toolbar-right">
        <el-button @click="loadData" :loading="loading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          <el-icon><Check /></el-icon>
          下发配置
        </el-button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>加载配置中...</span>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-state">
      <el-alert
        :title="error"
        type="error"
        :closable="false"
        show-icon
      />
    </div>

    <!-- Render Container/List -->
    <template v-else-if="schema">
      <!-- Container 类型：渲染分组面板 -->
      <template v-if="schema.type === 'container'">
        <YangPanel v-for="child in editableChildren" :key="child.path" :node="child">
          <!-- list 类型：渲染表格 -->
          <YangTable
            v-if="child.type === 'list'"
            :node="child"
            v-model="formData[child.name]"
          />

          <!-- container 嵌套：递归渲染 -->
          <template v-else-if="child.type === 'container'">
            <YangRenderer
              :yang-path="child.path"
              :device-ip="deviceIp"
              :root-schema="child"
              :root-data="formData[child.name]"
            />
          </template>

          <!-- 简单字段：渲染表单 -->
          <el-form v-else label-width="180px">
            <YangField
              v-for="field in editableChildFields"
              :key="field.path"
              :node="field"
              v-model="formData[field.name]"
            />
          </el-form>
        </YangPanel>
      </template>

      <!-- 直接是 list 类型 -->
      <YangTable
        v-else-if="schema.type === 'list'"
        :node="schema"
        v-model="formData"
      />
    </template>

    <!-- No Schema -->
    <div v-else class="empty-state">
      <el-empty description="暂无 Schema 定义" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { Loading, Refresh, Check } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import YangPanel from './YangPanel.vue'
import YangTable from './YangTable.vue'
import YangField from './YangField.vue'
import { getSchemaByPath, getDefaultValue, convertKeysToKebab } from '../../types/yang-schema'
import { getConfig, setConfig } from '../../api'
import type { YangNode, FormData, FieldValue } from '../../types/yang-schema'

interface Props {
  /** YANG 路径，如 '/vlans' */
  yangPath: string
  /** 设备 IP */
  deviceIp: string
  /** 根 Schema (可选，不传则通过路径查找) */
  rootSchema?: YangNode
  /** 根数据 (可选，用于嵌套渲染) */
  rootData?: FormData
}

const props = defineProps<Props>()

const loading = ref(false)
const submitting = ref(false)
const error = ref('')
const formData = ref<FormData>({})

const schema = computed(() =>
  props.rootSchema || getSchemaByPath(props.yangPath)
)

const schemaDescription = computed(() =>
  schema.value?.description || props.yangPath
)

// 可编辑的子节点
const editableChildren = computed(() =>
  schema.value?.children?.filter(c => c.config !== false) || []
)

// 简单字段（非 container、非 list）
const editableChildFields = computed(() =>
  editableChildren.value.filter(c =>
    c.type !== 'container' && c.type !== 'list'
  )
)

const loadData = async () => {
  // 如果有传入 rootData，说明是嵌套渲染，直接使用
  if (props.rootData) {
    formData.value = props.rootData
    return
  }

  loading.value = true
  error.value = ''

  try {
    const res = await getConfig(props.deviceIp, props.yangPath)
    if (res.data.success && res.data.data) {
      // 后端返回 { vlans: [...], fromCache, lastSync } 格式
      // 使用除了元数据字段之外的配置数据
      const { fromCache, lastSync, ...configData } = res.data.data
      // 统一转换为 kebab-case 格式，确保与 schema 一致
      formData.value = convertKeysToKebab(configData)
    } else {
      // 初始化默认值
      initDefaultValues()
    }
  } catch (err: any) {
    error.value = err.message || '加载配置失败'
    initDefaultValues()
  } finally {
    loading.value = false
  }
}

const handleSubmit = async () => {
  submitting.value = true
  try {
    const res = await setConfig(props.deviceIp, props.yangPath, formData.value)

    if (res.data.success) {
      ElMessage.success('配置下发成功')
      await loadData()
    } else {
      ElMessage.error(res.data.message || '下发失败')
    }
  } catch (err: any) {
    ElMessage.error(err.message || '下发失败')
  } finally {
    submitting.value = false
  }
}

const initDefaultValues = () => {
  if (!schema.value) return

  const data: FormData = {}
  schema.value.children?.forEach(child => {
    if (child.config !== false) {
      data[child.name] = getDefaultValue(child)
    }
  })
  formData.value = data
}

// 暴露方法给父组件
defineExpose({
  loadData,
  formData
})

onMounted(() => {
  loadData()
})

watch(() => props.yangPath, () => {
  loadData()
})
</script>

<style lang="scss" scoped>
.yang-renderer {
  width: 100%;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--spacing-xl);

  .toolbar-title {
    font-size: var(--font-size-lg);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
  }

  .toolbar-right {
    display: flex;
    gap: var(--spacing-md);
  }
}

.loading-state, .error-state {
  padding: 60px 20px;
  text-align: center;
}

.loading-state {
  color: var(--text-secondary);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--spacing-md);

  .el-icon {
    font-size: 32px;
    color: var(--color-primary);
  }
}

.empty-state {
  padding: 60px 20px;
}
</style>
