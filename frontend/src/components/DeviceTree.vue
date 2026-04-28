<template>
  <div class="device-tree">
    <el-tree
      :data="treeData"
      :props="props"
      @node-click="handleNodeClick"
      default-expand-all
      :expand-on-click-node="false"
      node-key="key"
      :highlight-current="true"
    >
      <template #default="{ node, data }">
        <div class="tree-node">
          <span v-if="data.device && !data.yangPath" class="node-status status-dot status-dot--success"></span>
          <span v-if="data.yangPath" class="node-module"></span>
          <span class="node-label">{{ node.label }}</span>
        </div>
      </template>
    </el-tree>
    <div class="device-actions">
      <el-button size="small" @click="refreshDevices" class="refresh-btn">
        刷新设备
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { listDevices } from '../api'
import { DeviceInfo } from '../types/yang'

const props = {
  label: 'label',
  children: 'children',
}

interface TreeNode {
  label: string
  key: string
  device?: DeviceInfo
  yangPath?: string
  yangName?: string
  children?: TreeNode[]
}

const devices = ref<DeviceInfo[]>([])
const treeData = computed(() => {
  return devices.value.map(device => ({
    label: device.ip,
    key: `device-${device.ip}`,
    device: device,
    children: [
      {
        label: 'Interfaces',
        key: `${device.ip}-/interfaces`,
        device: device,
        yangPath: '/interfaces',
        yangName: 'Interfaces',
      },
      {
        label: 'VLANs',
        key: `${device.ip}-/vlans`,
        device: device,
        yangPath: '/vlans',
        yangName: 'VLANs',
      },
      {
        label: 'System',
        key: `${device.ip}-/system`,
        device: device,
        yangPath: '/system',
        yangName: 'System',
      },
    ],
  }))
})

const emit = defineEmits<{
  'node-selected': [device: DeviceInfo, yangPath: string, yangName: string]
}>()

const handleNodeClick = (node: TreeNode) => {
  if (node.yangPath && node.device) {
    emit('node-selected', node.device, node.yangPath, node.yangName || node.label)
  }
}

const refreshDevices = async () => {
  try {
    const res = await listDevices()
    if (res.data.success) {
      devices.value = res.data.data.devices || []
    }
  } catch (err) {
    console.error('Failed to refresh devices', err)
  }
}

onMounted(() => {
  refreshDevices()
})
</script>

<style lang="scss" scoped>
.device-tree {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.node-status {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
  background-color: var(--color-success);
  box-shadow: 0 0 0 3px var(--color-success-bg);
}

.node-module {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
  background-color: var(--text-tertiary);
  opacity: 0.5;
}

.node-label {
  font-size: var(--font-size-sm);
  font-weight: var(--font-weight-medium);
  color: var(--text-primary);
  white-space: nowrap;
}

:deep(.el-tree-node__content) {
  white-space: nowrap;
}

.device-actions {
  padding: var(--spacing-lg) var(--spacing-xl);
  border-top: 1px solid var(--border-color);
  margin-top: auto;

  .refresh-btn {
    width: 100%;
  }
}
</style>
