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
        <!-- 攒批工具栏（FE-23，一期预留位）：变更内容/试运行/重置/提交配置。
             提交编排 PR-5 接入（commit-request 暂为占位事件）。 -->
        <BatchToolbar
          v-if="store.selectedDeviceIp"
          :device="store.selectedDeviceIp"
          @commit-request="onCommitRequest"
          @reset="onBatchReset"
        />
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
    <!-- rpc 直达模式（FE-19）：左树 rpc 节点入口，仅渲染该 rpc 执行面板；
         未知 rpcName 明确报错不崩（R08）。 -->
    <template v-else-if="rpcMode">
      <div v-if="activeRpcTab" class="console-tabs">
        <RpcExecuteTab :tab="activeRpcTab" :module="rootName" :device="store.selectedDeviceIp" />
      </div>
      <el-alert
        v-else-if="schemaLoaded"
        data-test="rpc-not-found"
        :title="t('console.rpcNotFound', { rpc: rpcName })"
        type="error"
        :closable="false"
        show-icon
      />
      <el-empty v-else :description="t('console.schemaLoading')" />
    </template>
    <!-- 一级 Tab：模块根顶层子节点派生（list→列表页、group/choice→表单页，FE-10）。
         rpc 不再进 Tab 栏（FE-19 导航落点=左树）。Tab 组件常驻（不销毁），切换
         保留各 Tab 表单/搜索状态。 -->
    <el-tabs v-else-if="tabs.length" v-model="activeTab" class="console-tabs">
      <el-tab-pane v-for="tab in tabs" :key="tab.name" :name="tab.name">
        <!-- FE-24 Tab 头淡化：设备不支持的页签诚实透出（不隐藏），仅视觉降级。 -->
        <template #label>
          <span
            :class="{ 'tab-unsupported': unsupportedTabs.includes(tab.name) }"
            :data-test="unsupportedTabs.includes(tab.name) ? 'tab-unsupported' : undefined"
          >{{ tab.label }}</span>
        </template>
        <!-- consoleEpoch：重置/提交成功后整体重挂 → 表单/列表回设备实际态并清标记 -->
        <ModuleListTab v-if="tab.kind === 'list'" :key="consoleEpoch" :tab="tab" :root-name="rootName" :device="store.selectedDeviceIp" :unsupported="unsupportedTabs.includes(tab.name)" />
        <ModuleFormTab v-else :key="consoleEpoch" :tab="tab" :root-name="rootName" :device="store.selectedDeviceIp" :unsupported="unsupportedTabs.includes(tab.name)" />
      </el-tab-pane>
    </el-tabs>
    <el-empty v-else-if="!schemaError" :description="t('console.schemaLoading')" />
    <BatchCommitDialog v-model:visible="commitOpen" :device="store.selectedDeviceIp" @committed="onCommitted" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRoute, onBeforeRouteLeave } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessageBox } from 'element-plus'
import { getYangSchema, getOwnership } from '../api'
import { localizeFields, localizeRpcs } from '../composables/useFieldLabels'
import { useLocaleStore } from '../stores/locale'
import { useMenuStore } from '../stores/menu'
import { useDeviceStore } from '../stores/device'
import { useChangesetStore } from '../stores/changeset'
import type { Field } from '../utils/crdSchemaParser'
import { deriveTabs, deriveRpcTabs, type ConsoleTab, type RpcDef } from '../utils/moduleConsole'
import BatchToolbar from '../components/config/BatchToolbar.vue'
import BatchCommitDialog from '../components/config/BatchCommitDialog.vue'
import ModuleListTab from '../components/config/ModuleListTab.vue'
import ModuleFormTab from '../components/config/ModuleFormTab.vue'
import RpcExecuteTab from '../components/config/RpcExecuteTab.vue'

const route = useRoute()
const localeStore = useLocaleStore()
const menuStore = useMenuStore()
const { t } = useI18n()
const store = useDeviceStore()
const changeset = useChangesetStore()

// ===== 攒批提交/重置编排（FE-03/FE-23） =====
const commitOpen = ref(false)
// 重置/提交成功 → 重挂 Tab 内容组件：表单回设备实际态、列表标记行还原。
const consoleEpoch = ref(0)

async function onCommitRequest() {
  const n = changeset.countFor(store.selectedDeviceIp)
  if (!n) return
  try {
    await ElMessageBox.confirm(
      t('console.batch.commitConfirm', { count: n }),
      t('console.batch.commit'),
      { type: 'warning' },
    )
  } catch {
    return
  }
  commitOpen.value = true
}

function onCommitted() {
  consoleEpoch.value++
}

function onBatchReset() {
  consoleEpoch.value++
}

// 路由离开确认（FE-23 负路径）：存在未提交变更时提示；取消停留、变更集保留。
onBeforeRouteLeave(async () => {
  if (!changeset.countFor(store.selectedDeviceIp)) return true
  try {
    await ElMessageBox.confirm(t('console.batch.leaveConfirm'), t('console.batch.changes'), { type: 'warning' })
    return true
  } catch {
    return false
  }
})

const moduleName = computed(() => String(route.params.module || ''))
// rpc 直达模式（FE-19）：/module/:module/rpc/:rpcName，仅渲染该 rpc 执行面板。
const rpcName = computed(() => String(route.params.rpcName || ''))
const rpcMode = computed(() => !!rpcName.value)
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
// 模块 rpc（FE-19）：与容器 Tab 平级呈现。首帧用后端原始名，res 就绪后经
// localizeRpcs 原位替换为本地化标签（UI-03，与配置字段同款查表）。
const rpcs = ref<RpcDef[]>([])

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

// 配置容器 Tab（FE-10）；rpc 不再进 Tab 栏（FE-19：导航落点迁移到左树）。
const tabs = computed<ConsoleTab[]>(() => deriveTabs(schemaFields.value))
// rpc 直达面板：按路由 rpcName 从派生 rpc Tab 中取（label/input 已本地化，UI-03）。
const activeRpcTab = computed<ConsoleTab | undefined>(() =>
  deriveRpcTabs(rpcs.value).find((t) => t.rpc?.name === rpcName.value),
)
// schema 是否已就位（rpc 模式区分「加载中」与「rpc 名不存在」，R08 明确报错）。
const schemaLoaded = computed(() => !!rootName.value)
const activeTab = ref('')
const activeTabLabel = computed(() =>
  rpcMode.value
    ? activeRpcTab.value?.label || ''
    : tabs.value.find((t) => t.name === activeTab.value)?.label || '',
)

// 原始 schema（YANG 节点名标签）；展示层按语言经 res 查表重标（UI-03）。
let rawFields: any[] = []
// 原始 rpc（YANG 节点名标签）；同经 res 查表重标（UI-03 rpc 扩展）。
let rawRpcs: RpcDef[] = []

async function relabelFields() {
  // 查不到/缺文件回退原始标签（R08）；locale 切换即时重查。res 懒加载为异步，
  // 首帧先渲染原始标签（不阻塞 Tab 派生），就绪后原位替换。配置字段与 rpc 同源
  // 并行查表，同一 rootName 守卫防止快速切模块时的回填竞态。
  const root = rootName.value
  const [localizedFields, localizedRpcs] = await Promise.all([
    localizeFields(rawFields, root, localeStore.locale, menuStore.leftTree),
    localizeRpcs(rawRpcs, root, localeStore.locale, menuStore.leftTree),
  ])
  if (rootName.value === root) {
    schemaFields.value = localizedFields
    rpcs.value = localizedRpcs
  }
}

// 设备不支持预标记（FE-24/CN-05）：schema ?device= 响应附 unsupported（相对模块根
// 首段名 = Tab name）；空集省略键。命中的 Tab 直接占位、不发取数请求。
const unsupportedTabs = ref<string[]>([])

async function loadSchema() {
  schemaError.value = ''
  schemaFields.value = []
  try {
    // 带设备取 schema：零额外请求拿到该设备的不支持预标记（CN-05）。
    const res = await getYangSchema(moduleName.value, 'nested', store.selectedDeviceIp || undefined)
    const data = res.data?.data
    unsupportedTabs.value = (data?.unsupported ?? []) as string[]
    rawFields = data?.fields ?? []
    rawRpcs = (data?.rpcs ?? []) as RpcDef[]
    rpcs.value = rawRpcs
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
// 设备切换重拉 schema（FE-24）：unsupported 预标记按设备学习，不可跨设备沿用。
watch(() => store.selectedDeviceIp, () => loadSchema())
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

/* FE-24：设备不支持页签的 Tab 头淡化（诚实透出，不隐藏） */
.tab-unsupported {
  color: var(--ink-3, #93a2b1);
}
</style>
