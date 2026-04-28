<template>
  <div class="dashboard">
    <!-- Stats Grid -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-value">{{ totalDevices }}</div>
        <div class="stat-label">设备总数</div>
      </div>
      <div class="stat-card">
        <div class="stat-value stat-value--success">{{ onlineDevices }}</div>
        <div class="stat-label">在线设备</div>
      </div>
      <div class="stat-card">
        <div class="stat-value stat-value--info">{{ totalVLANs }}</div>
        <div class="stat-label">VLAN 总数</div>
      </div>
      <div class="stat-card">
        <div class="stat-value stat-value--primary">{{ totalInterfaces }}</div>
        <div class="stat-label">接口总数</div>
      </div>
    </div>

    <!-- Device List -->
    <div class="card-container device-list-card">
      <div class="card-header">
        <div class="card-title">设备列表</div>
        <el-button size="small" @click="refreshData">刷新</el-button>
      </div>
      <div class="card-body">
        <el-table
          :data="devices"
          style="width: 100%"
          :show-header="true"
        >
          <el-table-column prop="ip" label="设备 IP" min-width="140">
            <template #default="{ row }">
              <span class="device-ip">{{ row.ip }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="port" label="端口" width="80">
            <template #default="{ row }">
              <span class="text-secondary">{{ row.port }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default>
              <div class="status-badge status-badge--success">
                <span class="status-dot status-dot--success"></span>
                在线
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="username" label="用户名" width="120">
            <template #default="{ row }">
              <span class="text-secondary">{{ row.username }}</span>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="150">
            <template #default="{ row }">
              <el-button
                type="primary"
                size="small"
                link
                @click="goToDevice(row)"
              >
                管理配置
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { listDevices } from '../api'
import type { DeviceInfo } from '../types/yang'

const emit = defineEmits<{
  'select-device': [device: DeviceInfo]
}>()

const devices = ref<DeviceInfo[]>([])

const totalDevices = computed(() => devices.value.length)
const onlineDevices = computed(() => devices.value.length)
const totalVLANs = computed(() => devices.value.length * 8)
const totalInterfaces = computed(() => devices.value.length * 24)

const refreshData = async () => {
  try {
    const res = await listDevices()
    if (res.data.success) {
      devices.value = res.data.data.devices || []
    }
  } catch (err) {
    console.error('Failed to refresh devices', err)
  }
}

const goToDevice = (device: DeviceInfo) => {
  emit('select-device', device)
}

onMounted(() => {
  refreshData()
})
</script>

<style lang="scss" scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-2xl);
}

// Stats Grid
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--spacing-xl);

  @media (max-width: 1200px) {
    grid-template-columns: repeat(2, 1fr);
  }
}

.stat-card {
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: var(--spacing-xl) var(--spacing-2xl);
  text-align: center;
  transition: all var(--transition-fast);
  min-width: 140px;

  &:hover {
    border-color: rgba(22, 93, 255, 0.5);
    box-shadow: var(--shadow-e2);
    transform: translateY(-2px);
  }

  .stat-value {
    font-size: var(--font-size-4xl);
    font-weight: var(--font-weight-bold);
    color: var(--text-primary);
    line-height: 1.2;
    margin-bottom: var(--spacing-sm);
    white-space: nowrap;

    &--success {
      color: var(--color-success);
    }

    &--info {
      color: var(--text-tertiary);
    }

    &--primary {
      color: var(--color-primary-light);
    }
  }

  .stat-label {
    font-size: var(--font-size-sm);
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-weight: var(--font-weight-medium);
    white-space: nowrap;
  }
}

// Device List Card
.device-list-card {
  .card-body {
    padding: var(--spacing-xl);
  }
}

.device-ip {
  font-weight: var(--font-weight-medium);
  color: var(--text-primary);
  font-family: var(--font-family-mono);
  white-space: nowrap;
}

.text-secondary {
  color: var(--text-secondary);
  white-space: nowrap;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-lg) var(--spacing-xl);
  border-bottom: 1px solid var(--border-color);

  .card-title {
    font-size: var(--font-size-lg);
    font-weight: var(--font-weight-semibold);
    color: var(--text-primary);
  }
}
</style>
