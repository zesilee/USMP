<template>
  <div class="vlan-toolbar">
    <div class="toolbar-left">
      <el-button
        type="primary"
        size="default"
        @click="$emit('add')"
      >
        <el-icon><Plus /></el-icon>
        新建 VLAN
      </el-button>

      <el-button
        size="default"
        @click="$emit('refresh')"
        :loading="loading"
      >
        <el-icon><Refresh /></el-icon>
        刷新
      </el-button>

      <el-button
        v-if="selectedCount > 0"
        type="danger"
        size="default"
        @click="$emit('batch-delete')"
      >
        <el-icon><Delete /></el-icon>
        删除选中 ({{ selectedCount }})
      </el-button>
    </div>

    <div class="toolbar-right">
      <el-input
        v-model="searchText"
        placeholder="搜索 VLAN ID 或名称..."
        size="default"
        clearable
        style="width: 240px"
      >
        <template #prefix>
          <el-icon><Search /></el-icon>
        </template>
      </el-input>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Plus, Refresh, Delete, Search } from '@element-plus/icons-vue'

interface Props {
  selectedCount: number
  loading: boolean
}

defineProps<Props>()

defineEmits<{
  'add': []
  'refresh': []
  'batch-delete': []
}>()

const searchText = ref('')
</script>

<style lang="scss" scoped>
@import '../../styles/variables.scss';

.vlan-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-lg $spacing-xl;
  border-bottom: 1px solid $border-color;
  background-color: rgba(255, 255, 255, 0.02);

  .toolbar-left {
    display: flex;
    gap: $spacing-md;
  }

  .toolbar-right {
    display: flex;
    align-items: center;
    gap: $spacing-md;
  }
}
</style>
