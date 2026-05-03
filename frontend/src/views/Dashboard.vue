<template>
  <div class="dashboard">
    <div class="stats-row">
      <StatCard
        title="设备总数"
        :value="28"
        :icon="Monitor"
        :trend="1"
        trend-label="较昨日"
      />
      <StatCard
        title="在线设备"
        :value="25"
        :icon="SuccessFilled"
        icon-color="#67c23a"
        icon-bg="#f0f9eb"
        :trend="2"
        trend-label="较昨日"
      />
      <StatCard
        title="配置同步率"
        :value="'98%'"
        :icon="CircleCheck"
        icon-color="#409eff"
        icon-bg="#ecf5ff"
      />
      <StatCard
        title="今日操作"
        :value="15"
        :icon="Document"
        icon-color="#e6a23c"
        icon-bg="#fdf6ec"
        :trend="5"
        trend-label="较昨日"
      />
    </div>

    <div class="chart-row">
      <StatusChart :online="10" :offline="2" :abnormal="0" />
    </div>

    <div class="logs-card">
      <div class="card-header">
        <h3>最近操作日志</h3>
      </div>
      <el-table :data="mockLogs" stripe>
        <el-table-column prop="time" label="时间" width="180" />
        <el-table-column prop="device" label="设备" width="200" />
        <el-table-column prop="type" label="操作类型" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === '成功' ? 'success' : 'danger'" size="small">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="operator" label="操作人" width="120" />
      </el-table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Monitor, SuccessFilled, CircleCheck, Document } from '@element-plus/icons-vue'
import StatCard from '../components/dashboard/StatCard.vue'
import StatusChart from '../components/dashboard/StatusChart.vue'

const mockLogs = [
  {
    time: '2026-05-03 14:30:25',
    device: 'Core-Switch-01',
    type: '配置下发',
    status: '成功',
    operator: 'admin'
  },
  {
    time: '2026-05-03 13:15:42',
    device: 'Access-Switch-03',
    type: 'VLAN配置',
    status: '成功',
    operator: 'admin'
  },
  {
    time: '2026-05-03 11:45:18',
    device: 'Core-Switch-02',
    type: '接口配置',
    status: '成功',
    operator: 'user01'
  },
  {
    time: '2026-05-03 10:20:55',
    device: 'Access-Switch-01',
    type: '配置同步',
    status: '失败',
    operator: 'admin'
  },
  {
    time: '2026-05-03 09:05:33',
    device: 'Access-Switch-02',
    type: '路由配置',
    status: '成功',
    operator: 'user01'
  }
]
</script>

<style scoped>
.dashboard {
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
}

.chart-row {
  display: flex;
  gap: 20px;
}

.chart-row > :first-child {
  flex: 1;
}

.logs-card {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.08);
  padding: 20px;
}

.card-header {
  margin-bottom: 20px;
}

.card-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin: 0;
}

@media (max-width: 1200px) {
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 768px) {
  .stats-row {
    grid-template-columns: 1fr;
  }
}
</style>
