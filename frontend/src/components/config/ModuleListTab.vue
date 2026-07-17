<template>
  <div class="module-list-tab">
    <!-- 工具栏：新增 + 高级搜索折叠开关（FE-11）；只读 Tab 无编辑入口（FE-14） -->
    <div class="toolbar">
      <el-button v-if="!tab.readonly" type="primary" :icon="Plus" :disabled="!device" @click="openAdd">新增</el-button>
      <el-button v-if="searchFields.length" link type="primary" class="adv-toggle" @click="searchOpen = !searchOpen">
        高级搜索
        <el-icon><ArrowUp v-if="searchOpen" /><ArrowDown v-else /></el-icon>
      </el-button>
    </div>

    <!-- 高级搜索面板：字段集 = support-filter 标注的叶（默认折叠） -->
    <el-collapse-transition>
      <div v-show="searchOpen" class="search-panel">
        <el-form inline @submit.prevent>
          <el-form-item v-for="f in searchFields" :key="f.path" :label="f.label">
            <el-select
              v-if="f.type === 'enum'"
              v-model="draft[keyOf(f)]"
              clearable
              :placeholder="`选择${f.label}`"
              class="search-ctl"
            >
              <el-option v-for="o in f.options" :key="String(o.value)" :label="o.label" :value="o.value" />
            </el-select>
            <el-input v-else v-model="draft[keyOf(f)]" clearable :placeholder="`输入${f.label}`" class="search-ctl" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="applySearch">查询</el-button>
            <el-button @click="resetSearch">重置</el-button>
          </el-form-item>
        </el-form>
      </div>
    </el-collapse-transition>

    <el-alert v-if="error" :title="error" type="warning" :closable="false" show-icon />

    <!-- 模型驱动数据表：列由 schema 分层派生；enum→Tag、up/down→状态点；
         带 when 的列按行数据求值，不满足显示 “-”（FE-11） -->
    <el-table :data="pagedRows" stripe v-loading="loading" class="list-table">
      <el-table-column
        v-for="col in columns"
        :key="col.path"
        :prop="keyOf(col)"
        :label="col.label"
        min-width="140"
      >
        <template #default="{ row }">
          <span v-if="!cellVisible(col, row)" class="cell-na">-</span>
          <span v-else-if="statusTone(row[keyOf(col)])" class="status-cell" :class="statusTone(row[keyOf(col)])">
            <span class="dot" aria-hidden="true"></span>{{ row[keyOf(col)] }}
          </span>
          <el-tag v-else-if="col.type === 'enum' && rowVal(row, col) !== ''" size="small" :type="tagType(col, row)">
            {{ rowVal(row, col) }}
          </el-tag>
          <el-tag v-else-if="col.type === 'boolean'" size="small" :type="row[keyOf(col)] ? 'success' : 'info'">
            {{ row[keyOf(col)] ? 'true' : 'false' }}
          </el-tag>
          <span v-else>{{ rowVal(row, col) }}</span>
        </template>
      </el-table-column>
      <el-table-column v-if="!tab.readonly && (canUpdate || canDelete)" label="操作" width="130" fixed="right">
        <template #default="{ row }">
          <el-button v-if="canUpdate" type="primary" size="small" link @click="openEdit(row)">编辑</el-button>
          <el-button
            v-if="canDelete"
            type="danger"
            size="small"
            link
            @click="onDelete(row)"
          >删除</el-button>
        </template>
      </el-table-column>
      <template #empty>
        <span>{{ device ? '暂无配置（点击新增）' : '请先选择设备' }}</span>
      </template>
    </el-table>

    <!-- 分页（客户端，FE-11） -->
    <div class="pager">
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="filteredRows.length"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next"
        background
      />
    </div>

    <!-- 新增/编辑抽屉：模型驱动表单 + 差异预览 + 对账进度（复用既有编排）；
         只读 Tab（state 子树）无编辑语义，整个抽屉不渲染（FE-14）。 -->
    <el-drawer v-if="!tab.readonly" v-model="drawerVisible" :title="editing ? '编辑' : '新增'" size="560px"
      :close-on-click-modal="!flowActive" :close-on-press-escape="!flowActive" @closed="onDrawerClosed">
      <template v-if="!flowActive">
        <el-form ref="formRef" :model="form.formData" :rules="form.rules.value" label-position="top" class="config-form">
          <el-form-item v-for="field in form.visibleFields.value" :key="field.path" :label="field.label"
            :prop="form.keyOf(field)">
            <FieldRenderer v-if="field.type === 'choice'" :field="field" :model-value="form.choiceScope(field)"
              @update:model-value="form.onChoiceUpdate(field, $event)" />
            <FieldRenderer v-else :field="field" :disabled="(editing && isCreateOnly(field)) || !!field.readonly"
              :model-value="form.formData[form.keyOf(field)]"
              @update:model-value="form.formData[form.keyOf(field)] = $event" />
          </el-form-item>
        </el-form>
        <DiffPreview :diff="form.diff.value" />
        <div class="form-tip">字段与约束由 YANG 模型生成，校验通过才会下发，下发即触发对账。</div>
      </template>
      <ReconcileSteps v-else :progress="submitFlow.progress.value" :timed-out="submitFlow.timedOut.value" />

      <template #footer>
        <template v-if="!flowActive">
          <el-button @click="drawerVisible = false">取消</el-button>
          <el-button type="primary" :disabled="!form.submittable.value" @click="submit">下发并对账</el-button>
        </template>
        <el-button v-else type="primary" :disabled="!flowDone" @click="drawerVisible = false">
          {{ flowDone ? '关闭' : '对账中…' }}
        </el-button>
      </template>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import { Plus, ArrowDown, ArrowUp } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance } from 'element-plus'
import { getConfig, deleteConfig } from '../../api'
import { ownershipRejectionOf, confirmOwnershipOverride } from '../../composables/ownershipGate'
import { useConfigSubmit } from '../../composables/useConfigSubmit'
import { useConfigForm } from '../../composables/useConfigForm'
import { useFreshnessStore } from '../../stores/freshness'
import type { Field } from '../../utils/crdSchemaParser'
import type { ConsoleTab } from '../../utils/moduleConsole'
import {
  deriveColumns,
  deriveKeyField,
  filterableFields,
  filterRows,
  cellVisible,
  configPathFor,
  statusTone,
  leafName,
} from '../../utils/moduleConsole'
import FieldRenderer from './FieldRenderer.vue'
import DiffPreview from './DiffPreview.vue'
import ReconcileSteps from './ReconcileSteps.vue'

const props = defineProps<{
  tab: ConsoleTab
  rootName: string
  device: string
}>()

const listField = computed<Field>(() => props.tab.listField || props.tab.field)
const configPath = computed(() => configPathFor(props.rootName, props.tab.field.path))
const keyField = computed(() => deriveKeyField(listField.value))
const columns = computed(() => deriveColumns(listField.value))
const searchFields = computed(() => filterableFields(listField.value))
// 编辑表单字段：list 全部子字段——readonly 叶禁用回显，payload/校验排除在 useConfigForm（FE-14）。
const itemFields = computed<Field[]>(() => listField.value.fields || [])

// list 级 operation-exclude 门禁（FE-11）；叶级 exclude 见 isCreateOnly。
const canUpdate = computed(() => !props.tab.field.operationExclude?.includes('update') &&
  !listField.value.operationExclude?.includes('update'))
const canDelete = computed(() => !props.tab.field.operationExclude?.includes('delete') &&
  !listField.value.operationExclude?.includes('delete'))

function isCreateOnly(f: Field): boolean {
  return !!f.operationExclude?.includes('update')
}

function keyOf(f: Field): string {
  return leafName(f)
}

function rowVal(row: Record<string, any>, col: Field): string {
  const v = row[keyOf(col)]
  return v == null ? '' : String(v)
}

// enum Tag 色板轮转（按枚举值序号取色，非语义映射，R05）。
const TAG_TYPES = ['primary', 'success', 'warning', 'info', 'danger'] as const
function tagType(col: Field, row: Record<string, any>) {
  const idx = (col.options || []).findIndex((o) => String(o.value) === rowVal(row, col))
  return TAG_TYPES[Math.max(idx, 0) % TAG_TYPES.length]
}

// ===== 数据加载 =====
const items = ref<Record<string, any>[]>([])
const loading = ref(false)
const error = ref('')
// POST 包裹键：默认 list 名，回读命中容器名（如 vlan 的 vlans）时跟随实际键。
const postKey = ref('')

function normalizeRows(subtree: any): { rows: Record<string, any>[]; key: string } {
  const candidates = [leafName(listField.value), leafName(props.tab.field)]
  for (const k of candidates) {
    const v = subtree?.[k]
    if (Array.isArray(v)) return { rows: v, key: k }
    if (v && typeof v === 'object') {
      return {
        rows: Object.entries(v).map(([kk, vv]) =>
          typeof vv === 'object' && vv !== null
            ? { [keyField.value]: isNaN(Number(kk)) ? kk : Number(kk), ...(vv as object) }
            : { [keyField.value]: kk },
        ),
        key: k,
      }
    }
  }
  if (Array.isArray(subtree)) return { rows: subtree, key: candidates[0] }
  return { rows: [], key: candidates[0] }
}

async function load() {
  if (!props.device) {
    items.value = []
    return
  }
  loading.value = true
  error.value = ''
  try {
    const res = await getConfig(props.device, configPath.value)
    const payload = res.data?.data
    useFreshnessStore().record({
      cache_age_seconds: payload?.cache_age_seconds,
      ttl_seconds: payload?.ttl_seconds,
      source: payload?.source,
    })
    const { rows, key } = normalizeRows(payload?.data ?? payload)
    items.value = rows
    postKey.value = key
  } catch (e: any) {
    error.value = e?.response?.data?.message || e?.message || '读取失败'
    items.value = []
  } finally {
    loading.value = false
  }
}

watch(() => props.device, load, { immediate: true })

// ===== 高级搜索（草稿→应用，查询/重置，FE-11） =====
const searchOpen = ref(false)
const draft = reactive<Record<string, any>>({})
const applied = ref<Record<string, any>>({})

function applySearch() {
  applied.value = { ...draft }
  page.value = 1
}
function resetSearch() {
  Object.keys(draft).forEach((k) => delete draft[k])
  applied.value = {}
  page.value = 1
}

const filteredRows = computed(() => filterRows(items.value, applied.value, searchFields.value))

// ===== 分页（客户端） =====
const page = ref(1)
const pageSize = ref(10)
const pagedRows = computed(() =>
  filteredRows.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value),
)

// ===== 新增/编辑抽屉（复用通用表单编排 + 对账流） =====
const drawerVisible = ref(false)
const editing = ref(false)
const formRef = ref<FormInstance>()
const form = useConfigForm(itemFields, keyField)
// useConfigSubmit 在 run() 时读取 opts 字段，故传响应式代理以跟随 postKey。
const submitOpts = reactive({ configPath: '', listKey: '' })
watch([configPath, postKey, listField], () => {
  submitOpts.configPath = configPath.value
  submitOpts.listKey = postKey.value || leafName(listField.value)
}, { immediate: true })
const submitFlow = useConfigSubmit(submitOpts)

const flowActive = computed(() => submitFlow.phase.value !== 'idle')
const flowDone = computed(() => submitFlow.progress.value.done || submitFlow.timedOut.value)

function openAdd() {
  editing.value = false
  submitFlow.reset()
  form.resetForm()
  formRef.value?.clearValidate()
  drawerVisible.value = true
}

function openEdit(row: Record<string, any>) {
  editing.value = true
  submitFlow.reset()
  form.resetForm({ ...row })
  drawerVisible.value = true
}

async function submit() {
  if (!props.device) return
  if (formRef.value) {
    try {
      await formRef.value.validate()
    } catch {
      /* 行内提示即可；下方权威门禁拦截 */
    }
  }
  if (form.blocked.value) return
  await submitFlow.run(props.device, form.visiblePayload())
  if (submitFlow.phase.value !== 'error') await load()
}

function onDrawerClosed() {
  submitFlow.reset()
}

// ===== 行删除（FE-16，命令语义）：确认 → DELETE → 成功刷新 / 失败如实透出（§9） =====
async function onDelete(row: Record<string, any>) {
  const key = row[keyField.value]
  try {
    await ElMessageBox.confirm(
      `将从设备删除 ${keyField.value} = ${key} 的条目，该操作不可撤销。`,
      '确认删除',
      { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' },
    )
  } catch {
    return // 用户取消：零请求
  }
  try {
    const res = await deleteConfig(props.device, configPath.value, key)
    // 归属硬锁（FE-18 二期）：信封 409 → 阻断确认 → 确认后携 force 重发，取消中止。
    const rej = ownershipRejectionOf(res)
    if (rej) {
      if (!(await confirmOwnershipOverride(rej))) return
      const forced = await deleteConfig(props.device, configPath.value, key, true)
      // 信封恒 200：force 重发失败按 success 判定，如实透出（§9）。
      if ((forced.data as any)?.success === false) {
        error.value = (forced.data as any)?.message || '强制删除失败'
        return
      }
    }
    ElMessage.success('已删除并触发对账')
    error.value = ''
    await load()
  } catch (e: any) {
    // 设备/门禁错误如实展示，列表保持原状（R08/§9）。
    error.value = e?.response?.data?.message || e?.message || '删除失败'
  }
}
</script>

<style scoped>
.module-list-tab {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 4px;
}

.adv-toggle {
  margin-left: 8px;
}

.search-panel {
  padding: 12px 14px 0;
  background: var(--sunken, #f5f7fa);
  border: 1px solid var(--line, #e4e7ed);
  border-radius: 8px;
}

.search-ctl {
  width: 200px;
}

.list-table {
  background: #fff;
  border-radius: 8px;
}

.cell-na {
  color: var(--ink-3, #93a2b1);
}

.status-cell {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.status-cell .dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-cell.ok .dot {
  background: var(--st-conv, #10814a);
}

.status-cell.bad .dot {
  background: var(--st-off, #c45656);
}

.pager {
  display: flex;
  justify-content: flex-end;
}

.config-form {
  padding: 0 4px;
}

.form-tip {
  margin-top: 14px;
  font-size: 11.5px;
  line-height: 1.6;
  color: var(--ink-3, #93a2b1);
}
</style>
