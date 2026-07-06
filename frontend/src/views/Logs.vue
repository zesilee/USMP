<template>
  <div class="logs">
    <div class="page-header">
      <div>
        <h2>操作日志</h2>
        <div class="sub">每条下发的对账结局都可追溯 · 保留于本地 JSON 元信息（§8）</div>
      </div>
      <div class="header-actions">
        <el-input v-model="searchKeyword" placeholder="按设备 / 操作人搜索" :prefix-icon="Search" clearable class="search-input" />
        <el-select v-model="statusFilter" placeholder="全部结果" clearable class="filter-select">
          <el-option v-for="o in statusOptions" :key="o.value" :label="o.label" :value="o.value" />
        </el-select>
        <el-button :icon="Refresh" @click="load" :loading="loading">刷新</el-button>
      </div>
    </div>

    <el-table :data="paginatedRows" class="logs-table" v-loading="loading">
      <el-table-column label="时间" width="180">
        <template #default="{ row }"><span class="mono dim">{{ formatTime(row.timestamp) }}</span></template>
      </el-table-column>
      <el-table-column label="操作" width="170">
        <template #default="{ row }">
          <div class="log-op">
            <div class="op-ico">
              <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M4 6h16M4 12h16M4 18h16" /></svg>
            </div>
            {{ row.opLabel }}
          </div>
        </template>
      </el-table-column>
      <el-table-column label="设备" width="150">
        <template #default="{ row }"><span class="strong">{{ row.device || '—' }}</span></template>
      </el-table-column>
      <el-table-column label="变更" min-width="160">
        <template #default="{ row }"><span class="mono change">{{ row.summary || '—' }}</span></template>
      </el-table-column>
      <el-table-column label="操作人" width="120">
        <template #default="{ row }"><span class="dim">{{ row.actor || '—' }}</span></template>
      </el-table-column>
      <el-table-column label="对账结局" width="130">
        <template #default="{ row }"><ReconcileChip :state="row.reconcileState" /></template>
      </el-table-column>
      <template #empty>
        <span>{{ loading ? '加载中…' : '暂无操作日志' }}</span>
      </template>
    </el-table>

    <div class="footnote">
      变更列为下发内容摘要（值级 was→now 差异待后端记录）· 操作人为 system（后端无鉴权用户上下文）· 对账结局为查询时实时态。
    </div>

    <div class="pagination-wrapper">
      <el-pagination v-model:current-page="currentPage" v-model:page-size="pageSize" :page-sizes="[20, 50, 100]"
        :total="filteredRows.length" layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange" @current-change="handleCurrentChange" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { Refresh, Search } from '@element-plus/icons-vue'
import { getLogs } from '../api'
import { deriveLogRows, type LogRow } from '../utils/logRows'
import type { DisplayState } from '../composables/useFleetOverview'
import ReconcileChip from '../components/dashboard/ReconcileChip.vue'

const statusOptions: { label: string; value: DisplayState }[] = [
  { label: '已收敛', value: 'conv' },
  { label: '收敛中', value: 'recon' },
  { label: '已漂移', value: 'drift' },
  { label: '下发失败', value: 'error' },
  { label: '未对账', value: 'unknown' },
]

const rows = ref<LogRow[]>([])
const loading = ref(false)
const searchKeyword = ref('')
const statusFilter = ref<DisplayState | ''>('')
const currentPage = ref(1)
const pageSize = ref(20)

// 一次拉一批后客户端筛选/分页（与设备页一致）。后端 /logs 单批上限 500 条
// （maxLogLimit）；审计量超 500 时最旧记录在此不可达（低频，可接受）。
async function load() {
  loading.value = true
  try {
    const res = await getLogs({ limit: 500 })
    rows.value = deriveLogRows(res.data?.data?.logs ?? [])
  } catch {
    rows.value = [] // 拉取失败降级空表（R08）
  } finally {
    loading.value = false
  }
}

const filteredRows = computed(() => {
  let result = rows.value
  if (searchKeyword.value) {
    const kw = searchKeyword.value.toLowerCase()
    result = result.filter((r) => r.device.toLowerCase().includes(kw) || r.actor.toLowerCase().includes(kw))
  }
  if (statusFilter.value) result = result.filter((r) => r.reconcileState === statusFilter.value)
  return result
})

const paginatedRows = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredRows.value.slice(start, start + pageSize.value)
})

watch([searchKeyword, statusFilter], () => {
  currentPage.value = 1
})

function formatTime(iso: string): string {
  if (!iso) return '—'
  const t = new Date(iso)
  return isNaN(t.getTime()) ? iso : t.toLocaleString('zh-CN', { hour12: false })
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
.logs {
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

.logs-table {
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

.change {
  font-size: 12.5px;
  color: var(--ink, #1f2d3d);
}

.log-op {
  display: flex;
  align-items: center;
  gap: 10px;
}

.op-ico {
  width: 30px;
  height: 30px;
  border-radius: var(--r-ctl, 8px);
  background: var(--sunken, #f4f6f9);
  display: grid;
  place-items: center;
  flex-shrink: 0;
  color: var(--ink-2, #52627a);
}

.op-ico svg {
  width: 15px;
  height: 15px;
  stroke: currentColor;
  fill: none;
  stroke-width: 1.7;
}

.footnote {
  font-size: 11.5px;
  line-height: 1.6;
  color: var(--ink-3, #93a2b1);
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
