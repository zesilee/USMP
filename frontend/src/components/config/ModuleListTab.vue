<template>
  <div class="module-list-tab">
    <!-- 工具区（FE-11）：创建/刷新 + 高级搜索开关 + 列设置；只读 Tab 无编辑入口（FE-14） -->
    <div class="toolbar">
      <el-button v-if="!tab.readonly" type="primary" :icon="Plus" :disabled="!device" data-test="list-create" @click="openCreate">
        {{ t('common.create') }}
      </el-button>
      <el-button :icon="RefreshRight" :disabled="!device" data-test="list-refresh" @click="load()">
        {{ t('common.refresh') }}
      </el-button>
      <!-- 批量菜单（FE-11 二期）：多选行批量删除入变更集 -->
      <el-dropdown v-if="!tab.readonly && canDelete" trigger="click" @command="onBatchCommand">
        <el-button data-test="batch-more">
          {{ t('console.moreActions') }}<el-icon><ArrowDown /></el-icon>
        </el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="batch-delete" :disabled="!selectedRows.length" data-test="batch-delete">
              {{ t('console.batchDelete') }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <el-button v-if="searchFields.length" link type="primary" class="adv-toggle" @click="searchOpen = !searchOpen">
        {{ t('console.advancedSearch') }}
        <el-icon><ArrowUp v-if="searchOpen" /><ArrowDown v-else /></el-icon>
      </el-button>
      <div class="toolbar-spacer" />
      <!-- 列设置（FE-11）：全集勾选显隐，默认集=分层前 9 -->
      <el-popover placement="bottom-end" :width="220" trigger="click">
        <template #reference>
          <el-button :icon="Setting" circle data-test="column-settings" :title="t('console.columnSettings')" />
        </template>
        <el-checkbox-group v-model="visibleCols" class="col-settings">
          <el-checkbox v-for="c in allColumns" :key="c.path" :value="c.path" :label="c.path">
            {{ c.label }}
          </el-checkbox>
        </el-checkbox-group>
      </el-popover>
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
              :placeholder="t('console.selectPlaceholder', { label: f.label })"
              class="search-ctl"
            >
              <el-option v-for="o in f.options" :key="String(o.value)" :label="o.label" :value="o.value" />
            </el-select>
            <el-input v-else v-model="draft[keyOf(f)]" clearable :placeholder="t('console.inputPlaceholder', { label: f.label })" class="search-ctl" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="applySearch">{{ t('common.apply') }}</el-button>
            <el-button @click="resetSearch">{{ t('common.reset') }}</el-button>
          </el-form-item>
        </el-form>
      </div>
    </el-collapse-transition>

    <el-alert v-if="error" :title="error" type="warning" :closable="false" show-icon />

    <!-- 模型驱动数据表（FE-11）：多选列 + 全列排序 + enum/boolean 列头筛选 +
         列设置显隐；点行/点编辑 → 下方详情区（FE-21），抽屉退役。 -->
    <el-table
      ref="tableRef"
      :data="pagedRows"
      stripe
      v-loading="loading"
      class="list-table"
      highlight-current-row
      :row-class-name="rowClass"
      @row-click="onRowClick"
      @selection-change="onSelectionChange"
    >
      <el-table-column type="selection" width="42" />
      <!-- 变更集标记合成视图（FE-11 二期）：待创建/已修改/待删除 -->
      <el-table-column width="72">
        <template #default="{ row }">
          <el-tag v-if="rowMark(row) === 'create'" size="small" type="success" data-test="mark-create">{{ t('console.markCreate') }}</el-tag>
          <el-tag v-else-if="rowMark(row) === 'update'" size="small" type="warning" data-test="mark-update">{{ t('console.markUpdate') }}</el-tag>
          <el-tag v-else-if="rowMark(row) === 'delete'" size="small" type="danger" data-test="mark-delete">{{ t('console.markDelete') }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column
        v-for="col in shownColumns"
        :key="col.path"
        :prop="keyOf(col)"
        :label="col.label"
        min-width="140"
        sortable
        :filters="headerFilters(col)"
        :filter-method="headerFilters(col) ? makeFilterMethod(col) : undefined"
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
      <el-table-column v-if="!tab.readonly" :label="t('common.actions')" width="200" fixed="right">
        <template #default="{ row }">
          <el-button v-if="canUpdate" type="primary" size="small" link @click.stop="openEdit(row)">{{ t('common.edit') }}</el-button>
          <el-button
            v-if="canDelete && rowMark(row) === 'delete'"
            type="warning"
            size="small"
            link
            data-test="undelete-btn"
            @click.stop="onUndelete(row)"
          >{{ t('console.undelete') }}</el-button>
          <el-button
            v-else-if="canDelete"
            type="danger"
            size="small"
            link
            @click.stop="onDelete(row)"
          >{{ t('common.delete') }}</el-button>
          <el-button type="primary" size="small" link @click.stop="fetchSource">{{ t('console.fetchSource') }}</el-button>
        </template>
      </el-table-column>
      <template #empty>
        <span>{{ device ? t('console.emptyNoConfig') : t('console.emptySelectDevice') }}</span>
      </template>
    </el-table>

    <!-- 查询时间戳 + 总记录数（过滤后全集）+ 分页（含跳页，FE-11） -->
    <div class="table-footer">
      <span v-if="queryAt" data-test="query-summary" class="query-summary">
        {{ t('console.queryFinished', { time: queryAt, total: filteredRows.length }) }}
      </span>
      <el-pagination
        v-model:current-page="page"
        v-model:page-size="pageSize"
        :total="filteredRows.length"
        :page-sizes="[10, 20, 50]"
        layout="total, sizes, prev, pager, next, jumper"
        background
      />
    </div>

    <!-- 列表详情同屏编辑区（FE-21）：点行/编辑=编辑态、创建=空表单；只读 Tab 不渲染 -->
    <ItemDetailPane
      v-if="detailMode && !tab.readonly"
      ref="paneRef"
      :tab="tab"
      :root-name="rootName"
      :device="device"
      :mode="detailMode"
      :row="selectedRow"
      :post-key="postKey || leafName(listField)"
      @close="closeDetail"
      @staged="onStaged"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, ArrowDown, ArrowUp, RefreshRight, Setting } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type TableInstance } from 'element-plus'
import { getConfig } from '../../api'
import { useFreshnessStore } from '../../stores/freshness'
import { useChangesetStore } from '../../stores/changeset'
import type { Field } from '../../utils/crdSchemaParser'
import type { ConsoleTab } from '../../utils/moduleConsole'
import {
  deriveColumns,
  deriveAllColumns,
  deriveKeyField,
  filterableFields,
  filterRows,
  cellVisible,
  configPathFor,
  statusTone,
  leafName,
} from '../../utils/moduleConsole'
import ItemDetailPane from './ItemDetailPane.vue'

const props = defineProps<{
  tab: ConsoleTab
  rootName: string
  device: string
}>()

const { t } = useI18n()

const listField = computed<Field>(() => props.tab.listField || props.tab.field)
const configPath = computed(() => configPathFor(props.rootName, props.tab.field.path))
// 变更集条目路径（与 ItemDetailPane 同源）：后端契约按锚点前缀匹配，带前导斜杠。
const entryPath = computed(() => '/' + configPath.value)
const changeset = useChangesetStore()
const keyField = computed(() => deriveKeyField(listField.value))
const allColumns = computed(() => deriveAllColumns(listField.value))
const searchFields = computed(() => filterableFields(listField.value))

// 列设置（FE-11）：默认显示集 = 分层前 9；勾选态仅本页会话内生效。
const visibleCols = ref<string[]>(deriveColumns(listField.value, 9).map((c) => c.path))
watch(listField, (lf) => {
  visibleCols.value = deriveColumns(lf, 9).map((c) => c.path)
})
const shownColumns = computed(() => allColumns.value.filter((c) => visibleCols.value.includes(c.path)))

// 列头筛选（FE-11）：enum 用选项集、boolean 用 true/false；其余列走高级搜索。
function headerFilters(col: Field): { text: string; value: any }[] | undefined {
  if (col.type === 'enum' && col.options?.length) {
    return col.options.map((o) => ({ text: String(o.label ?? o.value), value: o.value }))
  }
  if (col.type === 'boolean') {
    return [
      { text: 'true', value: true },
      { text: 'false', value: false },
    ]
  }
  return undefined
}
function makeFilterMethod(col: Field) {
  return (value: any, row: Record<string, any>) => String(row[keyOf(col)]) === String(value)
}

// list 级 operation-exclude 门禁（FE-11）。
const canUpdate = computed(() => !props.tab.field.operationExclude?.includes('update') &&
  !listField.value.operationExclude?.includes('update'))
const canDelete = computed(() => !props.tab.field.operationExclude?.includes('delete') &&
  !listField.value.operationExclude?.includes('delete'))

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
// 查询完成时刻（FE-11）：随每次 load/获取数据源刷新。
const queryAt = ref('')
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

async function load(force = false) {
  if (!props.device) {
    items.value = []
    return
  }
  loading.value = true
  error.value = ''
  try {
    const res = await getConfig(props.device, configPath.value, force)
    const payload = res.data?.data
    useFreshnessStore().record({
      cache_age_seconds: payload?.cache_age_seconds,
      ttl_seconds: payload?.ttl_seconds,
      source: payload?.source,
    })
    const { rows, key } = normalizeRows(payload?.data ?? payload)
    items.value = rows
    postKey.value = key
    queryAt.value = new Date().toLocaleString()
    syncCurrentRow()
  } catch (e: any) {
    error.value = e?.response?.data?.message || e?.message || (force ? t('console.fetchSourceFailed') : t('console.readFailed'))
    // 强制回读失败保留原列表（§9 保留原配置）；常规加载失败清空避免陈旧数据误导。
    if (!force) items.value = []
  } finally {
    loading.value = false
  }
}

// 行操作「获取数据源」（FE-11/BR-04）：force_refresh 绕缓存回读该 list 路径。
// API 无单行读取粒度——整表回读并保持行选中，不伪造单行语义。
function fetchSource() {
  return load(true)
}

watch(() => props.device, () => load(), { immediate: true })

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

// 标记合成视图（FE-11 二期）：设备行 + 变更集待创建行（按主键去重，设备行优先）。
const mergedItems = computed(() => {
  const existing = new Set(items.value.map((r) => String(r[keyField.value])))
  const pendingCreates = changeset
    .entriesFor(props.device)
    .filter((e) => e.path === entryPath.value && e.op === 'create' && !existing.has(String(e.keyValue)))
    .map((e) => ({ ...(e.payload ?? {}) }))
  return [...items.value, ...pendingCreates]
})

// 行标记：create/update/delete/''（重置或提交清空后即时还原）。
function rowMark(row: Record<string, any>): '' | 'create' | 'update' | 'delete' {
  const e = changeset.entryFor(props.device, entryPath.value, String(row[keyField.value]))
  return (e?.op as any) ?? ''
}

function rowClass({ row }: { row: Record<string, any> }): string {
  const m = rowMark(row)
  return m ? `row-${m}` : ''
}

const filteredRows = computed(() => filterRows(mergedItems.value, applied.value, searchFields.value))

// ===== 分页（客户端） =====
const page = ref(1)
const pageSize = ref(10)
const pagedRows = computed(() =>
  filteredRows.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value),
)

// ===== 列表详情同屏（FE-21）：选中行 + 详情态，切行/切建带未提交草稿确认 =====
const tableRef = ref<TableInstance>()
const selectedRows = ref<Record<string, any>[]>([])
function onSelectionChange(rows: Record<string, any>[]) {
  selectedRows.value = rows
}
const paneRef = ref<InstanceType<typeof ItemDetailPane>>()
const selectedRow = ref<Record<string, any> | null>(null)
const detailMode = ref<'edit' | 'create' | null>(null)

// 未提交草稿守卫（FE-21 负路径）：取消则停留原条目、草稿保留。
async function ensureNoDraft(): Promise<boolean> {
  if (!detailMode.value || !paneRef.value?.dirty) return true
  try {
    await ElMessageBox.confirm(t('console.unsavedSwitch'), t('console.unsavedTitle'), {
      type: 'warning',
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
    })
    return true
  } catch {
    return false
  }
}

function highlight(row: Record<string, any> | null) {
  nextTick(() => tableRef.value?.setCurrentRow(row ?? undefined))
}

async function openEdit(row: Record<string, any>) {
  if (props.tab.readonly) return
  const same = detailMode.value === 'edit' && selectedRow.value?.[keyField.value] === row[keyField.value]
  if (same) return
  if (!(await ensureNoDraft())) {
    highlight(selectedRow.value) // 行点击已把高亮切走：退回原选中行
    return
  }
  selectedRow.value = row
  detailMode.value = 'edit'
  highlight(row)
}

function onRowClick(row: Record<string, any>) {
  void openEdit(row)
}

async function openCreate() {
  if (!(await ensureNoDraft())) return
  selectedRow.value = null
  detailMode.value = 'create'
  highlight(null)
}

function closeDetail() {
  detailMode.value = null
  selectedRow.value = null
  highlight(null)
}

// 重载后行对象身份变化：按主键把选中行对齐到新数据（详情表单同步刷新）。
function syncCurrentRow() {
  if (detailMode.value !== 'edit' || !selectedRow.value) return
  const key = selectedRow.value[keyField.value]
  const found = items.value.find((r) => r[keyField.value] === key) || null
  selectedRow.value = found
  if (!found) detailMode.value = null // 条目已消失（如他处删除）：收起详情，不悬挂陈旧表单
  highlight(found)
}

// 确定入集（FE-21 攒批）：设备数据未变不重载；标记行即时出现；创建态切为
// 该待创建条目的编辑态（合成视图行）。
function onStaged(key: string) {
  ElMessage.success(t('console.stagedOk'))
  const found = mergedItems.value.find((r) => String(r[keyField.value]) === key) || null
  if (found) {
    selectedRow.value = found
    detailMode.value = 'edit'
    highlight(found)
  }
}

// ===== 行删除（FE-16 攒批）：确认 → 入变更集删除项（零请求）；提交经「提交配置」 =====
async function onDelete(row: Record<string, any>) {
  const key = String(row[keyField.value])
  const pendingCreate = rowMark(row) === 'create'
  try {
    await ElMessageBox.confirm(
      t('console.deleteConfirm', { key: keyField.value, value: key }),
      t('common.confirmDelete'),
      { type: 'warning', confirmButtonText: t('common.delete'), cancelButtonText: t('common.cancel') },
    )
  } catch {
    return // 用户取消：变更集零改动
  }
  changeset.markDelete(props.device, {
    path: entryPath.value,
    listKey: postKey.value || leafName(listField.value),
    keyValue: key,
    label: `${props.tab.label} ${key}`,
  })
  // 删除的是当前详情条目 → 收起详情（待创建被移除/既有转待删除，均不悬挂表单）。
  if (String(selectedRow.value?.[keyField.value] ?? '') === key) closeDetail()
  ElMessage.success(pendingCreate ? t('console.pendingCreateRemoved') : t('console.markedDelete'))
}

// 取消删除（FE-16）：移除该删除项，行恢复常态。
function onUndelete(row: Record<string, any>) {
  changeset.unmarkDelete(props.device, entryPath.value, String(row[keyField.value]))
}

// 批量删除（FE-11 二期）：选中行逐条入集（list 级门禁由入口控制）。
async function onBatchCommand(cmd: string) {
  if (cmd !== 'batch-delete' || !selectedRows.value.length) return
  try {
    await ElMessageBox.confirm(
      t('console.batchDeleteConfirm', { n: selectedRows.value.length }),
      t('common.confirmDelete'),
      { type: 'warning', confirmButtonText: t('common.delete'), cancelButtonText: t('common.cancel') },
    )
  } catch {
    return
  }
  for (const row of selectedRows.value) {
    const key = String(row[keyField.value])
    changeset.markDelete(props.device, {
      path: entryPath.value,
      listKey: postKey.value || leafName(listField.value),
      keyValue: key,
      label: `${props.tab.label} ${key}`,
    })
    if (String(selectedRow.value?.[keyField.value] ?? '') === key) closeDetail()
  }
  ElMessage.success(t('console.batchMarkedDelete', { n: selectedRows.value.length }))
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

.toolbar-spacer {
  flex: 1;
}

.adv-toggle {
  margin-left: 8px;
}

.col-settings {
  display: flex;
  flex-direction: column;
  max-height: 320px;
  overflow-y: auto;
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

.table-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.query-summary {
  font-size: 12.5px;
  color: var(--ink-2, #5c6b7a);
}
</style>
