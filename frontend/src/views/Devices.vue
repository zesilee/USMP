<template>
  <div class="devices">
    <div class="page-header">
      <div>
        <h2>设备管理</h2>
        <div class="sub">直连 NETCONF (SSH 830) / gNMI · 会话状态实时刷新</div>
      </div>
      <div class="header-actions">
        <el-input v-model="searchKeyword" placeholder="按 IP 或名称搜索" :prefix-icon="Search" clearable class="search-input" />
        <el-select v-model="statusFilter" placeholder="全部状态" clearable class="filter-select">
          <el-option label="在线" value="online" />
          <el-option label="离线" value="offline" />
        </el-select>
        <el-select v-model="vendorFilter" placeholder="全部厂商" clearable class="filter-select">
          <el-option v-for="v in vendors" :key="v" :label="v" :value="v" />
        </el-select>
        <el-button :icon="Refresh" @click="handleRefresh" :loading="loading">刷新</el-button>
      </div>
    </div>

    <el-table :data="paginatedRows" class="device-table" v-loading="loading">
      <el-table-column label="IP 地址" width="150">
        <template #default="{ row }"><span class="mono strong">{{ row.ip }}</span></template>
      </el-table-column>
      <el-table-column label="名称" width="170">
        <template #default="{ row }"><span class="strong">{{ row.name || '—' }}</span></template>
      </el-table-column>
      <el-table-column label="厂商 / 型号" min-width="150">
        <template #default="{ row }"><span class="dim">{{ row.vendorModel || '—' }}</span></template>
      </el-table-column>
      <el-table-column label="会话" width="120">
        <template #default="{ row }">
          <span class="chip" :class="row.session === 'connected' ? 'conv' : 'off'">
            <span class="glyph" aria-hidden="true"></span>{{ row.session === 'connected' ? '已连接' : '断开' }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="负载 (1h)" width="110">
        <template #default="{ row }"><Sparkline :points="row.load" /></template>
      </el-table-column>
      <el-table-column label="收敛态" width="120">
        <template #default="{ row }"><ReconcileChip :state="row.reconcileState" /></template>
      </el-table-column>
      <el-table-column label="最后同步" min-width="140">
        <template #default="{ row }"><span class="mono dim">{{ row.lastSync || '—' }}</span></template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" size="small" link @click="goToConfig(row)">查看配置</el-button>
          <el-button type="info" size="small" link @click="handleTestConnection(row)">连接测试</el-button>
        </template>
      </el-table-column>
      <template #empty>
        <span>{{ loading ? '加载中…' : '暂无设备' }}</span>
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
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useDeviceStore } from '../stores/device'
import { getFleetReconcile } from '../api'
import { deriveDeviceRows, type DeviceRow } from '../utils/deviceRows'
import type { FleetInput } from '../composables/useFleetOverview'
import ReconcileChip from '../components/dashboard/ReconcileChip.vue'
import Sparkline from '../components/common/Sparkline.vue'

const router = useRouter()
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
  router.push({ name: 'interface', query: { device: row.id } })
}

async function handleTestConnection(row: DeviceRow) {
  const result = await store.testConnection(row.id)
  if (result.success) ElMessage.success(`${row.name || row.ip} 连接测试成功`)
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
