<template>
  <div class="device-tree">
    <el-tree
      :data="treeData"
      :props="props"
      @node-click="handleNodeClick"
      default-expand-all
    />
    <div class="device-actions">
      <el-button type="primary" size="small" @click="refreshDevices">
        刷新
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
      devices.value = res.data.data || []
    }
  } catch (err) {
    console.error('Failed to refresh devices', err)
  }
}

onMounted(() => {
  refreshDevices()
})
</script>

<style scoped>
.device-tree {
  padding: 10px 0;
}

.device-actions {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid #dcdfe6;
  text-align: center;
}
</style>
