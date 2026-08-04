<template>
  <div class="devices">
    <div class="page-header">
      <div>
        <h2>{{ t('devices.title') }}</h2>
        <div class="sub">{{ t('devices.subtitle') }}</div>
      </div>
      <div class="header-actions">
        <el-input v-model="searchKeyword" :placeholder="t('devices.searchPlaceholder')" :prefix-icon="Search" clearable class="search-input" />
        <el-select v-model="statusFilter" :placeholder="t('devices.allStatus')" clearable class="filter-select">
          <el-option :label="t('common.online')" value="online" />
          <el-option :label="t('common.offline')" value="offline" />
        </el-select>
        <el-select v-model="vendorFilter" :placeholder="t('devices.allVendors')" clearable class="filter-select">
          <el-option v-for="v in vendors" :key="v" :label="v" :value="v" />
        </el-select>
        <el-button :icon="Refresh" @click="handleRefresh" :loading="loading">{{ t('common.refresh') }}</el-button>
        <el-button type="primary" :icon="Plus" data-test="add-device-btn" @click="openAddDialog">{{ t('devices.addDevice') }}</el-button>
      </div>
    </div>

    <el-dialog v-model="addDialogVisible" :title="t('devices.addDeviceTitle')" width="460px">
      <div data-test="add-device-dialog">
        <el-form ref="addFormRef" :model="addForm" :rules="addRules" label-width="90px">
          <el-form-item :label="t('devices.fieldIp')" prop="ip" data-test="add-ip">
            <el-input v-model="addForm.ip" placeholder="7.225.21.14" />
          </el-form-item>
          <el-form-item :label="t('devices.fieldPort')" prop="port" data-test="add-port">
            <el-input v-model="addForm.port" placeholder="830" />
          </el-form-item>
          <el-form-item :label="t('devices.fieldUsername')" prop="username" data-test="add-username">
            <el-input v-model="addForm.username" autocomplete="off" />
          </el-form-item>
          <el-form-item :label="t('devices.fieldPassword')" prop="password" data-test="add-password">
            <el-input v-model="addForm.password" type="password" show-password autocomplete="new-password" />
          </el-form-item>
          <el-form-item :label="t('devices.fieldVendor')" prop="vendor" data-test="add-vendor">
            <el-input v-model="addForm.vendor" placeholder="huawei" />
          </el-form-item>
          <el-form-item :label="t('devices.fieldRole')" prop="role" data-test="add-role">
            <el-input v-model="addForm.role" :placeholder="t('devices.rolePlaceholder')" />
          </el-form-item>
        </el-form>
      </div>
      <template #footer>
        <el-button @click="addDialogVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" data-test="add-submit" :loading="adding" @click="submitAddDevice">
          {{ t('common.confirm') }}
        </el-button>
      </template>
    </el-dialog>

    <el-table :data="paginatedRows" class="device-table" v-loading="loading">
      <el-table-column :label="t('devices.colIp')" width="150">
        <template #default="{ row }"><span class="mono strong">{{ row.ip }}</span></template>
      </el-table-column>
      <el-table-column :label="t('devices.colName')" width="170">
        <template #default="{ row }"><span class="strong">{{ row.name || '—' }}</span></template>
      </el-table-column>
      <el-table-column :label="t('devices.colVendorModel')" min-width="150">
        <template #default="{ row }"><span class="dim">{{ row.vendorModel || '—' }}</span></template>
      </el-table-column>
      <el-table-column :label="t('devices.colRole')" width="100">
        <template #default="{ row }">
          <el-tag v-if="row.role" size="small" type="info" data-test="device-role">{{ row.role }}</el-tag>
          <span v-else class="dim">—</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('devices.colSession')" width="120">
        <template #default="{ row }">
          <span class="chip" :class="row.session === 'connected' ? 'conv' : 'off'">
            <span class="glyph" aria-hidden="true"></span>{{ row.session === 'connected' ? t('devices.sessionConnected') : t('devices.sessionDisconnected') }}
          </span>
        </template>
      </el-table-column>
      <el-table-column :label="t('devices.colLoad')" width="110">
        <template #default="{ row }"><Sparkline :points="row.load" /></template>
      </el-table-column>
      <el-table-column :label="t('devices.colReconcile')" width="120">
        <template #default="{ row }"><ReconcileChip :state="row.reconcileState" /></template>
      </el-table-column>
      <el-table-column :label="t('devices.colLastSync')" min-width="140">
        <template #default="{ row }"><span class="mono dim">{{ row.lastSync || '—' }}</span></template>
      </el-table-column>
      <el-table-column :label="t('common.actions')" width="180" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" size="small" link @click="goToConfig(row)">{{ t('devices.viewConfig') }}</el-button>
          <el-button type="info" size="small" link @click="handleTestConnection(row)">{{ t('devices.testConnection') }}</el-button>
          <el-button type="danger" size="small" link data-test="device-delete-btn" @click="handleDelete(row)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
      <template #empty>
        <span>{{ loading ? t('devices.loadingEllipsis') : t('devices.emptyNone') }}</span>
      </template>
    </el-table>

    <div class="pagination-wrapper">
      <el-pagination v-model:current-page="currentPage" v-model:page-size="pageSize" :page-sizes="[10, 20, 50]"
        :total="filteredRows.length" layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange" @current-change="handleCurrentChange" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { Plus, Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import { useDeviceStore } from '../stores/device'
import { addDevice, getFleetReconcile, removeDevice } from '../api'
import { deriveDeviceRows, type DeviceRow } from '../utils/deviceRows'
import { validateDeviceForm, toAddDevicePayload } from '../utils/deviceForm'
import type { FleetInput } from '../composables/useFleetOverview'
import ReconcileChip from '../components/dashboard/ReconcileChip.vue'
import Sparkline from '../components/common/Sparkline.vue'

const router = useRouter()
const { t } = useI18n()
const store = useDeviceStore()

const searchKeyword = ref('')
const statusFilter = ref('')
const vendorFilter = ref('')
const currentPage = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const fleet = ref<FleetInput>({}) // /reconcile/status 聚合，join 出收敛态

// 设备事实 + 会话态 + 对账真数据（离线优先）。
const rows = computed<DeviceRow[]>(() => deriveDeviceRows(store.devices, fleet.value))
const vendors = computed(() => [...new Set(rows.value.map((r) => r.vendor).filter(Boolean))].sort())

// 筛选/搜索变化回到第一页，避免停在越界空页（filteredRows 有数据但 paginatedRows 空）
watch([searchKeyword, statusFilter, vendorFilter], () => {
  currentPage.value = 1
})

const filteredRows = computed(() => {
  let result = rows.value
  if (searchKeyword.value) {
    const kw = searchKeyword.value.toLowerCase()
    result = result.filter((r) => r.ip.toLowerCase().includes(kw) || r.name.toLowerCase().includes(kw))
  }
  if (statusFilter.value) result = result.filter((r) => (statusFilter.value === 'online' ? r.online : !r.online))
  if (vendorFilter.value) result = result.filter((r) => r.vendor === vendorFilter.value)
  return result
})

// 修既有缺陷：表格此前绑 filteredDevices 未真正分页，改绑 paginatedRows。
const paginatedRows = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredRows.value.slice(start, start + pageSize.value)
})

async function load() {
  loading.value = true
  try {
    // 设备与对账聚合并行；对账失败不阻断设备表（收敛态降级为 unknown/off）
    const [, fleetRes] = await Promise.allSettled([store.fetchDevices(), getFleetReconcile()])
    fleet.value = fleetRes.status === 'fulfilled' ? (fleetRes.value.data?.data ?? {}) : {}
  } finally {
    loading.value = false
  }
}

function handleRefresh() {
  currentPage.value = 1
  load()
}

function goToConfig(row: DeviceRow) {
  // 旧配置页路由（name:'interface'）已随 FE-13 退役，跳通用模块控制台；
  // device 传 IP，与控制台设备下拉的 value 口径一致。
  // 同步写全局设备上下文（FE-10）：选一次设备，后续模块切换沿用；
  // query 仍携带以保证 URL 可分享（双写幂等）。
  store.selectDevice(row.ip)
  router.push({ name: 'module-console', params: { module: 'ifm' }, query: { device: row.ip } })
}

// ── 添加/删除设备（后端 POST/DELETE /devices 早已就绪，此处补前端入口）──

const addDialogVisible = ref(false)
const adding = ref(false)
const addFormRef = ref<FormInstance>()
const addForm = reactive({ ip: '', port: '', username: '', password: '', vendor: '', role: '' })

const ipv4Re = /^(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}$/
const addRules: FormRules = {
  ip: [
    { required: true, message: () => t('devices.ruleIpRequired'), trigger: 'blur' },
    {
      validator: (_r, v, cb) => (v && !ipv4Re.test(v) ? cb(new Error(t('devices.ruleIpInvalid'))) : cb()),
      trigger: ['blur', 'change'],
    },
  ],
  port: [
    {
      validator: (_r, v, cb) => {
        if (!v) return cb()
        const n = Number(v)
        return Number.isInteger(n) && n >= 1 && n <= 65535 ? cb() : cb(new Error(t('devices.rulePortInvalid')))
      },
      trigger: ['blur', 'change'],
    },
  ],
  username: [{ required: true, message: () => t('devices.ruleUserRequired'), trigger: 'blur' }],
  password: [{ required: true, message: () => t('devices.rulePassRequired'), trigger: 'blur' }],
  // BR-14 同口径：≤32 位 [A-Za-z0-9_-]
  role: [
    {
      validator: (_r, v, cb) => (v && !/^[A-Za-z0-9_-]{1,32}$/.test(v) ? cb(new Error(t('devices.ruleRoleInvalid'))) : cb()),
      trigger: ['blur', 'change'],
    },
  ],
}

function openAddDialog() {
  Object.assign(addForm, { ip: '', port: '', username: '', password: '', vendor: '', role: '' })
  addFormRef.value?.clearValidate()
  addDialogVisible.value = true
}

async function submitAddDevice() {
  // 闸门用纯函数（与渲染无关，见 utils/deviceForm 注释）；el-form 的 rules 只
  // 负责行内提示，故这里额外触发一次 validate() 把错误渲染出来（结果不作判据）。
  const errors = validateDeviceForm(addForm)
  addFormRef.value?.validate().catch(() => undefined)
  if (Object.keys(errors).length > 0) {
    const firstKey = Object.values(errors)[0]
    if (firstKey) ElMessage.error(t(firstKey))
    return
  }
  adding.value = true
  try {
    await addDevice(toAddDevicePayload(addForm))
    ElMessage.success(t('devices.addSuccess', { ip: addForm.ip.trim() }))
    addDialogVisible.value = false
    await load()
  } catch (e: any) {
    // 后端拒绝（连不上/参数不合规）：不关框，展示原因供改后重试
    ElMessage.error(t('devices.addFailed', { reason: e?.response?.data?.message || e?.message || e }))
  } finally {
    adding.value = false
  }
}

async function handleDelete(row: DeviceRow) {
  try {
    await ElMessageBox.confirm(t('devices.deleteConfirm', { ip: row.ip }), t('common.delete'), { type: 'warning' })
  } catch {
    return // 用户取消
  }
  try {
    await removeDevice(row.ip)
    ElMessage.success(t('devices.deleteSuccess', { ip: row.ip }))
    await load()
  } catch (e: any) {
    ElMessage.error(t('devices.deleteFailed', { reason: e?.response?.data?.message || e?.message || e }))
  }
}

async function handleTestConnection(row: DeviceRow) {
  const result = await store.testConnection(row.id)
  if (result.success) ElMessage.success(t('devices.connTestSuccess', { name: row.name || row.ip }))
  else ElMessage.error(`${row.name || row.ip} ${result.message}`)
}

function handleSizeChange(size: number) {
  pageSize.value = size
  currentPage.value = 1
}

function handleCurrentChange(page: number) {
  currentPage.value = page
}

onMounted(load)
</script>

<style scoped>
.devices {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  flex-wrap: wrap;
}

.page-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--ink, #1f2d3d);
}

.page-header .sub {
  margin-top: 4px;
  font-size: 12.5px;
  color: var(--ink-3, #93a2b1);
}

.header-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
  align-items: center;
}

.search-input {
  width: 240px;
}

.filter-select {
  width: 140px;
}

.device-table {
  background: var(--bg-card, #fff);
  border-radius: var(--r-card, 12px);
}

.mono {
  font-family: var(--f-mono, monospace);
}

.strong {
  font-weight: 600;
  color: var(--ink, #1f2d3d);
}

.dim {
  color: var(--ink-2, #52627a);
  font-size: 12.5px;
}

.mono.dim {
  color: var(--ink-3, #93a2b1);
  font-size: 12px;
}

/* 会话 chip（已连接/断开）——与对账 chip 同基座，配色走令牌 */
.chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 23px;
  padding: 0 9px 0 8px;
  border-radius: var(--r-chip, 999px);
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.chip .glyph {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
  background: currentColor;
}

.chip.conv {
  background: var(--st-conv-bg, #e4f2e8);
  color: var(--st-conv, #2f8a4c);
}

.chip.off {
  background: var(--st-off-bg, #fbe6e3);
  color: var(--st-off, #c0392b);
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 8px 0;
}

@media (max-width: 768px) {
  .search-input,
  .filter-select {
    width: 100%;
    flex: 1;
    min-width: 120px;
  }
}
</style>
