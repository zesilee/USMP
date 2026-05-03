<template>
  <div class="status-chart-card">
    <div class="card-header">
      <h3>设备状态分布</h3>
    </div>
    <div class="chart-content">
      <div ref="chartRef" class="status-chart-container"></div>
      <div class="legend-area">
        <div class="legend-item">
          <span class="legend-dot online"></span>
          <span class="legend-label">在线</span>
          <span class="legend-value">{{ online }}</span>
        </div>
        <div class="legend-item">
          <span class="legend-dot offline"></span>
          <span class="legend-label">离线</span>
          <span class="legend-value">{{ offline }}</span>
        </div>
        <div class="legend-item">
          <span class="legend-dot abnormal"></span>
          <span class="legend-label">异常</span>
          <span class="legend-value">{{ abnormal }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import * as echarts from 'echarts'

interface Props {
  online: number
  offline: number
  abnormal: number
}

const props = defineProps<Props>()

const chartRef = ref<HTMLElement | null>(null)
let chartInstance: echarts.ECharts | null = null

const initChart = () => {
  if (!chartRef.value) return

  chartInstance = echarts.init(chartRef.value)
  updateChart()
}

const updateChart = () => {
  if (!chartInstance) return

  const option: echarts.EChartsOption = {
    series: [
      {
        type: 'pie',
        radius: ['60%', '80%'],
        center: ['50%', '50%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 4,
          borderColor: '#fff',
          borderWidth: 2
        },
        label: {
          show: false
        },
        emphasis: {
          label: {
            show: false
          }
        },
        data: [
          { value: props.online, name: '在线', itemStyle: { color: '#67c23a' } },
          { value: props.offline, name: '离线', itemStyle: { color: '#909399' } },
          { value: props.abnormal, name: '异常', itemStyle: { color: '#f56c6c' } }
        ]
      }
    ]
  }

  chartInstance.setOption(option)
}

const handleResize = () => {
  chartInstance?.resize()
}

watch(() => [props.online, props.offline, props.abnormal], () => {
  updateChart()
})

onMounted(() => {
  initChart()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  chartInstance?.dispose()
})
</script>

<style scoped>
.status-chart-card {
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

.chart-content {
  display: flex;
  align-items: center;
  gap: 30px;
}

.status-chart-container {
  width: 200px;
  height: 200px;
  flex-shrink: 0;
}

.legend-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 12px;
}

.legend-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  flex-shrink: 0;
}

.legend-dot.online {
  background: #67c23a;
}

.legend-dot.offline {
  background: #909399;
}

.legend-dot.abnormal {
  background: #f56c6c;
}

.legend-label {
  font-size: 14px;
  color: #606266;
  flex: 1;
}

.legend-value {
  font-size: 20px;
  font-weight: 600;
  color: #303133;
}
</style>
