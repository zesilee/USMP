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
            {{ row[column.path] ? t('common.enabled') : t('common.disabled') }}
          </el-tag>
          <span v-else>{{ row[column.path] }}</span>
        </template>
      </el-table-column>
      <el-table-column :label="t('common.actions')" width="150" fixed="right">
        <template #default="{ row, $index }">
          <el-button type="primary" size="small" link @click="$emit('edit', row, $index)">
            {{ t('common.edit') }}
          </el-button>
          <el-button type="danger" size="small" link @click="$emit('delete', row, $index)">
            {{ t('common.delete') }}
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="table-actions">
      <el-button type="primary" class="add-button" @click="$emit('add')">
        {{ t('console.addConfigItem') }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Field } from '../../utils/crdSchemaParser'

const { t } = useI18n()

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
