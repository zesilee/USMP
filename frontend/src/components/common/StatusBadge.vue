<template>
  <div :class="['status-badge', `status-${phase.toLowerCase()}`]">
    <el-icon v-if="phase === 'Updating'" class="is-loading">
      <Loading />
    </el-icon>
    <span class="status-text">{{ statusText }}</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Loading } from '@element-plus/icons-vue'
import type { ConfigPhase } from '../../composables/useK8sCRD'

const props = defineProps<{
  phase: ConfigPhase
}>()

const statusText = computed(() => {
  const map: Record<ConfigPhase, string> = {
    Pending: '待同步',
    Updating: '同步中',
    Ready: '已同步',
    Failed: '同步失败'
  }
  return map[props.phase]
})
</script>

<style scoped>
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border-radius: 16px;
  font-size: 12px;
  font-weight: 500;
}

.status-pending {
  background-color: #f4f4f5;
  color: #909399;
}

.status-updating {
  background-color: #ecf5ff;
  color: #409eff;
}

.status-updating .el-icon {
  animation: rotate 1s linear infinite;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.status-ready {
  background-color: #f0f9eb;
  color: #67c23a;
}

.status-failed {
  background-color: #fef0f0;
  color: #f56c6c;
}

.status-text {
  line-height: 1;
}
</style>
