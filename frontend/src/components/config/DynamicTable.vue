<template>
  <div class="dynamic-table">
    <el-table :data="data" stripe border class="config-table">
      <el-table-column
        v-for="column in columns"
        :key="column.path"
        :prop="column.path"
        :label="column.label"
        min-width="120"
      >
        <template #default="{ row }">
          <el-tag
            v-if="column.type === 'boolean'"
            :type="row[column.path] ? 'success' : 'info'"
            size="small"
          >
            {{ row[column.path] ? '启用' : '禁用' }}
          </el-tag>
          <span v-else>{{ row[column.path] }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row, $index }">
          <el-button type="primary" size="small" link @click="$emit('edit', row, $index)">
            编辑
          </el-button>
          <el-button type="danger" size="small" link @click="$emit('delete', row, $index)">
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="table-actions">
      <el-button type="primary" class="add-button" @click="$emit('add')">
        新增配置项
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Field } from '../../api/crd'

defineProps<{
  columns: Field[]
  data: Record<string, any>[]
}>()

defineEmits<{
  add: []
  edit: [row: Record<string, any>, index: number]
  delete: [row: Record<string, any>, index: number]
}>()
</script>

<style scoped>
.dynamic-table {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.config-table {
  border-radius: 8px;
  overflow: hidden;
}

.table-actions {
  display: flex;
  justify-content: flex-end;
}

.add-button {
  align-self: flex-start;
}
</style>
