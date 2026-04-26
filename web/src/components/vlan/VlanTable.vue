<template>
  <div class="vlan-table">
    <el-table
      ref="tableRef"
      :data="vlans"
      v-loading="loading"
      @selection-change="handleSelectionChange"
      style="width: 100%"
      stripe
    >
      <el-table-column type="selection" width="55" align="center" />

      <el-table-column prop="id" label="VLAN ID" width="100" sortable align="center">
        <template #default="{ row }">
          <span class="vlan-id">{{ row.id }}</span>
        </template>
      </el-table-column>

      <el-table-column prop="name" label="名称" min-width="140">
        <template #default="{ row }">
          <div class="vlan-name">
            <span>{{ row.name || '-' }}</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="运行状态" width="110" align="center">
        <template #default="{ row }">
          <StatusBadge type="oper" :value="row.operStatus" />
        </template>
      </el-table-column>

      <el-table-column label="管理状态" width="100" align="center">
        <template #default="{ row }">
          <StatusBadge type="admin" :value="row.adminStatus" />
        </template>
      </el-table-column>

      <el-table-column label="Tagged 端口" min-width="160">
        <template #default="{ row }">
          <div class="port-tags">
            <template v-if="row.taggedPorts.length">
              <span v-for="port in row.taggedPorts.slice(0, 2)" :key="port" class="port-tag">
                {{ formatPortName(port) }}
              </span>
              <span v-if="row.taggedPorts.length > 2" class="port-more">
                +{{ row.taggedPorts.length - 2 }}
              </span>
            </template>
            <span v-else class="empty-text">-</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="Untagged 端口" min-width="160">
        <template #default="{ row }">
          <div class="port-tags">
            <template v-if="row.untaggedPorts.length">
              <span v-for="port in row.untaggedPorts.slice(0, 2)" :key="port" class="port-tag">
                {{ formatPortName(port) }}
              </span>
              <span v-if="row.untaggedPorts.length > 2" class="port-more">
                +{{ row.untaggedPorts.length - 2 }}
              </span>
            </template>
            <span v-else class="empty-text">-</span>
          </div>
        </template>
      </el-table-column>

      <el-table-column label="操作" width="140" align="center" fixed="right">
        <template #default="{ row }">
          <div class="table-actions">
            <el-button link type="primary" size="small" @click="handleEdit(row)">
              编辑
            </el-button>
            <el-button link type="danger" size="small" @click="handleDelete(row)">
              删除
            </el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <div class="table-footer" v-if="!loading && vlans.length">
      <span class="total-info">共 {{ vlans.length }} 个 VLAN</span>
      <el-pagination
        v-model:current-page="currentPage"
        v-model:page-size="pageSize"
        :page-sizes="[10, 20, 50]"
        layout="sizes, prev, pager, next"
        :total="vlans.length"
        size="small"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import StatusBadge from './StatusBadge.vue'
import type { VlanItem } from '../../types/vlan'

interface Props {
  vlans: VlanItem[]
  loading: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'selection-change': [selection: VlanItem[]]
  'edit': [vlan: VlanItem]
  'delete': [vlan: VlanItem]
}>()

const tableRef = ref()
const currentPage = ref(1)
const pageSize = ref(20)

const handleSelectionChange = (selection: VlanItem[]) => {
  emit('selection-change', selection)
}

const handleEdit = (row: VlanItem) => {
  emit('edit', row)
}

const handleDelete = (row: VlanItem) => {
  emit('delete', row)
}

const formatPortName = (name: string) => {
  // 简化端口名称显示
  return name
    .replace('GigabitEthernet', 'GE')
    .replace('TenGigabitEthernet', '10GE')
    .replace('Ethernet', 'Eth')
}
</script>

<style lang="scss" scoped>
@import '../../styles/variables.scss';

.vlan-table {
  .vlan-id {
    font-family: 'SF Mono', Monaco, monospace;
    font-weight: $font-weight-semibold;
    color: $color-primary-light;
    font-size: $font-size-base;
  }

  .vlan-name {
    color: $text-primary;
    font-weight: $font-weight-medium;
  }

  .port-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
    align-items: center;

    .port-tag {
      padding: 2px 6px;
      background-color: $bg-elevated;
      border-radius: $radius-sm;
      font-size: $font-size-xs;
      color: $text-secondary;
      border: 1px solid $border-color;
    }

    .port-more {
      font-size: $font-size-xs;
      color: $text-tertiary;
    }

    .empty-text {
      color: $text-disabled;
      font-size: $font-size-sm;
    }
  }

  .table-actions {
    display: flex;
    gap: $spacing-sm;
    justify-content: center;
  }
}

.table-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: $spacing-lg $spacing-xl;
  border-top: 1px solid $border-color;

  .total-info {
    font-size: $font-size-sm;
    color: $text-secondary;
  }
}
</style>
