<template>
  <div class="module-console">
    <div class="page-header">
      <!-- 面包屑：配置 / 厂商 / 模块 / 激活 Tab（FE-10） -->
      <el-breadcrumb separator=">">
        <el-breadcrumb-item>{{ t('console.breadcrumbConfig') }}</el-breadcrumb-item>
        <el-breadcrumb-item v-if="vendor">{{ vendor }}</el-breadcrumb-item>
        <el-breadcrumb-item>{{ title }}</el-breadcrumb-item>
        <el-breadcrumb-item v-if="activeTabLabel">{{ activeTabLabel }}</el-breadcrumb-item>
      </el-breadcrumb>
      <div class="header-actions">
        <!-- 软归属徽标（FE-18）：本模块在选中设备上被业务意图认领时提示（不拦截）。 -->
        <el-tooltip
          v-if="ownershipIntents.length"
          :content="t('console.ownedTooltip', { intents: ownershipIntents.join('、') })"
        >
          <el-tag type="warning" size="small" data-test="ownership-badge">
            {{ t('console.ownedBadge', { n: ownershipIntents.length }) }}
          </el-tag>
        </el-tooltip>
        <!-- 全局设备上下文（FE-10）：下拉直绑 store，选一次跨模块保持。 -->
        <el-select v-model="store.selectedDeviceIp" :placeholder="t('console.selectDevicePlaceholder')" style="width: 220px">
          <el-option v-for="d in store.devices" :key="d.id" :label="d.ip" :value="d.ip" />
        </el-select>
      </div>
    </div>

    <el-alert v-if="schemaError" :title="schemaError" type="error" :closable="false" show-icon />

    <!-- 未选设备：引导先选设备（FE-10），不静默渲染空数据。
         schema 失败时让位给错误告警（此时选设备无济于事，引导反而误导）。 -->
    <el-empty
      v-if="!schemaError && !store.selectedDeviceIp"
      data-test="select-device-empty"
      :description="t('console.selectDeviceFirst')"
    />
    <!-- 一级 Tab：模块根顶层子节点派生（list→列表页、group/choice→表单页，FE-10）。
         Tab 组件常驻（不销毁），切换保留各 Tab 表单/搜索状态。 -->
    <el-tabs v-else-if="tabs.length" v-model="activeTab" class="console-tabs">
      <el-tab-pane v-for="tab in tabs" :key="tab.name" :label="tab.label" :name="tab.name">
        <ModuleListTab v-if="tab.kind === 'list'" :tab="tab" :root-name="rootName" :device="store.selectedDeviceIp" />
        <ModuleFormTab v-else :tab="tab" :root-name="rootName" :device="store.selectedDeviceIp" />
      </el-tab-pane>
    </el-tabs>
    <el-empty v-else-if="!schemaError" :description="t('console.schemaLoading')" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getYangSchema, getOwnership } from '../api'
import { localizeFields } from '../composables/useFieldLabels'
import { useLocaleStore } from '../stores/locale'
import { useMenuStore } from '../stores/menu'
import { useDeviceStore } from '../stores/device'
import type { Field } from '../utils/crdSchemaParser'
import { deriveTabs, type ConsoleTab } from '../utils/moduleConsole'
import ModuleListTab from '../components/config/ModuleListTab.vue'
import ModuleFormTab from '../components/config/ModuleFormTab.vue'

const route = useRoute()
const localeStore = useLocaleStore()
const menuStore = useMenuStore()
const { t } = useI18n()
const store = useDeviceStore()

const moduleName = computed(() => String(route.params.module || ''))
// 入页优先级 query > store：深链/「查看配置」显式指定则覆盖全局上下文，
// 无 query 时沿用既有选中（跨模块保持）。用 watch 而非仅 setup 一次性执行：
// 组件在 /module/:module 间复用，前进/后退到携带 ?device= 的历史条目也须生效。
// 重复 query 参数取首个（数组经 String 会拼出 'a,b' 垃圾值污染全局上下文）。
function applyDeviceQuery() {
  const q = route.query?.device
  const ip = Array.isArray(q) ? q[0] : q
  if (ip) store.selectDevice(String(ip))
}
applyDeviceQuery()
watch(() => route.query?.device, applyDeviceQuery)
const schemaError = ref('')
const title = ref('')
const vendor = ref('')
const rootName = ref('')
const schemaFields = ref<Field[]>([])

// 软归属（FE-18）：选中设备上本模块的认领意图清单；查询失败静默降级为无徽标（R08）。
const ownershipIntents = ref<string[]>([])
async function loadOwnership() {
  ownershipIntents.value = []
  if (!store.selectedDeviceIp || !moduleName.value) return
  try {
    const res = await getOwnership(store.selectedDeviceIp)
    const claims: any[] = res.data?.data?.claims || []
    const intents = new Set<string>()
    for (const c of claims) {
      if (c?.module === moduleName.value && c?.intent) intents.add(c.intent)
    }
    ownershipIntents.value = [...intents].sort()
  } catch {
    /* 无徽标即可，不打扰原生配置主流程 */
  }
}

const tabs = computed<ConsoleTab[]>(() => deriveTabs(schemaFields.value))
const activeTab = ref('')
const activeTabLabel = computed(() => tabs.value.find((t) => t.name === activeTab.value)?.label || '')

// 原始 schema（YANG 节点名标签）；展示层按语言经 res 查表重标（UI-03）。
let rawFields: any[] = []

async function relabelFields() {
  // 查不到/缺文件回退原始标签（R08）；locale 切换即时重查。res 懒加载为异步，
  // 首帧先渲染原始标签（不阻塞 Tab 派生），就绪后原位替换。
  const root = rootName.value
  const localized = await localizeFields(rawFields, root, localeStore.locale, menuStore.leftTree)
  if (rootName.value === root) schemaFields.value = localized
}

async function loadSchema() {
  schemaError.value = ''
  schemaFields.value = []
  try {
    const res = await getYangSchema(moduleName.value, 'nested')
    const data = res.data?.data
    rawFields = data?.fields ?? []
    title.value = data?.title || moduleName.value
    vendor.value = data?.vendor || ''
    // 运行时配置路径的根段 = 模块根容器名（schema title 即 root.Name()）。
    rootName.value = data?.title || moduleName.value
    schemaFields.value = rawFields
    activeTab.value = tabs.value[0]?.name || ''
    void relabelFields()
  } catch (e: any) {
    // schema 拉取失败降级：页面不崩，明确报错（R08/§9）。
    schemaError.value = e?.response?.data?.message || e?.message || t('console.schemaLoadFailed')
  }
}

watch(() => localeStore.locale, relabelFields)

watch(moduleName, loadSchema)
// immediate：全局上下文使「挂载时设备已选中」成为主流程（查看配置入口/跨页返回），
// 仅靠变化触发会漏掉首帧归属查询（FE-18 徽标静默缺失）。
watch([() => store.selectedDeviceIp, moduleName], loadOwnership, { immediate: true })

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
