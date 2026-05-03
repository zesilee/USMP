<template>
  <div class="devices">
    <div class="page-header">
      <h2>设备管理</h2>
      <el-button type="primary" :icon="Refresh" @click="handleRefresh" :loading="store.isLoading">
        刷新
      </el-button>
    </div>

    <div class="filter-bar">
      <el-input
        v-model="searchKeyword"
        placeholder="按 IP 或名称搜索"
        :prefix-icon="Search"
        clearable
        class="search-input"
      />
      <el-select v-model="statusFilter" placeholder="状态筛选" clearable class="filter-select">
        <el-option label="在线" value="online" />
        <el-option label="离线" value="offline" />
      </el-select>
      <el-select v-model="vendorFilter" placeholder="厂商筛选" clearable class="filter-select">
        <el-option label="H3C" value="H3C" />
        <el-option label="Huawei" value="Huawei" />
        <el-option label="Cisco" value="Cisco" />
      </el-select>
    </div>

    <el-table :data="filteredDevices" stripe class="device-table" v-loading="store.isLoading">
      <el-table-column prop="ip" label="IP" width="140" />
      <el-table-column prop="name" label="名称" width="180" />
      <el-table-column prop="vendor" label="厂商" width="100" />
      <el-table-column prop="model" label="型号" width="120" />
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="row.status === 'online' ? 'success' : 'danger'" size="small">
            {{ row.status === 'online' ? '在线' : '离线' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="lastSync" label="最后同步时间" min-width="180" />
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" size="small" link @click="goToConfig(row)">
            查看配置
          </el-button>
          <el-button type="info" size="small" link @click="handleTestConnection(row)">
            连接测试
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pagination-wrapper">
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50]"
        :total="filteredDevices.length"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useDeviceStore } from '../stores/device'
import { Refresh, Search } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'

const router = useRouter()
const store = useDeviceStore()

const searchKeyword = ref('')
const statusFilter = ref('')
const vendorFilter = ref('')
const currentPage = ref(1)
const pageSize = ref(10)

const filteredDevices = computed(() => {
  let result = [...store.devices]

  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    result = result.filter(d =>
      d.ip.toLowerCase().includes(keyword) ||
      d.name.toLowerCase().includes(keyword)
    )
  }

  if (statusFilter.value) {
    result = result.filter(d => d.status === statusFilter.value)
  }

  if (vendorFilter.value) {
    result = result.filter(d => d.vendor === vendorFilter.value)
  }

  return result
})

const paginatedDevices = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredDevices.value.slice(start, start + pageSize.value)
})

function handleRefresh() {
  store.fetchDevices()
}

function goToConfig(device: any) {
  router.push({ name: 'interface', query: { device: device.id } })
}

async function handleTestConnection(device: any) {
  const result = await store.testConnection(device.id)
  if (result.success) {
    ElMessage.success(`${device.name} 连接测试成功`)
  } else {
    ElMessage.error(`${device.name} ${result.message}`)
  }
}

function handleSizeChange(size: number) {
  pageSize.value = size
  currentPage.value = 1
}

function handleCurrentChange(page: number) {
  currentPage.value = page
}

onMounted(() => {
  store.fetchDevices()
})
</script>

<style scoped>
.devices {
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

.filter-select {
  width: 150px;
}

.device-table {
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
  .search-input {
    width: 100%;
  }

  .filter-select {
    flex: 1;
    min-width: 120px;
  }
}
</style>
