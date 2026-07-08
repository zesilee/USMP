<template>
  <div class="module-console">
    <div class="page-header">
      <!-- 面包屑：配置 / 厂商 / 模块 / 激活 Tab（FE-10） -->
      <el-breadcrumb separator=">">
        <el-breadcrumb-item>配置</el-breadcrumb-item>
        <el-breadcrumb-item v-if="vendor">{{ vendor }}</el-breadcrumb-item>
        <el-breadcrumb-item>{{ title }}</el-breadcrumb-item>
        <el-breadcrumb-item v-if="activeTabLabel">{{ activeTabLabel }}</el-breadcrumb-item>
      </el-breadcrumb>
      <div class="header-actions">
        <el-select v-model="selectedDevice" placeholder="选择设备" style="width: 220px">
          <el-option v-for="d in store.devices" :key="d.id" :label="d.ip" :value="d.ip" />
        </el-select>
      </div>
    </div>

    <el-alert v-if="schemaError" :title="schemaError" type="error" :closable="false" show-icon />

    <!-- 一级 Tab：模块根顶层子节点派生（list→列表页、group/choice→表单页，FE-10）。
         Tab 组件常驻（不销毁），切换保留各 Tab 表单/搜索状态。 -->
    <el-tabs v-if="tabs.length" v-model="activeTab" class="console-tabs">
      <el-tab-pane v-for="tab in tabs" :key="tab.name" :label="tab.label" :name="tab.name">
        <ModuleListTab v-if="tab.kind === 'list'" :tab="tab" :root-name="rootName" :device="selectedDevice" />
        <ModuleFormTab v-else :tab="tab" :root-name="rootName" :device="selectedDevice" />
      </el-tab-pane>
    </el-tabs>
    <el-empty v-else-if="!schemaError" description="模块 schema 加载中…" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { getYangSchema } from '../api'
import { useDeviceStore } from '../stores/device'
import type { Field } from '../utils/crdSchemaParser'
import { deriveTabs, type ConsoleTab } from '../utils/moduleConsole'
import ModuleListTab from '../components/config/ModuleListTab.vue'
import ModuleFormTab from '../components/config/ModuleFormTab.vue'

const route = useRoute()
const store = useDeviceStore()

const moduleName = computed(() => String(route.params.module || ''))
const selectedDevice = ref('')
const schemaError = ref('')
const title = ref('')
const vendor = ref('')
const rootName = ref('')
const schemaFields = ref<Field[]>([])

const tabs = computed<ConsoleTab[]>(() => deriveTabs(schemaFields.value))
const activeTab = ref('')
const activeTabLabel = computed(() => tabs.value.find((t) => t.name === activeTab.value)?.label || '')

async function loadSchema() {
  schemaError.value = ''
  schemaFields.value = []
  try {
    const res = await getYangSchema(moduleName.value, 'nested')
    const data = res.data?.data
    schemaFields.value = data?.fields ?? []
    title.value = data?.title || moduleName.value
    vendor.value = data?.vendor || ''
    // 运行时配置路径的根段 = 模块根容器名（schema title 即 root.Name()）。
    rootName.value = data?.title || moduleName.value
    activeTab.value = tabs.value[0]?.name || ''
  } catch (e: any) {
    // schema 拉取失败降级：页面不崩，明确报错（R08/§9）。
    schemaError.value = e?.response?.data?.message || e?.message || 'schema 加载失败'
  }
}

watch(moduleName, loadSchema)

onMounted(async () => {
  await Promise.allSettled([store.fetchDevices(), loadSchema()])
})
</script>

<style scoped>
.module-console {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.console-tabs {
  background: #fff;
  border-radius: 8px;
  padding: 4px 16px 16px;
}
</style>
