<template>
  <div class="logs">
    <div class="page-header">
      <h2>操作日志</h2>
      <el-button type="primary" :icon="Download" @click="handleExport">
        导出
      </el-button>
    </div>

    <div class="filter-bar">
      <el-input
        v-model="searchKeyword"
        placeholder="搜索设备或操作类型"
        :prefix-icon="Search"
        clearable
        class="search-input"
        @keyup.enter="fetchLogs"
      />
      <el-date-picker
        v-model="dateRange"
        type="daterange"
        range-separator="至"
        start-placeholder="开始日期"
        end-placeholder="结束日期"
        class="date-picker"
        @change="fetchLogs"
      />
      <el-select v-model="statusFilter" placeholder="状态筛选" clearable class="filter-select" @change="fetchLogs">
        <el-option label="全部" value="" />
        <el-option label="成功" value="success" />
        <el-option label="失败" value="failed" />
      </el-select>
    </div>

    <el-table :data="logs" stripe class="logs-table" v-loading="loading">
      <el-table-column prop="time" label="时间" width="180" />
      <el-table-column prop="device" label="设备" width="140" />
      <el-table-column prop="type" label="操作类型" width="140" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'success' ? 'success' : 'danger'" size="small">
            {{ row.status === 'success' ? '成功' : '失败' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="user" label="操作人" width="120" />
      <el-table-column prop="detail" label="详情" min-width="200" show-overflow-tooltip />
    </el-table>

    <div class="pagination-wrapper">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[20, 50, 100]"
        :total="total"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="fetchLogs"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Download, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getLogs, type Log } from '../api/logs'

const logs = ref<Log[]>([])
const loading = ref(false)
const searchKeyword = ref('')
const dateRange = ref<[string, string] | null>(null)
const statusFilter = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)

async function fetchLogs() {
  loading.value = true
  try {
    const params: any = {
      page: currentPage.value,
      pageSize: pageSize.value
    }
    if (searchKeyword.value) {
      params.keyword = searchKeyword.value
    }
    if (dateRange.value && dateRange.value.length === 2) {
      params.startTime = dateRange.value[0]
      params.endTime = dateRange.value[1]
    }
    if (statusFilter.value) {
      params.status = statusFilter.value
    }

    const result = await getLogs(params)
    logs.value = result.data
    total.value = result.total
  } catch (err) {
    ElMessage.error('获取日志失败')
  } finally {
    loading.value = false
  }
}

function handleSizeChange(size: number) {
  pageSize.value = size
  currentPage.value = 1
  fetchLogs()
}

function handleExport() {
  ElMessage.info('导出功能开发中')
}

onMounted(() => {
  fetchLogs()
})
</script>

<style scoped>
.logs {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-header h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}

.filter-bar {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.search-input {
  width: 300px;
}

.date-picker {
  width: 360px;
}

.filter-select {
  width: 150px;
}

.logs-table {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.08);
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 0;
}

@media (max-width: 768px) {
  .search-input,
  .date-picker,
  .filter-select {
    width: 100%;
  }
}
</style>
